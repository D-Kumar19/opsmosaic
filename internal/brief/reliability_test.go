package brief

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestKindSelectionBeforeUnseenPreference(t *testing.T) {
	items := []Item{
		{ID: "new-news", Kind: "news", Topic: "aws"},
		{ID: "seen-tip", Kind: "tip", Topic: "aws"},
	}
	calls := 0
	got := selectRandomItem(items, []string{"aws"}, []string{"seen-tip"}, func(n int) int {
		calls++
		if calls == 1 {
			if n != 2 {
				t.Fatalf("seen tips were excluded from kind choice: %d kinds", n)
			}
			return 1
		}
		return 0
	})
	if len(got) != 1 || got[0].ID != "seen-tip" {
		t.Fatalf("tip must remain eligible: %+v", got)
	}
	items = append(items, Item{ID: "new-tip", Kind: "tip", Topic: "aws"})
	got = selectRandomItem(items, []string{"aws"}, []string{"seen-tip"}, func(n int) int { return n - 1 })
	if len(got) != 1 || got[0].ID != "new-tip" {
		t.Fatalf("must prefer unseen within selected kind: %+v", got)
	}
}

func TestHNPartialAndTotalFailure(t *testing.T) {
	for _, failure := range []string{"http", "malformed", "timeout"} {
		t.Run(failure, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/topstories.json":
					fmt.Fprint(w, "[1,2,3]")
				case "/item/1.json":
					fmt.Fprint(w, `{"type":"story","title":"AI release","url":"https://example.com/new"}`)
				case "/item/2.json":
					fmt.Fprint(w, `{"type":"story","title":"Cake recipe","url":"https://example.com/cake"}`)
				default:
					switch failure {
					case "http":
						w.WriteHeader(503)
					case "malformed":
						fmt.Fprint(w, "{broken")
					case "timeout":
						<-r.Context().Done()
					}
				}
			}))
			defer srv.Close()
			client := srv.Client()
			client.Timeout = 100 * time.Millisecond
			source := Source{ID: "ai", Topic: "ai", Format: "hn", URL: srv.URL}
			items, err := source.Fetch(context.Background(), client)
			var partial *PartialFetchError
			if len(items) != 1 || !errors.As(err, &partial) || partial.Failed != 1 || partial.Total != 3 {
				t.Fatalf("lost partial results: %+v, %v", items, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := source.Fetch(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
		})
	}
}

func TestPartialCacheMergeRetryAndRecovery(t *testing.T) {
	a := newTestApp(t)
	now := a.Now()
	var phase atomic.Int32
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/topstories.json" {
			requests.Add(1)
			fmt.Fprint(w, "[1,2]")
			return
		}
		if phase.Load() == 2 || (phase.Load() == 0 && r.URL.Path == "/item/2.json") {
			w.WriteHeader(503)
			return
		}
		link := "https://example.com/new"
		if r.URL.Path == "/item/2.json" {
			link = "https://example.com/recovered"
		}
		fmt.Fprintf(w, `{"type":"story","title":"AI news","url":%q}`, link)
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "ai", Name: "HN", Topic: "ai", Kind: "news", Format: "hn", URL: srv.URL}}
	oldTime := now.Add(-48 * time.Hour)
	old := []Item{
		{ID: "https://example.com/new", URL: "https://example.com/new?utm_source=old", Kind: "news", Topic: "ai"},
		{ID: "old", URL: "https://example.com/old", Kind: "news", Topic: "ai"},
	}
	for i := 0; i < 110; i++ {
		old = append(old, Item{ID: fmt.Sprint(i), URL: fmt.Sprintf("https://example.com/%d", i), Kind: "news", Topic: "ai"})
	}
	cache := Cache{Version: 1, Sources: map[string]SourceCache{"ai": {Fetched: oldTime, Items: old}}}
	items, notes := a.news(context.Background(), &cache, []string{"ai"}, now, false, false)
	entry := cache.Sources["ai"]
	if len(items) != 100 || len(notes) != 1 || !strings.Contains(notes[0], "partial") || !entry.RetryAfter.Equal(now.Add(15*time.Minute)) {
		t.Fatalf("partial merge: items=%d cache=%+v notes=%v", len(items), entry, notes)
	}
	if items[0].Cached || !items[0].FetchedAt.Equal(now) || !items[1].Cached || !items[1].FetchedAt.Equal(oldTime) {
		t.Fatalf("merged items need accurate freshness: %+v", items[:2])
	}
	for _, item := range items[1:] {
		if canonicalURL(item.URL) == "https://example.com/new" {
			t.Fatal("duplicate survived partial merge")
		}
	}
	a.news(context.Background(), &cache, []string{"ai"}, now.Add(15*time.Minute-time.Nanosecond), false, false)
	if requests.Load() != 1 {
		t.Fatal("partial source refetched too soon")
	}
	phase.Store(1)
	items, notes = a.news(context.Background(), &cache, []string{"ai"}, now.Add(15*time.Minute), false, false)
	if requests.Load() != 2 || len(items) != 2 || len(notes) != 0 || !cache.Sources["ai"].RetryAfter.IsZero() {
		t.Fatalf("retry/recovery: %d, %+v, %v", requests.Load(), items, notes)
	}
	before := cache.Sources["ai"]
	phase.Store(2)
	items, notes = a.news(context.Background(), &cache, []string{"ai"}, now.Add(time.Hour), true, false)
	entry = cache.Sources["ai"]
	if len(items) != 2 || len(notes) != 1 || !entry.Fetched.Equal(before.Fetched) || entry.Items[0] != before.Items[0] {
		t.Fatalf("total failure overwrote cache: %+v, %v", entry, notes)
	}
}

