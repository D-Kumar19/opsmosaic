package brief

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSavedBriefingReplayAndBookmarks(t *testing.T) {
	a := newTestApp(t)
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	if _, _, err := runApp(a, []string{"--tips", "--count", "3"}, "Alex\n1,2,3\n"); err != nil {
		t.Fatal(err)
	}
	state, err := a.Store.saved()
	if err != nil || state.Last == nil || len(state.Last.Items) != 3 {
		t.Fatalf("last briefing not saved: %+v %v", state, err)
	}
	info, _ := os.Stat(a.Store.savedPath())
	if info.Mode().Perm() != 0600 {
		t.Fatal("saved state is not private")
	}
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err == nil {
		t.Fatal("multi-item add requires a number")
	}
	for _, number := range []string{"0", "4", "bad", "-1"} {
		if _, _, err := runApp(a, []string{"bookmarks", "add", number}, ""); err == nil {
			t.Fatalf("accepted item %s", number)
		}
	}
	for i := 0; i < 2; i++ {
		if _, _, err := runApp(a, []string{"bookmarks", "add", "2"}, ""); err != nil {
			t.Fatal(err)
		}
	}
	state, _ = a.Store.saved()
	if len(state.Bookmarks) != 1 || state.Bookmarks[0].Items[0].ID != state.Last.Items[1].ID {
		t.Fatal("bookmark indexing or duplicate handling incorrect")
	}
	cacheBefore, _ := os.ReadFile(a.Store.CachePath)
	savedBefore, _ := os.ReadFile(a.Store.savedPath())
	// Replay and saved bookmarks do not depend on feeds, cache or profile.
	os.Remove(a.Store.ConfigPath)
	os.Remove(a.Store.CachePath)
	os.WriteFile(a.Store.feedsPath(), []byte("corrupt"), 0600)
	for _, args := range [][]string{
		{"replay"}, {"replay", "--plain", "--detailed"}, {"bookmarks", "show", state.Bookmarks[0].ID}, {"bookmarks", "list"},
	} {
		out, warnings, err := runApp(a, args, "")
		if err != nil || warnings != "" || strings.Contains(out, "Good ") || strings.Contains(out, "Alex") ||
			!strings.Contains(strings.ToLower(out), "captured") {
			t.Fatalf("saved display: %s %s %v", out, warnings, err)
		}
		if args[0] == "replay" && strings.Count(out, "TIP |") != 3 {
			t.Fatal("replay lost items")
		}
	}
	after, _ := os.ReadFile(a.Store.savedPath())
	if string(after) != string(savedBefore) {
		t.Fatal("saved display modified state")
	}
	if _, err := os.Stat(a.Store.CachePath); !os.IsNotExist(err) || len(cacheBefore) == 0 {
		t.Fatal("replay recreated cache")
	}
	if _, _, err := runApp(a, []string{"bookmarks", "remove", state.Bookmarks[0].ID}, ""); err != nil {
		t.Fatal(err)
	}
	out, _, err := runApp(a, []string{"bookmarks", "list"}, "")
	if err != nil || !strings.Contains(out, "None saved.") {
		t.Fatal("bookmark removal/list failed")
	}
}

func TestSavedSingleItemDuplicatesAndFullLimit(t *testing.T) {
	a := newTestApp(t)
	runApp(a, []string{"--tips"}, "Alex\n1\n")
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err != nil {
		t.Fatal(err)
	}
	state, _ := a.Store.saved()
	for i := 1; i < 100; i++ {
		item := state.Last.Items[0]
		item.ID = fmt.Sprint(i)
		item.URL = fmt.Sprintf("https://example.org/%d", i)
		state.Bookmarks = append(state.Bookmarks, Bookmark{ID: bookmarkID(item),
			SavedBriefing: SavedBriefing{CapturedAt: a.Now(), Items: []Item{item}}})
	}
	writeJSON(a.Store.savedPath(), state)
	// Duplicates remain usable even at the limit.
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err != nil {
		t.Fatal(err)
	}
	state.Last.Items[0].URL = "https://example.org/new"
	state.Last.Items[0].ID = "new-tip"
	writeJSON(a.Store.savedPath(), state)
	before, _ := os.ReadFile(a.Store.savedPath())
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err == nil {
		t.Fatal("bookmark limit did not reject addition")
	}
	after, _ := os.ReadFile(a.Store.savedPath())
	if string(before) != string(after) {
		t.Fatal("full limit silently evicted/mutated entries")
	}
}

