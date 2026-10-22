package brief

import (
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestStableReleaseLabels(t *testing.T) {
	for _, label := range []string{
		"v1.2.3", "1.2.3", "v0.0.0", "v10.20.30+build.01",
		"helm-v3.16.0", "component/v1.2.3", "component@1.2.3",
		"argo-cd-v3.1.0", "Release v1.2.3",
	} {
		if !stableRelease(Item{Title: label}) {
			t.Errorf("stable label rejected: %s", label)
		}
	}
	for _, label := range []string{
		"v1.2.3-rc.1", "v1.2.3-alpha.1", "1.2.3-beta", "v1.2.3-nightly",
		"nightly", "latest", "stable", "v1.2", "v01.2.3", "v1.02.3", "v1.2.03",
		"New release v1.2.3", "v1.2.3 fixes v1.2.2", "v1.2.3+", "v1.2.3+build..1",
		"v1.2.3-rc.1+build", "2026-10-19", "nested/component/v1.2.3", "",
		"nightly-v1.2.3", "component-preview-v1.2.3", "snapshot/v1.2.3",
	} {
		if stableRelease(Item{Title: label}) {
			t.Errorf("unstable/unknown label accepted: %s", label)
		}
	}
}

func TestStableReleaseTagPrecedenceAndFeedOutput(t *testing.T) {
	body := []byte(`<feed>
<entry><title>Marketing title</title><link href="https://github.com/helm/helm/releases/tag/v3.16.0"/></entry>
<entry><title>v3.16.0</title><link href="https://github.com/helm/helm/releases/tag/v3.16.0-rc.1"/></entry>
<entry><title>Component update</title><link href="https://github.com/example/project/releases/tag/component%2Fv1.2.3"/></entry>
<entry><title>v1.2.3</title><link href="https://github.com/example/project/releases/tag/unknown"/></entry>
</feed>`)
	items, err := parseFeed(body, Source{Name: "Fixture", Kind: "release", Topic: "helm"})
	if err != nil || len(items) != 4 {
		t.Fatalf("adapter fixture: %+v %v", items, err)
	}
	got := stableItems(items)
	if len(got) != 2 || got[0].Title != "Marketing title" || got[1].Title != "Component update" {
		t.Fatalf("tag precedence: %+v", got)
	}
}

func TestStableOnlyOfflineCLIAndCachePreservation(t *testing.T) {
	a := newTestApp(t)
	feed := CustomFeed{ID: "example", Name: "Example", Topic: "helm", Kind: "release", URL: "https://example.org/feed"}
	writeJSON(a.Store.feedsPath(), FeedConfig{Version: 1, Feeds: []CustomFeed{feed}})
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"helm"}})
	items := []Item{
		{ID: "stable", Kind: "release", Topic: "helm", Title: "v1.2.3", URL: "https://example.org/stable"},
		{ID: "rc", Kind: "release", Topic: "helm", Title: "v1.3.0-rc.1", URL: "https://example.org/rc"},
		{ID: "unknown", Kind: "release", Topic: "helm", Title: "Unversioned update", URL: "https://example.org/unknown"},
		{ID: "news", Kind: "news", Topic: "helm", Title: "News about a beta", URL: "https://example.org/news"},
	}
	writeJSON(a.Store.CachePath, Cache{Version: 1, Sources: map[string]SourceCache{
		feed.source().ID: {Fetched: a.Now(), Items: items},
	}})
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	out, _, err := runApp(a, []string{"--offline", "--news", "--stable-only", "--count", "20"}, "")
	if err != nil || !strings.Contains(out, "v1.2.3") || !strings.Contains(out, "News about a beta") ||
		strings.Contains(out, "v1.3.0-rc.1") || strings.Contains(out, "Unversioned update") {
		t.Fatalf("stable output: %s %v", out, err)
	}
	cache, _ := a.Store.cache()
	if !reflect.DeepEqual(cache.Sources[feed.source().ID].Items, items) {
		t.Fatal("filter changed cached source entries")
	}
	out, _, err = runApp(a, []string{"--offline", "--news", "--count", "20"}, "")
	if err != nil || !strings.Contains(out, "v1.3.0-rc.1") || !strings.Contains(out, "Unversioned update") {
		t.Fatalf("defaults changed: %s %v", out, err)
	}
}

func TestStableOnlyEmptyFallbackAndArguments(t *testing.T) {
	a := newTestApp(t)
	a.Sources = []Source{{ID: "helm", Topic: "helm", Kind: "release"}}
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"helm"}})
	writeJSON(a.Store.CachePath, Cache{Version: 1, Sources: map[string]SourceCache{
		"helm": {Fetched: a.Now(), Items: []Item{{ID: "rc", Kind: "release", Topic: "helm", Title: "v1.0.0-rc.1", URL: "https://example.org/rc"}}},
	}})
	out, _, err := runApp(a, []string{"--offline", "--news", "--stable-only"}, "")
	if err == nil || out != "" || !strings.Contains(err.Error(), "stable-only") {
		t.Fatalf("empty stable news must fail: %s %v", out, err)
	}
	out, warnings, err := runApp(a, []string{"--offline", "--stable-only"}, "")
	if err != nil || strings.Count(out, "TIP |") != 1 || !strings.Contains(warnings, "stable-only") {
		t.Fatalf("mixed empty fallback: %s %s %v", out, warnings, err)
	}
	out, _, err = runApp(a, []string{"--tips", "--stable-only"}, "")
	if err != nil || strings.Count(out, "TIP |") != 1 {
		t.Fatal("stable filter should leave tips unchanged")
	}
	for _, args := range [][]string{{"--version", "--stable-only"}, {"config", "--stable-only"}, {"--stable-only=invalid"}} {
		if _, _, err := runApp(a, args, ""); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestStableOnlyRefreshRetainsPrereleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`<rss><channel>
<item><title>v1.2.3</title><link>https://example.org/stable</link></item>
<item><title>v1.3.0-beta.1</title><link>https://example.org/beta</link></item>
</channel></rss>`))
	}))
	defer server.Close()
	a := newTestApp(t)
	a.Sources = []Source{{ID: "helm", Topic: "helm", Kind: "release", URL: server.URL}}
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"helm"}})
	out, _, err := runApp(a, []string{"refresh", "--news", "--stable-only"}, "")
	if err != nil || !strings.Contains(out, "v1.2.3") || strings.Contains(out, "beta") {
		t.Fatalf("refresh filter: %s %v", out, err)
	}
	cache, _ := a.Store.cache()
	if len(cache.Sources["helm"].Items) != 2 {
		t.Fatal("refresh discarded prereleases from cache")
	}
	if _, err := os.Stat(a.Store.feedsPath()); !os.IsNotExist(err) {
		t.Fatal("filter created custom configuration")
	}
}