func TestSelectedCacheFreshness(t *testing.T) {
	for _, tc := range []struct {
		name    string
		age     time.Duration
		unknown bool
		label   string
		stale   bool
	}{
		{"fresh", time.Hour, false, "cached 1h ago", false},
		{"before TTL", 6*time.Hour - time.Nanosecond, false, "cached 5h ago", false},
		{"TTL", 6 * time.Hour, false, "stale", true},
		{"old", 5 * 24 * time.Hour, false, "cached 5d ago", true},
		{"future", -time.Hour, false, "cache age unknown", true},
		{"missing", 0, true, "cache age unknown", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t)
			stamp := a.Now().Add(-tc.age)
			if tc.unknown {
				stamp = time.Time{}
			}
			a.Sources = []Source{{ID: "aws", Topic: "aws", Kind: "news"}}
			items := []Item{
				{ID: "one", Kind: "news", Topic: "aws", Title: "First article", URL: "https://example.com/one"},
				{ID: "two", Kind: "news", Topic: "aws", Title: "Second article", URL: "https://example.com/two"},
			}
			cache := Cache{Version: 1, Sources: map[string]SourceCache{"aws": {Fetched: stamp, Items: items}}}
			if err := writeJSON(a.Store.CachePath, cache); err != nil {
				t.Fatal(err)
			}
			out, warnings, err := runApp(a, []string{"--offline", "--news", "--count", "2"}, "Alex\n3\n")
			if err != nil || !strings.Contains(out, tc.label) {
				t.Fatalf("freshness: %s, %v", out, err)
			}
			hints := strings.Count(warnings, "opsmosaic refresh")
			if tc.stale && hints != 1 || !tc.stale && hints != 0 {
				t.Fatalf("expected one hint only for selected stale items: %s", warnings)
			}
			out, warnings, err = runApp(a, []string{"--tips"}, "")
			if err != nil || strings.Contains(warnings, "opsmosaic refresh") || strings.Contains(out, "stale") {
				t.Fatalf("unselected stale cache should not emit hint: %s %s %v", out, warnings, err)
			}
		})
	}
}

func TestBusyLockSnapshot(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--tips"}, "Alex\n3\n"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(a.Store.CachePath)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(503)
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Topic: "aws", URL: srv.URL}}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	start := time.Now()
	out, warnings, err := runApp(a, []string{"refresh"}, "")
	elapsed := time.Since(start)
	if err != nil || strings.Count(out, "TIP |") != 1 || !strings.Contains(warnings, "read-only snapshot") || requests.Load() != 0 {
		t.Fatalf("busy lock fallback: %s %s %v requests=%d", out, warnings, err, requests.Load())
	}
	if elapsed < 250*time.Millisecond || elapsed > time.Second {
		t.Fatalf("unexpected wait: %s", elapsed)
	}
	after, err := os.ReadFile(a.Store.CachePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("busy reader modified cache: %v", err)
	}
	if _, _, err := runApp(a, []string{"config"}, "New\n1\n"); err == nil {
		t.Fatal("busy config must fail instead of pretending to save")
	}
}