func TestSavedMissingCorruptAndArguments(t *testing.T) {
	a := newTestApp(t)
	for _, args := range [][]string{
		{"replay"}, {"replay", "extra"}, {"replay", "--offline"}, {"replay", "--color=invalid"},
		{"bookmarks"}, {"bookmarks", "unknown"}, {"bookmarks", "add"}, {"bookmarks", "add", "1", "2"},
		{"bookmarks", "list", "extra"}, {"bookmarks", "show"}, {"bookmarks", "show", "missing"},
		{"bookmarks", "remove"}, {"bookmarks", "remove", "missing"}, {"bookmarks", "remove", "id", "extra"},
	} {
		if _, _, err := runApp(a, args, ""); err == nil {
			t.Fatalf("accepted unavailable/invalid command: %v", args)
		}
	}
	for _, data := range []string{`bad`, `{}`, `{"version":1,"bookmarks":null}`, `{"version":2,"bookmarks":[]}`, `{"version":1,"bookmarks":[],"last":{"items":[]}}`} {
		os.WriteFile(a.Store.savedPath(), []byte(data), 0600)
		if _, _, err := runApp(a, []string{"bookmarks", "list"}, ""); err == nil {
			t.Fatal("invalid saved state hidden")
		}
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		if err == nil || !strings.Contains(out, "saved_invalid") {
			t.Fatal("doctor missed corrupt saved state")
		}
	}
}

func TestSavedConcurrentEditsAndBusyReplay(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--tips", "--count", "3"}, "Alex\n1,2,3\n"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	type result struct {
		number int
		err    error
	}
	results := make(chan result, 3)
	start := make(chan struct{})
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(number int) {
			defer wg.Done()
			<-start
			_, _, err := runApp(a, []string{"bookmarks", "add", fmt.Sprint(number)}, "")
			results <- result{number, err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	var busy []int
	for result := range results {
		if errors.Is(result.err, errStateBusy) {
			busy = append(busy, result.number)
		} else if result.err != nil {
			t.Fatal(result.err)
		}
	}
	state, err := a.Store.saved()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Bookmarks) != 3-len(busy) {
		t.Fatal("successful concurrent bookmark update lost")
	}
	// The bounded lock may reject contention; retry rejected edits after writers finish.
	for _, number := range busy {
		if _, _, err := runApp(a, []string{"bookmarks", "add", fmt.Sprint(number)}, ""); err != nil {
			t.Fatal(err)
		}
	}
	state, err = a.Store.saved()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Bookmarks) != 3 {
		t.Fatal("concurrent bookmark update lost")
	}
	for _, item := range state.Last.Items {
		found := false
		for _, bookmark := range state.Bookmarks {
			if bookmark.ID == bookmarkID(item) {
				found = true
			}
		}
		if !found {
			t.Fatalf("bookmark missing for item %s", item.ID)
		}
	}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	before, _ := os.ReadFile(a.Store.savedPath())
	if _, _, err := runApp(a, []string{"bookmarks", "remove", state.Bookmarks[0].ID}, ""); !errors.Is(err, errStateBusy) {
		t.Fatalf("busy bookmark mutation must report state contention: %v", err)
	}
	if _, _, err := runApp(a, []string{"replay"}, ""); err != nil {
		t.Fatal("read-only replay unnecessarily acquired lock")
	}
	if _, _, err := runApp(a, []string{"--tips"}, ""); err != nil {
		t.Fatal("busy briefing snapshot failed")
	}
	after, _ := os.ReadFile(a.Store.savedPath())
	if string(before) != string(after) {
		t.Fatal("busy snapshot changed last replay")
	}
}

