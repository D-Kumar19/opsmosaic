package brief

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOnboard(t *testing.T) {
	var out bytes.Buffer
	p, err := Onboard(bufio.NewReader(strings.NewReader("\nDheeraj\n0\n1,2,2,9\n")), &out)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Dheeraj" || strings.Join(p.Topics, ",") != "docker,kubernetes,ai" {
		t.Fatalf("unexpected profile: %+v", p)
	}
	if !strings.Contains(out.String(), "Name") || !strings.Contains(out.String(), "Invalid") {
		t.Fatalf("missing validation: %s", out.String())
	}
}

func TestOnboardEOF(t *testing.T) {
	_, err := Onboard(bufio.NewReader(strings.NewReader("Dheeraj\n")), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "input ended") {
		t.Fatalf("expected useful EOF error, got %v", err)
	}
}

func TestProfileValidation(t *testing.T) {
	for _, p := range []Profile{
		{Version: 99, Name: "Name", Topics: []string{"aws"}},
		{Version: 1, Name: "", Topics: []string{"aws"}},
		{Version: 1, Name: "Name"},
		{Version: 1, Name: "Name", Topics: []string{"unknown"}},
		{Version: 1, Name: "Name", Topics: []string{"aws", "aws"}},
		{Version: 1, Name: "\x1b[31mName", Topics: []string{"aws"}},
		{Version: 1, Name: strings.Repeat("x", 61), Topics: []string{"aws"}},
	} {
		if err := p.Validate(); err == nil {
			t.Errorf("accepted invalid profile: %+v", p)
		}
	}
}

func TestStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := Profile{Version: 1, Name: "Dheeraj", Topics: []string{"aws", "ai"}}
	if err := writeJSON(path, want); err != nil {
		t.Fatal(err)
	}
	var got Profile
	if err := readJSON(path, &got); err != nil || got.Name != want.Name {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v, %v", info, err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := readJSON(path, &got); err == nil {
		t.Fatal("corrupt JSON must fail explicitly")
	}
}

func TestCleanText(t *testing.T) {
	input := "\x1b[31mHello\x1b[0m \x1b]8;;https://evil.test\aWorld\x1b]8;;\x1b\\\n&lt;b&gt;AWS&lt;/b&gt;\x00"
	if got := cleanText(input); got != "Hello World AWS" {
		t.Fatalf("unsafe/incorrect text: %q", got)
	}
}

func TestSafeURL(t *testing.T) {
	for _, url := range []string{"javascript:alert(1)", "file:///etc/passwd", "https://", "https://user:pass@example.com", "https://example.com/\x1b[31m"} {
		if safeURL(url) != "" {
			t.Errorf("accepted unsafe URL %q", url)
		}
	}
	if got := canonicalURL("https://example.com/article/?utm_source=test#anchor"); got != "https://example.com/article" {
		t.Fatalf("canonical URL: %q", got)
	}
}