func TestStateProcessHelper(t *testing.T) {
	if os.Getenv("OPSMOSAIC_TEST_STATE_HELPER") != "1" {
		return
	}
	store, err := DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("OPSMOSAIC_TEST_APP") == "1" {
		a := &App{Store: store, Now: time.Now}
		_, warnings, err := runApp(a, []string{"--tips"}, "")
		if err != nil {
			t.Fatal(err)
		}
		fmt.Print(warnings)
		return
	}
	ctx := context.Background()
	lock, err := store.lock(ctx, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("OPSMOSAIC_TEST_HOLD_LOCK") == "1" {
		fmt.Println("locked")
		time.Sleep(time.Minute)
		return
	}
	defer lock.Close()
	cache, err := store.cache()
	if err != nil {
		t.Fatal(err)
	}
	cache.GreetingVisits++
	cache.Recent = remember(cache.Recent, []Item{{ID: os.Getenv("OPSMOSAIC_TEST_ITEM")}})
	// Ensure the transaction spans enough time for competing processes to overlap.
	time.Sleep(20 * time.Millisecond)
	if err := writeJSON(store.CachePath, cache); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentCLIProcesses(t *testing.T) {
	a := newTestApp(t)
	if err := writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"aws"}}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var commands []*exec.Cmd
	var outputs []*bytes.Buffer
	for i := 0; i < 4; i++ {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStateProcessHelper$")
		cmd.Env = append(os.Environ(), "OPSMOSAIC_TEST_STATE_HELPER=1", "OPSMOSAIC_TEST_APP=1", "OPSMOSAIC_HOME="+filepath.Dir(a.Store.CachePath))
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands, outputs = append(commands, cmd), append(outputs, &output)
	}
	var readOnly uint64
	for i, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("CLI subprocess: %v %s", err, outputs[i].String())
		}
		if strings.Contains(outputs[i].String(), "Warning: state is busy; showing a read-only snapshot") {
			readOnly++
		}
	}
	got, err := a.Store.cache()
	if err != nil || got.GreetingVisits != 4-readOnly || readOnly == 4 {
		t.Fatalf("CLI lost writable state updates: %+v %v; read-only subprocesses=%d", got, err, readOnly)
	}
}

func TestLockCancellationAndMissingProfile(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--help"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.Store.ConfigPath + ".lock"); !os.IsNotExist(err) {
		t.Fatal("help should not create a lock")
	}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := a.Store.lock(ctx, time.Second); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock ignored cancellation: %v", err)
	}
	out, _, err := runApp(a, []string{"--offline"}, "Alex\n3\n")
	if err == nil || !strings.Contains(err.Error(), "setup is in progress") || strings.Contains(out, "What should I call you") {
		t.Fatalf("busy first run must not prompt or write: %s %v", out, err)
	}
	if _, err := os.Stat(a.Store.ConfigPath); !os.IsNotExist(err) {
		t.Fatal("busy first run created profile")
	}
}

func TestPartialWithoutMatchingStoriesAndExplicitRetry(t *testing.T) {
	a := newTestApp(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			requests.Add(1)
			fmt.Fprint(w, "[1,2]")
		case "/item/1.json":
			fmt.Fprint(w, `{"type":"story","title":"Cake recipe","url":"https://example.com/cake"}`)
		default:
			w.WriteHeader(503)
		}
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "ai", Topic: "ai", Format: "hn", URL: srv.URL}}
	cache := Cache{Version: 1, Sources: map[string]SourceCache{}}
	now := a.Now()
	items, notes := a.news(context.Background(), &cache, []string{"ai"}, now, false, false)
	if len(items) != 0 || len(notes) != 1 || cache.Sources["ai"].RetryAfter.IsZero() {
		t.Fatalf("empty partial should warn and throttle retries: %+v %v", cache, notes)
	}
	a.news(context.Background(), &cache, []string{"ai"}, now.Add(time.Minute), false, false)
	if requests.Load() != 1 {
		t.Fatal("empty partial refetched before deadline")
	}
	a.news(context.Background(), &cache, []string{"ai"}, now.Add(time.Minute), true, false)
	if requests.Load() != 2 {
		t.Fatal("explicit refresh must bypass partial retry deadline")
	}
}

func TestFailedRefreshShowsOldAge(t *testing.T) {
	a := newTestApp(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Name: "AWS", Topic: "aws", Kind: "news", URL: srv.URL}}
	cache := Cache{Version: 1, Sources: map[string]SourceCache{
		"aws": {Fetched: a.Now().Add(-48 * time.Hour), Items: []Item{{ID: "old", Kind: "news", Topic: "aws", Title: "Old article", URL: "https://example.com/old"}}},
	}}
	if err := writeJSON(a.Store.CachePath, cache); err != nil {
		t.Fatal(err)
	}
	out, warnings, err := runApp(a, []string{"refresh", "--news"}, "Alex\n3\n")
	if err != nil || !strings.Contains(out, "cached 2d ago | stale") || !strings.Contains(warnings, "HTTP 503") || strings.Count(warnings, "opsmosaic refresh") != 1 {
		t.Fatalf("failed refresh hid stale age: %s %s %v", out, warnings, err)
	}
	got, err := a.Store.cache()
	if err != nil || !got.Sources["aws"].Fetched.Equal(cache.Sources["aws"].Fetched) {
		t.Fatalf("failed fetch changed age: %+v %v", got, err)
	}
}