func TestSavedValidationAndOutputFailures(t *testing.T) {
	a := newTestApp(t)
	runApp(a, []string{"--tips"}, "Alex\n1\n")
	state, _ := a.Store.saved()
	original := *state.Last
	for _, mutate := range []func(*SavedBriefing){
		func(s *SavedBriefing) { s.CapturedAt = time.Time{} },
		func(s *SavedBriefing) { s.Items[0].Topic = "unknown" },
		func(s *SavedBriefing) { s.Items[0].URL = "javascript:bad()" },
		func(s *SavedBriefing) { s.Items[0].Kind = "invalid" },
	} {
		copy := SavedBriefing{CapturedAt: original.CapturedAt, Items: append([]Item(nil), original.Items...)}
		mutate(&copy)
		if validSavedBriefing(copy) == nil {
			t.Fatal("invalid saved item accepted")
		}
	}
	for _, args := range [][]string{{"replay"}, {"bookmarks", "list"}, {"bookmarks", "--help"}} {
		if err := a.Run(context.Background(), args, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
			t.Fatalf("output failure hidden: %v", args)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Run(ctx, []string{"replay"}, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
		t.Fatal("cancelled replay succeeded")
	}
	// A later briefing replaces replay but preserves bookmarks.
	runApp(a, []string{"bookmarks", "add"}, "")
	before, _ := a.Store.saved()
	runApp(a, []string{"--tips"}, "")
	after, _ := a.Store.saved()
	if !reflect.DeepEqual(before.Bookmarks, after.Bookmarks) {
		t.Fatal("new briefing changed bookmarks")
	}
}

func TestSavedIdentity(t *testing.T) {
	first := Item{ID: "first", Kind: "tip", URL: "https://example.org/docs"}
	second := first
	second.ID = "second"
	if bookmarkID(first) == bookmarkID(second) {
		t.Fatal("distinct tips sharing documentation must not merge")
	}
	first.Kind, second.Kind = "news", "news"
	second.URL += "#section"
	if bookmarkID(first) != bookmarkID(second) {
		t.Fatal("canonical article duplicates must reuse the existing bookmark")
	}
}

func TestSavedContentSurvivesSourcesAndSanitizes(t *testing.T) {
	a := newTestApp(t)
	stamp := a.Now().Add(-48 * time.Hour)
	item := Item{ID: "removed-article", Kind: "news", Topic: "aws",
		Title: "Article\x1b[31m title", Source: "Original publisher\x1b[2J",
		URL: "https://example.org/article", Published: stamp.Add(-time.Hour)}
	state := SavedState{Version: 1, Last: &SavedBriefing{CapturedAt: stamp, Items: []Item{item}},
		Bookmarks: []Bookmark{{ID: bookmarkID(item), SavedBriefing: SavedBriefing{CapturedAt: stamp, Items: []Item{item}}}}}
	if err := a.Store.writeSaved(state); err != nil {
		t.Fatal(err)
	}
	a.Sources = nil
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	a.Now = func() time.Time { t.Fatal("replay must not use a new clock"); return time.Time{} }
	for _, args := range [][]string{{"replay", "--color=always"}, {"bookmarks", "show", bookmarkID(item), "--detailed", "--plain"}} {
		out, warnings, err := runApp(a, args, "")
		if err != nil || warnings != "" || strings.Contains(out, "\x1b") ||
			!strings.Contains(out, "Original publisher") || !strings.Contains(out, item.URL) ||
			!strings.Contains(out, stamp.Format(time.RFC3339)) || !strings.Contains(out, item.Published.Format("02 Jan 2006 15:04 MST")) ||
			!strings.Contains(out, "not live news") {
			t.Fatalf("saved content not preserved safely: %q %s %v", out, warnings, err)
		}
	}
}

func TestSavedReadOnlyMissingState(t *testing.T) {
	a := newTestApp(t)
	for _, args := range [][]string{{"bookmarks", "list"}, {"replay", "--help"}, {"bookmarks", "--help"}} {
		if _, _, err := runApp(a, args, ""); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range []string{a.Store.savedPath(), a.Store.ConfigPath + ".lock", a.Store.ConfigPath, a.Store.CachePath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("read-only command created %s", path)
		}
	}
}

func TestSavedRejectedStateAndSize(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--tips"}, "Alex\n1\n"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err != nil {
		t.Fatal(err)
	}
	state, _ := a.Store.saved()
	state.Bookmarks = append(state.Bookmarks, state.Bookmarks[0])
	if err := writeJSON(a.Store.savedPath(), state); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(a.Store.savedPath())
	if _, _, err := runApp(a, []string{"--tips"}, ""); err == nil {
		t.Fatal("normal run overwrote invalid bookmarks")
	}
	after, _ := os.ReadFile(a.Store.savedPath())
	if !bytes.Equal(before, after) {
		t.Fatal("invalid state was mutated")
	}
	state.Bookmarks = state.Bookmarks[:1]
	state.Last.Items[0].Title = strings.Repeat("x", 16<<20)
	if err := a.Store.writeSaved(state); err == nil {
		t.Fatal("accepted file exceeding read limit")
	}
	after, _ = os.ReadFile(a.Store.savedPath())
	if !bytes.Equal(before, after) {
		t.Fatal("size rejection destroyed previous saved state")
	}
}

func TestSavedFailureAndNonBriefingPreserveSnapshot(t *testing.T) {
	a := newTestApp(t)
	if _, _, err := runApp(a, []string{"--tips"}, "Alex\n1\n"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(a.Store.savedPath())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<rss><channel><item><title>Docker update</title><link>https://example.org/docker</link></item></channel></rss>`)
	}))
	defer server.Close()
	a.Sources = []Source{{ID: "docker", Name: "Docker", Topic: "docker", Kind: "news", URL: server.URL}}
	commands := [][]string{{"doctor", "--json"}, {"--help"}, {"--version"}, {"feeds", "list"}, {"schedule", "refresh"}}
	for _, args := range commands {
		if _, _, err := runApp(a, args, ""); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if err := a.Run(context.Background(), []string{"--tips"}, strings.NewReader(""), failedWriter{}, &bytes.Buffer{}); err == nil {
		t.Fatal("normal display output failure hidden")
	}
	after, _ := os.ReadFile(a.Store.savedPath())
	if !bytes.Equal(before, after) {
		t.Fatal("refresh/read-only commands or failed display replaced replay")
	}
	// Both add and remove enforce the same state lock.
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, _, err := runApp(a, []string{"bookmarks", "add"}, ""); err == nil {
		t.Fatal("busy add succeeded")
	}
}

func TestSavedDoctorPrivacyAndPathCollision(t *testing.T) {
	a := newTestApp(t)
	private := "private-title-or-url-sentinel"
	if err := os.WriteFile(a.Store.savedPath(), []byte(private), 0600); err != nil {
		t.Fatal(err)
	}
	out, warnings, err := runApp(a, []string{"doctor", "--json"}, "")
	if err == nil || !strings.Contains(out, "saved_invalid") || strings.Contains(out+warnings+err.Error(), private) {
		t.Fatal("doctor leaked invalid saved content or missed failure")
	}
	a.Store.CachePath = a.Store.savedPath()
	if _, err := a.Store.saved(); err == nil {
		t.Fatal("accepted saved/cache path overlap")
	}
}