func TestParseFeed(t *testing.T) {
	cases := []struct {
		name, body string
	}{
		{"RSS", `<rss><channel><item><title>&lt;b&gt;Launch&lt;/b&gt;</title><link>https://example.com/news</link><pubDate>Tue, 13 Oct 2026 12:00:00 GMT</pubDate></item><item><title>Unsafe</title><link>javascript:alert(1)</link></item></channel></rss>`},
		{"Atom", `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Launch</title><link rel="self" href="https://api.example.com"/><link rel="alternate" href="https://example.com/news"/><published>2026-10-13T12:00:00Z</published></entry></feed>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items, err := parseFeed([]byte(tc.body), Source{ID: "example", Name: "Example", Topic: "aws", Kind: "news"})
			if err != nil || len(items) != 1 {
				t.Fatalf("parse: %v, %v", items, err)
			}
			if items[0].Title != "Launch" || items[0].Published.IsZero() || items[0].URL != "https://example.com/news" {
				t.Fatalf("item: %+v", items[0])
			}
		})
	}
}

func TestParseFeedErrors(t *testing.T) {
	for _, body := range []string{"broken", "<rss>", "<html><body>Not a feed</body></html>"} {
		if _, err := parseFeed([]byte(body), Source{}); err == nil {
			t.Errorf("accepted malformed/unsupported feed %q", body)
		}
	}
}

func TestFeedHTTP(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"status", 503, "Unavailable"},
		{"oversized", 200, strings.Repeat("x", maxBody+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			_, err := (Source{ID: "test", URL: srv.URL}).Fetch(context.Background(), srv.Client())
			if err == nil {
				t.Fatal("expected fetch failure")
			}
		})
	}
}

func TestHackerNews(t *testing.T) {
	now := time.Now()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/topstories.json":
			fmt.Fprint(w, "[1,2,3,4]")
		default:
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/item/"), ".json")
			title := "AI model release"
			if id == "2" {
				title = "A recipe for cake"
			}
			data := map[string]interface{}{"id": id, "type": "story", "title": title, "url": "https://example.com/" + id, "time": now.Unix(), "dead": id == "3", "deleted": id == "4"}
			if err := json.NewEncoder(w).Encode(data); err != nil {
				t.Error(err)
			}
		}
	}))
	defer srv.Close()
	items, err := (Source{ID: "hn", Name: "HN", Topic: "ai", Kind: "news", Format: "hn", URL: srv.URL}).Fetch(context.Background(), srv.Client())
	if err != nil || len(items) != 1 || items[0].Title != "AI model release" {
		t.Fatalf("HN filtering: %+v, %v", items, err)
	}
}

func TestTipsCoverTopics(t *testing.T) {
	for _, topic := range Topics {
		found := false
		for _, tip := range Tips() {
			if tip.Topic == topic.Slug {
				found = true
				if tip.Kind != "tip" || safeURL(tip.URL) == "" || tip.Title == "" {
					t.Errorf("incomplete tip: %+v", tip)
				}
			}
		}
		if !found {
			t.Errorf("missing tip for %s", topic.Slug)
		}
	}
}

func TestSelectItems(t *testing.T) {
	now := time.Now()
	items := []Item{
		{ID: "aws1", Topic: "aws", URL: "https://a.test/1", Published: now},
		{ID: "aws2", Topic: "aws", URL: "https://a.test/2", Published: now.Add(-time.Hour)},
		{ID: "k8s1", Topic: "kubernetes", URL: "https://k.test/1"},
		{ID: "duplicate", Topic: "kubernetes", URL: "https://a.test/1?utm_source=x"},
		{ID: "ai1", Topic: "ai", URL: "https://ai.test/1"},
	}
	got := selectItems(items, []string{"aws", "kubernetes"}, nil, 3)
	if len(got) != 3 || got[0].ID != "aws1" || got[1].ID != "k8s1" || got[2].ID != "aws2" {
		t.Fatalf("unbalanced or duplicates: %+v", got)
	}
	got = selectItems(items, []string{"aws"}, []string{"aws1"}, 1)
	if len(got) != 1 || got[0].ID != "aws2" {
		t.Fatalf("did not prefer unseen: %+v", got)
	}
}

func TestHistoryBounded(t *testing.T) {
	history := make([]string, 100)
	for i := range history {
		history[i] = fmt.Sprint(i)
	}
	got := remember(history, []Item{{ID: "new"}, {ID: "99"}})
	if len(got) != 100 || got[len(got)-1] != "99" || got[len(got)-2] != "new" {
		t.Fatalf("history: %+v", got)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	return &App{
		Store:   Store{ConfigPath: filepath.Join(home, "config.json"), CachePath: filepath.Join(home, "cache.json")},
		Client:  &http.Client{Timeout: time.Second},
		Sources: []Source{},
		Now:     func() time.Time { return time.Date(2026, 10, 13, 21, 14, 0, 0, time.UTC) },
	}
}

func runApp(a *App, args []string, input string) (string, string, error) {
	var out, warnings bytes.Buffer
	err := a.Run(context.Background(), args, strings.NewReader(input), &out, &warnings)
	return out.String(), warnings.String(), err
}

func TestCLIOnboardingAndReturn(t *testing.T) {
	a := newTestApp(t)
	out, _, err := runApp(a, []string{"--tips"}, "Dheeraj\n2,3\n")
	if err != nil || !strings.Contains(out, "Dheeraj") || !strings.Contains(out, "TIP") {
		t.Fatalf("first run: %s, %v", out, err)
	}
	out, _, err = runApp(a, []string{"--tips"}, "")
	if err != nil || !strings.Contains(out, "Welcome back, Dheeraj!") || !strings.Contains(out, "21:14 UTC") {
		t.Fatalf("return: %s, %v", out, err)
	}
}

func TestCLIInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--count", "0"}, {"--count", "21"}, {"--tips", "--news"},
		{"--topic", "invalid"}, {"unknown"}, {"refresh", "--offline"}, {"--typo"},
		{"--topic", ""}, {"--topic", "aws,aws"}, {"config", "--tips"}, {"refresh", "--tips"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			a := newTestApp(t)
			_, _, err := runApp(a, args, "")
			if err == nil {
				t.Fatalf("accepted %v", args)
			}
			if _, err := os.Stat(a.Store.ConfigPath); !os.IsNotExist(err) {
				t.Fatal("invalid arguments should not create a profile")
			}
		})
	}
}

func TestEmptyFeedTTLAndFutureClock(t *testing.T) {
	a := newTestApp(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, "<rss><channel></channel></rss>")
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Topic: "aws", Kind: "news", URL: srv.URL}}
	if _, _, err := runApp(a, nil, "Dheeraj\n3\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, nil, ""); err != nil || requests.Load() != 1 {
		t.Fatalf("empty feed refetched within TTL: %v, %d", err, requests.Load())
	}
	old := a.Now()
	a.Now = func() time.Time { return old.Add(-time.Minute) }
	if _, _, err := runApp(a, nil, ""); err != nil || requests.Load() != 2 {
		t.Fatalf("future-dated cache was reused: %v, %d", err, requests.Load())
	}
}

func TestRefreshWithOneItem(t *testing.T) {
	a := newTestApp(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, "<rss><channel></channel></rss>")
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Topic: "aws", Kind: "news", URL: srv.URL}}
	out, _, err := runApp(a, []string{"refresh", "--count", "1"}, "Dheeraj\n3\n")
	if err != nil || requests.Load() != 1 || strings.Count(out, "TIP |") != 1 {
		t.Fatalf("refresh did not fetch: %s, %v, %d", out, err, requests.Load())
	}
}

func TestTopicsRotateAcrossBriefings(t *testing.T) {
	topics := []string{"aws", "kubernetes", "ai"}
	items := []Item{{ID: "a", Topic: "aws"}, {ID: "k", Topic: "kubernetes"}, {ID: "i", Topic: "ai"}}
	rotated := rotateTopics(topics, []string{"a", "k", "tip-ai-model-card"}, items)
	if strings.Join(rotated, ",") != "ai,aws,kubernetes" || strings.Join(topics, ",") != "aws,kubernetes,ai" {
		t.Fatalf("rotation altered preferences or order: %v, %v", topics, rotated)
	}
}

func TestNewsPartialFailure(t *testing.T) {
	a := newTestApp(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failed" {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `<rss><channel><item><title>Working source</title><link>https://example.com/good</link></item></channel></rss>`)
	}))
	defer srv.Close()
	a.Sources = []Source{
		{ID: "aws", Name: "AWS", Topic: "aws", Kind: "news", URL: srv.URL + "/good"},
		{ID: "docker", Name: "Docker", Topic: "docker", Kind: "news", URL: srv.URL + "/failed"},
	}
	out, warnings, err := runApp(a, []string{"--news"}, "Dheeraj\n1,3\n")
	if err != nil || !strings.Contains(out, "Working source") || !strings.Contains(warnings, "Docker: HTTP 503") {
		t.Fatalf("partial failure: %s, %s, %v", out, warnings, err)
	}
}

func TestFetchCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := (Source{URL: srv.URL}).Fetch(ctx, srv.Client())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
}

func TestHackerNewsErrors(t *testing.T) {
	for _, tc := range []struct {
		name, ids, item string
		code            int
	}{
		{"invalid IDs", "{}", "", 200},
		{"invalid item", "[1]", "{broken", 200},
		{"failed item", "[1]", "", 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/topstories.json" {
					fmt.Fprint(w, tc.ids)
					return
				}
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.item)
			}))
			defer srv.Close()
			_, err := (Source{Format: "hn", URL: srv.URL}).Fetch(context.Background(), srv.Client())
			if err == nil {
				t.Fatal("HN failures must be explicit")
			}
		})
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) {
	return 0, errors.New("output unavailable")
}

func TestRenderSafetyAndWriteError(t *testing.T) {
	item := Item{Kind: "news", Topic: "aws", Title: "\x1b[31mTitle\x1b[0m", Source: "\x1b]0;bad\aSource", URL: "javascript:alert(1)"}
	var out bytes.Buffer
	if err := render(&out, "Name", []string{"aws"}, []Item{item}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(out.String(), "\x1b\a") || strings.Contains(out.String(), "javascript:") {
		t.Fatalf("unsafe terminal rendering: %q", out.String())
	}
	if err := render(failedWriter{}, "Name", nil, nil, time.Now()); err == nil {
		t.Fatal("output errors must propagate")
	}
}

func TestStateErrorsAndNoFalseHistory(t *testing.T) {
	a := newTestApp(t)
	if err := os.WriteFile(a.Store.ConfigPath, []byte(`{"version":1,"name":"Name","topics":["aws"]} {}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"--tips"}, ""); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
	if _, _, err := runApp(a, []string{"config"}, "New name\n2\n"); err != nil {
		t.Fatal(err)
	}
	if err := a.Run(context.Background(), []string{"--tips"}, strings.NewReader(""), failedWriter{}, &bytes.Buffer{}); err == nil {
		t.Fatal("output failure must propagate")
	}
	if _, err := os.Stat(a.Store.CachePath); !os.IsNotExist(err) {
		t.Fatal("failed display should not update history")
	}
	if err := writeJSON(a.Store.CachePath, Cache{Version: 99, Sources: map[string]SourceCache{}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"--tips"}, ""); err == nil {
		t.Fatal("unknown cache schema was accepted")
	}
}

func TestDefaultStoreOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("OPSMOSAIC_HOME", home)
	store, err := DefaultStore()
	if err != nil || store.ConfigPath != filepath.Join(home, "config.json") || store.CachePath != filepath.Join(home, "cache.json") {
		t.Fatalf("override: %+v, %v", store, err)
	}
	app, err := NewApp()
	if err != nil || app.Store != store || len(app.Sources) != len(Topics) || app.Client.Timeout != 5*time.Second {
		t.Fatalf("default app: %+v, %v", app, err)
	}
}

func TestCISAPublicationDate(t *testing.T) {
	body := `<rss><channel><item><title>Advisory</title><link>https://example.com/advisory</link><pubDate>Tue, 13 Oct 26 12:00:00 +0000</pubDate></item></channel></rss>`
	items, err := parseFeed([]byte(body), Source{Topic: "security", Kind: "news"})
	want := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	if err != nil || len(items) != 1 || !items[0].Published.Equal(want) {
		t.Fatalf("CISA date was lost: %+v, %v", items, err)
	}
}
func TestCLIHelpWithoutProfile(t *testing.T) {
	a := newTestApp(t)
	out, _, err := runApp(a, []string{"--help"}, "")
	if err != nil || !strings.Contains(out, "OpsMosaic") {
		t.Fatalf("help: %s, %v", out, err)
	}
}

func TestConfigPreservesProfileOnEOF(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--tips"}, "Dheeraj\n3\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"config"}, ""); err == nil {
		t.Fatal("expected EOF")
	}
	var p Profile
	if err := readJSON(a.Store.ConfigPath, &p); err != nil || p.Name != "Dheeraj" {
		t.Fatalf("profile was lost: %+v, %v", p, err)
	}
}

func TestCLICacheLifecycle(t *testing.T) {
	a := newTestApp(t)
	var requests atomic.Int32
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if fail.Load() {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `<rss><channel><item><title>AWS launch</title><link>https://example.com/news</link></item></channel></rss>`)
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Name: "AWS", Topic: "aws", Kind: "news", URL: srv.URL}}
	out, _, err := runApp(a, []string{"--news"}, "Dheeraj\n3\n")
	if err != nil || !strings.Contains(out, "AWS launch") || requests.Load() != 1 {
		t.Fatalf("fetch: %s, %v, %d", out, err, requests.Load())
	}
	out, _, err = runApp(a, []string{"--news"}, "")
	if err != nil || requests.Load() != 1 || !strings.Contains(out, "cached") {
		t.Fatalf("cache: %s, %v, %d", out, err, requests.Load())
	}
	fail.Store(true)
	out, warnings, err := runApp(a, []string{"refresh", "--news"}, "")
	if err != nil || !strings.Contains(out, "AWS launch") || !strings.Contains(warnings, "503") || !strings.Contains(out, "cached") {
		t.Fatalf("fallback: %s, %s, %v", out, warnings, err)
	}
	before := requests.Load()
	out, _, err = runApp(a, []string{"--offline", "--news"}, "")
	if err != nil || requests.Load() != before || !strings.Contains(out, "AWS launch") {
		t.Fatalf("offline: %s, %v", out, err)
	}
	if _, _, err = runApp(a, []string{"--tips"}, ""); err != nil || requests.Load() != before {
		t.Fatalf("tips performed requests: %v", err)
	}
}

