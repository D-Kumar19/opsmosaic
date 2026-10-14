package brief

import (
	"strings"
	"testing"
	"time"
)

func TestGreetingHourBoundaries(t *testing.T) {
	for _, tc := range []struct {
		hour  int
		title string
	}{
		{0, "Hey night owl"}, {4, "Hey night owl"},
		{5, "Good morning"}, {11, "Good morning"},
		{12, "Good afternoon"}, {16, "Good afternoon"},
		{17, "Good evening"}, {21, "Good evening"},
		{22, "Hey night owl"}, {23, "Hey night owl"},
	} {
		now := time.Date(2026, 10, 14, tc.hour, 0, 0, 0, time.UTC)
		title, comment := greeting("Alex", now, 0)
		if !strings.Contains(title, tc.title+", Alex!") || comment == "" {
			t.Errorf("hour %d: %q, %q", tc.hour, title, comment)
		}
	}
}

func TestGreetingRotationAndLocalTime(t *testing.T) {
	now := time.Date(2026, 10, 14, 4, 0, 0, 0, time.UTC).In(time.FixedZone("IST", 5*3600+30*60))
	comments := map[string]bool{}
	for visit := uint64(0); visit < 3; visit++ {
		title, comment := greeting("\x1b[31mAlex\x1b[0m", now, visit)
		if !strings.Contains(title, "Good morning, Alex!") || strings.Contains(title, "\x1b") {
			t.Fatalf("local time or name sanitization failed: %q", title)
		}
		comments[comment] = true
	}
	if len(comments) != 3 {
		t.Fatalf("comments did not rotate: %v", comments)
	}
	_, first := greeting("Alex", now, 0)
	_, repeated := greeting("Alex", now, 3)
	if first != repeated {
		t.Fatal("greeting rotation must wrap")
	}
}

func TestCLIGreetingRotationPersists(t *testing.T) {
	a := newTestApp(t)
	a.Now = func() time.Time { return time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC) }
	comments := []string{"Need a coffee?", "Fresh day, fresh context.", "Breakfast for your brain", "Need a coffee?"}
	for i, expected := range comments {
		input := ""
		if i == 0 {
			input = "Alex\n3\n"
		}
		out, _, err := runApp(a, []string{"--tips"}, input)
		if err != nil || !strings.Contains(out, "☀️ Good morning, Alex!") || !strings.Contains(out, expected) || !strings.Contains(out, "Showing 1 items") {
			t.Fatalf("visit %d: %s, %v", i, out, err)
		}
		// Reload state via a fresh App rather than relying on memory.
		next := newTestApp(t)
		next.Store, next.Now = a.Store, a.Now
		a = next
	}
	cache, err := a.Store.cache()
	if err != nil || cache.GreetingVisits != 4 {
		t.Fatalf("visits were not persisted: %+v, %v", cache, err)
	}
}

func TestLegacyCacheGreetingDefault(t *testing.T) {
	a := newTestApp(t)
	if err := writeJSON(a.Store.CachePath, map[string]interface{}{"version": 1, "sources": map[string]SourceCache{}}); err != nil {
		t.Fatal(err)
	}
	cache, err := a.Store.cache()
	if err != nil || cache.GreetingVisits != 0 {
		t.Fatalf("legacy cache incompatible: %+v, %v", cache, err)
	}
}