func TestFreshnessAgeFormatting(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		age  time.Duration
		want string
	}{
		{0, "cached just now"},
		{30 * time.Minute, "cached 30m ago"},
		{2 * time.Hour, "cached 2h ago"},
	} {
		label, stale := cacheFreshness(Item{Cached: true, FetchedAt: now.Add(-tc.age)}, now)
		if stale || label != tc.want {
			t.Fatalf("age %s: %q stale=%v", tc.age, label, stale)
		}
	}
	label, stale := cacheFreshness(Item{FetchedAt: now}, now)
	if stale || label != "" {
		t.Fatalf("live items should not be labeled cached: %q, %v", label, stale)
	}
}

func TestConcurrentProcessesAndInterruptedLock(t *testing.T) {
	a := newTestApp(t)
	cache := Cache{Version: 1, Sources: map[string]SourceCache{"kept": {Fetched: a.Now(), Items: []Item{{ID: "retained"}}}}}
	if err := writeJSON(a.Store.CachePath, cache); err != nil {
		t.Fatal(err)
	}
	home := filepath.Dir(a.Store.CachePath)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var commands []*exec.Cmd
	var outputs []*bytes.Buffer
	for i := 0; i < 4; i++ {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStateProcessHelper$")
		cmd.Env = append(os.Environ(), "OPSMOSAIC_TEST_STATE_HELPER=1", "OPSMOSAIC_HOME="+home, fmt.Sprintf("OPSMOSAIC_TEST_ITEM=%d", i))
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands, outputs = append(commands, cmd), append(outputs, &output)
	}
	for i, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("child: %v %s", err, outputs[i].String())
		}
	}
	got, err := a.Store.cache()
	if err != nil || got.GreetingVisits != 4 || len(got.Recent) != 4 || len(got.Sources["kept"].Items) != 1 {
		t.Fatalf("lost cross-process updates: %+v %v", got, err)
	}
	// The kernel must release the persistent-file lock even after abrupt exit.
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStateProcessHelper$")
	cmd.Env = append(os.Environ(), "OPSMOSAIC_TEST_STATE_HELPER=1", "OPSMOSAIC_TEST_HOLD_LOCK=1", "OPSMOSAIC_HOME="+home)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	var message string
	if _, err := fmt.Fscanln(pipe, &message); err != nil || message != "locked" {
		t.Fatalf("child readiness: %q %v", message, err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	lock, err := a.Store.lock(ctx, time.Second)
	if err != nil {
		t.Fatalf("interrupted process left stale lock: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	got, err = a.Store.cache()
	if err != nil || got.GreetingVisits != 4 {
		t.Fatalf("interruption corrupted JSON: %+v %v", got, err)
	}
}

func TestHNDeadlineKeepsCompletedStories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			fmt.Fprint(w, "[1,2]")
		case "/item/1.json":
			fmt.Fprint(w, `{"type":"story","title":"AI release","url":"https://example.com/completed"}`)
		default:
			<-r.Context().Done()
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	items, err := (Source{Format: "hn", URL: srv.URL, Topic: "ai"}).Fetch(ctx, srv.Client())
	var partial *PartialFetchError
	if len(items) != 1 || !errors.As(err, &partial) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("source deadline discarded completed stories: %+v %v", items, err)
	}
}

func TestLockErrorsAreExplicit(t *testing.T) {
	a := newTestApp(t)
	blocker := filepath.Join(filepath.Dir(a.Store.ConfigPath), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Store.ConfigPath = filepath.Join(blocker, "config.json")
	if _, err := a.Store.lock(context.Background(), time.Second); err == nil || !strings.Contains(err.Error(), "lock directory") {
		t.Fatalf("lock setup error was hidden: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.Store.lock(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled lock request did work: %v", err)
	}
}

func TestCancelledBriefingDoesNotSaveCache(t *testing.T) {
	a := newTestApp(t)
	if err := writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"aws"}}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Topic: "aws", URL: srv.URL}}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	var out, warnings bytes.Buffer
	err := a.Run(ctx, []string{"--news"}, strings.NewReader(""), &out, &warnings)
	if !errors.Is(err, context.DeadlineExceeded) || out.Len() != 0 {
		t.Fatalf("cancellation was not propagated: %s %v", out.String(), err)
	}
	if _, err := os.Stat(a.Store.CachePath); !os.IsNotExist(err) {
		t.Fatal("cancelled briefing saved cache")
	}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("cancelled run leaked lock: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
}