func TestCLICacheTTL(t *testing.T) {
	a := newTestApp(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, `<rss><channel><item><title>News</title><link>https://example.com/n</link></item></channel></rss>`)
	}))
	defer srv.Close()
	a.Sources = []Source{{ID: "aws", Topic: "aws", Kind: "news", URL: srv.URL}}
	if _, _, err := runApp(a, []string{"--news"}, "Dheeraj\n3\n"); err != nil {
		t.Fatal(err)
	}
	old := a.Now()
	a.Now = func() time.Time { return old.Add(6 * time.Hour) }
	if _, _, err := runApp(a, []string{"--news"}, ""); err != nil || requests.Load() != 2 {
		t.Fatalf("TTL did not expire: %v, %d", err, requests.Load())
	}
}

func TestCLIEmptyAndCorruptCache(t *testing.T) {
	a := newTestApp(t)
	out, warnings, err := runApp(a, []string{"--offline"}, "Dheeraj\n3\n")
	if err != nil || !strings.Contains(out, "TIP") || !strings.Contains(warnings, "No news") {
		t.Fatalf("empty default: %s, %s, %v", out, warnings, err)
	}
	if _, _, err = runApp(a, []string{"--offline", "--news"}, ""); err == nil {
		t.Fatal("empty news-only should fail")
	}
	if err := os.WriteFile(a.Store.CachePath, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err = runApp(a, []string{"--offline"}, ""); err == nil {
		t.Fatal("corrupt cache should fail explicitly")
	}
}

func TestCLITopicOverrideAndCount(t *testing.T) {
	a := newTestApp(t)
	out, _, err := runApp(a, []string{"--tips", "--topic", "aws,helm", "--count", "2"}, "Dheeraj\n1\n")
	if err != nil || strings.Count(out, "TIP |") != 2 || !strings.Contains(out, "Helm") || strings.Contains(out, "TIP | Docker") {
		t.Fatalf("override/count: %s, %v", out, err)
	}
	var p Profile
	if err := readJSON(a.Store.ConfigPath, &p); err != nil || strings.Join(p.Topics, ",") != "docker" {
		t.Fatalf("override changed preferences: %+v, %v", p, err)
	}
}

func TestRandomSelectionKindsAndHistory(t *testing.T) {
	items := []Item{
		{ID: "tip", Kind: "tip", Topic: "aws"},
		{ID: "news", Kind: "news", Topic: "aws", URL: "https://example.com/news"},
		{ID: "release", Kind: "release", Topic: "helm", URL: "https://example.com/release"},
		{ID: "other", Kind: "news", Topic: "ai"},
	}
	kinds := map[string]bool{}
	for index := 0; index < 3; index++ {
		calls := 0
		got := selectRandomItem(items, []string{"aws", "helm"}, nil, func(n int) int {
			calls++
			if calls == 1 {
				if n != 3 {
					t.Fatalf("expected three equally selectable kinds, got %d", n)
				}
				return index
			}
			return 0
		})
		if len(got) != 1 || got[0].ID == "other" {
			t.Fatalf("incorrect random selection: %+v", got)
		}
		kinds[got[0].Kind] = true
	}
	if len(kinds) != 3 {
		t.Fatalf("not every kind can be selected: %v", kinds)
	}
	got := selectRandomItem(items, []string{"aws"}, []string{"tip"}, func(n int) int { return 0 })
	if len(got) != 1 || got[0].ID != "news" {
		t.Fatalf("did not prefer unseen content: %+v", got)
	}
	got = selectRandomItem(items, []string{"helm"}, []string{"release"}, func(n int) int { return 0 })
	if len(got) != 1 || got[0].ID != "release" {
		t.Fatalf("all-seen pool should allow repeats: %+v", got)
	}
	if got := selectRandomItem(items, []string{"crossplane"}, nil, nil); len(got) != 0 {
		t.Fatalf("unselected topic leaked: %+v", got)
	}
}

func TestRandomSelectionWithinKindAndDeduplication(t *testing.T) {
	items := []Item{
		{ID: "first", Kind: "news", Topic: "aws", URL: "https://example.com/1"},
		{ID: "duplicate", Kind: "news", Topic: "aws", URL: "https://example.com/1?utm_source=test"},
		{ID: "second", Kind: "news", Topic: "aws", URL: "https://example.com/2"},
	}
	for index, want := range []string{"first", "second"} {
		calls := 0
		got := selectRandomItem(items, []string{"aws"}, nil, func(n int) int {
			calls++
			if calls == 1 {
				return 0
			}
			if n != 2 {
				t.Fatalf("duplicate article biased random pool: %d", n)
			}
			return index
		})
		if len(got) != 1 || got[0].ID != want {
			t.Fatalf("expected %s, got %+v", want, got)
		}
	}
}

func TestCLISingleRandomDefaultAndExplicitOptions(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		kind      string
		topic     string
		kindIndex int
	}{
		{"default tip", nil, "tip", "aws", 2},
		{"default news", nil, "news", "aws", 0},
		{"default release", nil, "release", "helm", 1},
		{"tips filter", []string{"--tips"}, "tip", "aws", 0},
		{"news filter", []string{"--news"}, "release", "helm", 1},
		{"topic filter", []string{"--topic", "helm"}, "tip", "helm", 1},
		{"explicit one", []string{"--count", "1"}, "news", "aws", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t)
			a.Sources = []Source{
				{ID: "aws", Topic: "aws", Kind: "news"},
				{ID: "helm", Topic: "helm", Kind: "release"},
			}
			news := Item{ID: "news", Kind: "news", Topic: "aws", Title: "AWS news", URL: "https://example.com/news"}
			release := Item{ID: "release", Kind: "release", Topic: "helm", Title: "Helm release", URL: "https://example.com/release"}
			cache := Cache{Version: 1, Sources: map[string]SourceCache{
				"aws":  {Fetched: a.Now(), Items: []Item{news}},
				"helm": {Fetched: a.Now(), Items: []Item{release}},
			}}
			if err := writeJSON(a.Store.CachePath, cache); err != nil {
				t.Fatal(err)
			}
			calls := 0
			a.randomIndex = func(n int) int {
				calls++
				if calls == 1 {
					return tc.kindIndex
				}
				return 0
			}
			out, warnings, err := runApp(a, tc.args, "Dheeraj\n3,5\n")
			if err != nil || warnings != "" || !strings.Contains(out, strings.ToUpper(tc.kind)+" | "+topicName(tc.topic)+"\n") || !strings.Contains(out, "Showing 1 items") {
				t.Fatalf("single random output: %s, %s, %v", out, warnings, err)
			}
			total := strings.Count(out, "TIP |") + strings.Count(out, "NEWS |") + strings.Count(out, "RELEASE |")
			if total != 1 {
				t.Fatalf("expected exactly one card, got %d", total)
			}
		})
	}
}

func TestCLIExplicitMixedCount(t *testing.T) {
	a := newTestApp(t)
	a.Sources = []Source{{ID: "aws", Topic: "aws", Kind: "news"}}
	cache := Cache{Version: 1, Sources: map[string]SourceCache{
		"aws": {Fetched: a.Now(), Items: []Item{
			{ID: "one", Kind: "news", Topic: "aws", URL: "https://example.com/1"},
			{ID: "two", Kind: "news", Topic: "aws", URL: "https://example.com/2"},
		}},
	}}
	if err := writeJSON(a.Store.CachePath, cache); err != nil {
		t.Fatal(err)
	}
	out, _, err := runApp(a, []string{"--count", "3"}, "Dheeraj\n3\n")
	if err != nil || strings.Count(out, "TIP |") != 1 || strings.Count(out, "NEWS |") != 2 || !strings.Contains(out, "Showing 3 items") {
		t.Fatalf("explicit count no longer honored: %s, %v", out, err)
	}
}
