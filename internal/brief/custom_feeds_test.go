package brief

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCustomFeedCommandsAndPersistence(t *testing.T) {
	a := newTestApp(t)
	add := []string{"feeds", "add", "--id", "example", "--name", "Example updates", "--topic", "docker", "--kind", "news", "--url", "https://example.org/feed"}
	for _, args := range [][]string{{"feeds", "list"}, add, {"feeds", "list"}} {
		out, _, err := runApp(a, args, "")
		if err != nil {
			t.Fatalf("%v: %s %v", args, out, err)
		}
	}
	if _, _, err := runApp(a, add, ""); err == nil {
		t.Fatal("duplicate ID/URL accepted")
	}
	config, err := a.Store.customFeeds()
	if err != nil || len(config.Feeds) != 1 || config.Version != 1 {
		t.Fatalf("feed round trip: %+v %v", config, err)
	}
	info, _ := os.Stat(a.Store.feedsPath())
	if info.Mode().Perm() != 0600 {
		t.Fatal("feed configuration must be private")
	}
	if _, err := os.Stat(a.Store.ConfigPath); !os.IsNotExist(err) {
		t.Fatal("feed commands created a profile")
	}
	if _, _, err := runApp(a, []string{"feeds", "remove", "unknown"}, ""); err == nil {
		t.Fatal("missing feed removal was hidden")
	}
	if _, _, err := runApp(a, []string{"feeds", "remove", "example"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"feeds", "remove", "example"}, ""); err == nil {
		t.Fatal("repeated missing feed removal was hidden")
	}
}

func TestCustomFeedInvalidConfiguration(t *testing.T) {
	base := CustomFeed{ID: "example", Name: "Example", Topic: "docker", Kind: "news", URL: "https://example.org/feed"}
	for _, mutate := range []func(*CustomFeed){
		func(f *CustomFeed) { f.ID = "../bad" },
		func(f *CustomFeed) { f.Name = "\x1b[31munsafe" },
		func(f *CustomFeed) { f.Topic = "unknown" },
		func(f *CustomFeed) { f.Kind = "tip" },
		func(f *CustomFeed) { f.URL = "http://example.org/feed" },
		func(f *CustomFeed) { f.URL = "https://user:secret@example.org/feed" },
		func(f *CustomFeed) { f.URL = "https://127.0.0.1/feed" },
	} {
		feed := base
		mutate(&feed)
		if err := (FeedConfig{Version: 1, Feeds: []CustomFeed{feed}}).Validate(); err == nil {
			t.Fatalf("invalid feed accepted: %+v", feed)
		}
	}
	other := base
	other.ID = "other"
	other.URL = "https://EXAMPLE.org:443/feed/"
	if err := (FeedConfig{Version: 1, Feeds: []CustomFeed{base, other}}).Validate(); err == nil {
		t.Fatal("duplicate canonical URL or fragment accepted")
	}
	for _, data := range []string{`not json`, `{}`, `{"version":1}`, `{"feeds":[]}`, `{"version":2,"feeds":[]}`, `{"version":1,"feeds":null}`, `{"version":1,"feeds":[]} {}`} {
		a := newTestApp(t)
		os.WriteFile(a.Store.feedsPath(), []byte(data), 0600)
		if _, _, err := runApp(a, []string{"feeds", "list"}, ""); err == nil {
			t.Fatal("corrupt feed configuration hidden")
		}
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		if err == nil || !strings.Contains(out, "feeds_invalid") {
			t.Fatal("doctor did not diagnose invalid custom feeds")
		}
	}
}

func TestCustomFeedOfflineLifecycle(t *testing.T) {
	a := newTestApp(t)
	feed := CustomFeed{ID: "example", Name: "Example", Topic: "docker", Kind: "release", URL: "https://example.org/feed"}
	writeJSON(a.Store.feedsPath(), FeedConfig{Version: 1, Feeds: []CustomFeed{feed}})
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
	item := Item{ID: "example-item", Title: "A custom release", Topic: "docker", Kind: "release", Source: "Example", URL: "https://example.org/releases/v1", FetchedAt: a.Now()}
	writeJSON(a.Store.CachePath, Cache{Version: 1, Sources: map[string]SourceCache{feed.source().ID: {Fetched: a.Now(), Items: []Item{item}}}})
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	out, _, err := runApp(a, []string{"--offline", "--news"}, "")
	if err != nil || !strings.Contains(out, "A custom release") || !strings.Contains(out, "RELEASE |") {
		t.Fatalf("offline custom feed: %s %v", out, err)
	}
	combined, err := a.withCustomFeeds()
	if err != nil || len(combined.Sources) != len(a.Sources)+1 || len(a.Sources) != 0 {
		t.Fatal("built-ins/injected sources mutated or custom source missing")
	}
	out, _, err = runApp(a, []string{"doctor", "--json"}, "")
	if err != nil || !strings.Contains(out, `"id": "custom-1"`) || strings.Contains(out, "https://example.org") || strings.Contains(out, `"example"`) {
		t.Fatalf("custom doctor leaks configuration or omits status: %s %v", out, err)
	}
	if _, _, err := runApp(a, []string{"feeds", "remove", "example"}, ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runApp(a, []string{"--offline", "--news"}, ""); err == nil {
		t.Fatal("removed feed stayed active through old cache")
	}
	cache, _ := a.Store.cache()
	if _, exists := cache.Sources[feed.source().ID]; exists {
		t.Fatal("removed feed cache not pruned")
	}
}

func TestCustomFeedBusyAndBadArguments(t *testing.T) {
	a := newTestApp(t)
	for _, args := range [][]string{{"feeds"}, {"feeds", "add"}, {"feeds", "list", "extra"}, {"feeds", "remove"}, {"feeds", "unknown"}} {
		if _, _, err := runApp(a, args, ""); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, _, err := runApp(a, []string{"feeds", "remove", "example"}, ""); err == nil {
		t.Fatal("busy removal succeeded")
	}
	if _, _, err := runApp(a, []string{"feeds", "list"}, ""); err != nil {
		t.Fatal("listing must not require lock")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(a.Store.ConfigPath), "feeds.json")); !os.IsNotExist(err) {
		t.Fatal("read-only list wrote configuration")
	}
}

func TestPublicFeedURLsAndAddresses(t *testing.T) {
	for _, raw := range []string{
		"http://example.org/feed", "https://localhost/feed", "https://name.local/feed",
		"https://name.internal/feed", "https://user@example.org/feed", "https://example.org:8443/feed",
		"https://example.org/feed#fragment", "https://127.1/feed", "https://2130706433/feed",
		"https://example.org/feed?token=secret", "https://example.org/feed?",
		"https://[::1]/feed", "https://169.254.169.254/latest/meta-data",
		"https://example.org/\nfeed", "https://example.org./feed",
	} {
		if _, err := publicFeedURL(raw); err == nil {
			t.Errorf("accepted URL %q", raw)
		}
	}
	for _, address := range []string{"127.0.0.1", "::1", "10.0.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1", "0.0.0.0", "224.0.0.1", "192.0.2.1", "198.18.0.1", "203.0.113.1", "fc00::1", "fe80::1", "2001:db8::1", "::ffff:127.0.0.1", "64:ff9b::7f00:1"} {
		if publicFeedIP(net.ParseIP(address)) {
			t.Errorf("accepted nonpublic address %s", address)
		}
	}
	for _, address := range []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"} {
		if !publicFeedIP(net.ParseIP(address)) {
			t.Errorf("rejected public address %s", address)
		}
	}
}

func TestPublicFeedDNSGuard(t *testing.T) {
	for _, addresses := range [][]net.IPAddr{
		{}, {{IP: net.ParseIP("127.0.0.1")}},
		{{IP: net.ParseIP("8.8.8.8")}, {IP: net.ParseIP("10.0.0.1")}},
	} {
		dialled := false
		transport := publicFeedTransport(
			func(context.Context, string) ([]net.IPAddr, error) { return addresses, nil },
			func(context.Context, string, string) (net.Conn, error) { dialled = true; return nil, nil })
		client := &http.Client{Transport: transport}
		if _, err := client.Get("https://example.org/feed"); err == nil || dialled {
			t.Fatalf("private/mixed/empty DNS results reached dialer: %v %v", addresses, err)
		}
		transport.CloseIdleConnections()
	}
}

func TestCustomFeedLimitsIdentityAndErrors(t *testing.T) {
	base := CustomFeed{ID: "example", Name: "Example", Topic: "docker", Kind: "news", URL: "https://example.org/feed"}
	for _, mutate := range []func(*CustomFeed){
		func(f *CustomFeed) { f.URL += "/other" },
		func(f *CustomFeed) { f.Topic = "aws" },
		func(f *CustomFeed) { f.Kind = "release" },
		func(f *CustomFeed) { f.Name = "Renamed" },
	} {
		copy := base
		mutate(&copy)
		if copy.source().ID == base.source().ID {
			t.Fatal("changed definition reused stale cache identity")
		}
	}
	config := FeedConfig{Version: 1, Feeds: []CustomFeed{}}
	for i := 0; i < 21; i++ {
		feed := base
		feed.ID = "feed-" + strings.Repeat("a", i+1)
		feed.URL += "/" + feed.ID
		config.Feeds = append(config.Feeds, feed)
	}
	if err := config.Validate(); err == nil {
		t.Fatal("unbounded feed count accepted")
	}
	config.Feeds = config.Feeds[:20]
	if err := config.Validate(); err != nil {
		t.Fatalf("20 feeds should be allowed: %v", err)
	}
	config.Feeds = []CustomFeed{base}
	a := newTestApp(t)
	writeJSON(a.Store.feedsPath(), config)
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runApp(a, []string{"feeds", "add", "--id", "other", "--name", "Other", "--topic", "docker", "--kind", "news", "--url", "https://example.org/other"}, "")
	lock.Close()
	if !errors.Is(err, errStateBusy) {
		t.Fatalf("busy add: %v", err)
	}
	if err := a.Run(context.Background(), []string{"feeds", "list"}, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
		t.Fatal("listing output failure hidden")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Run(ctx, []string{"feeds", "remove", "example"}, strings.NewReader(""), failedWriter{}, failedWriter{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation hidden: %v", err)
	}
}

func TestPublicFeedTransportFailureAndCancellation(t *testing.T) {
	for _, lookup := range []feedLookup{
		func(context.Context, string) ([]net.IPAddr, error) { return nil, errors.New("DNS failure") },
		func(context.Context, string) ([]net.IPAddr, error) {
			return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
		},
	} {
		transport := publicFeedTransport(lookup, func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("connection failure")
		})
		if _, err := publicFeedClient(transport).Get("https://example.org/feed"); err == nil {
			t.Fatal("network failure hidden")
		}
		if _, err := transport.DialContext(context.Background(), "tcp", "example.org:80"); err == nil {
			t.Fatal("HTTP port accepted")
		}
		transport.CloseIdleConnections()
	}
	source := Source{PublicOnly: true, URL: "https://127.0.0.1/feed"}
	if _, err := source.Fetch(context.Background(), &http.Client{Transport: doctorNoNetwork{t}}); err == nil {
		t.Fatal("production custom source skipped URL policy")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source.URL = "https://example.org/feed"
	if _, err := source.Fetch(ctx, &http.Client{Transport: doctorNoNetwork{t}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("production fetch ignored cancellation: %v", err)
	}
}
func TestPublicFeedTLSAndRedirects(t *testing.T) {
	for _, target := range []string{"", "http://example.org/feed", "https://127.0.0.1/feed", "https://private.example.org/feed", "https://example.org/feed?token=secret", "https://example.org/loop"} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if target != "" {
				http.Redirect(w, r, target, http.StatusFound)
				return
			}
			w.Write([]byte(`<feed><entry><title>Unfiltered ordinary title</title><link href="https://example.org/article"/></entry></feed>`))
		}))
		dials := 0
		transport := publicFeedTransport(
			func(_ context.Context, host string) ([]net.IPAddr, error) {
				if host == "private.example.org" {
					return []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}}, nil
				}
				return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
			},
			func(ctx context.Context, network, address string) (net.Conn, error) {
				if address != "8.8.8.8:443" {
					t.Errorf("DNS result was not pinned: %s", address)
				}
				dials++
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			})
		transport.TLSClientConfig = server.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
		transport.TLSClientConfig.ServerName = "example.com"
		client := publicFeedClient(transport)
		body, err := getBody(context.Background(), client, "https://example.org/feed")
		if target == "" {
			items, parseErr := parseFeed(body, Source{Name: "Example", Topic: "docker", Kind: "news"})
			if err != nil || parseErr != nil || len(items) != 1 || items[0].Title != "Unfiltered ordinary title" {
				t.Fatalf("public TLS feed: %s %v %v", body, err, parseErr)
			}
		} else if err == nil {
			t.Fatalf("unsafe/loop redirect accepted: %s", target)
		}
		if strings.Contains(target, "127.0.0.1") || strings.HasPrefix(target, "http:") {
			if dials != 1 {
				t.Fatal("unsafe redirect reached connection stage")
			}
		}
		transport.CloseIdleConnections()
		server.Close()
	}
}

func TestCustomFeedConcurrentEditsAndMigration(t *testing.T) {
	a := newTestApp(t)
	profile := Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}}
	writeJSON(a.Store.ConfigPath, profile)
	before, _ := os.ReadFile(a.Store.ConfigPath)
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for _, id := range []string{"first", "second"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_, _, err := runApp(a, []string{"feeds", "add", "--id", id, "--name", "Example", "--topic", "docker", "--kind", "news", "--url", "https://example.org/" + id}, "")
			failures <- err
		}(id)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	config, err := a.Store.customFeeds()
	if err != nil || len(config.Feeds) != 2 {
		t.Fatalf("concurrent update lost: %+v %v", config, err)
	}
	after, _ := os.ReadFile(a.Store.ConfigPath)
	if string(before) != string(after) {
		t.Fatal("adding feeds migrated/damaged existing profile")
	}
}

func TestCustomFeedSharedCacheLifecycle(t *testing.T) {
	// Exercise common cache logic with local RSS; the guarded HTTPS connection
	// layer is independently exercised above with a TLS server and pinned DNS.
	a := newTestApp(t)
	fail := false
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if fail {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`<rss><channel><item><title>Ordinary &lt;b&gt;entry&lt;/b&gt;</title><link>https://example.org/item</link></item><item><title>Unsafe link</title><link>javascript:bad()</link></item></channel></rss>`))
	}))
	defer server.Close()
	a.Sources = []Source{{ID: "custom:fixture", Name: "Example", Topic: "docker", Kind: "news", URL: server.URL}}
	cache := Cache{Version: 1, Sources: map[string]SourceCache{}}
	items, notes := a.news(context.Background(), &cache, []string{"docker"}, a.Now(), false, false)
	if len(notes) != 0 || len(items) != 1 || items[0].Title != "Ordinary entry" {
		t.Fatalf("custom RSS sanitation: %v %v", items, notes)
	}
	items, notes = a.news(context.Background(), &cache, []string{"docker"}, a.Now(), false, false)
	if calls != 1 || len(notes) != 0 || len(items) != 1 || !items[0].Cached {
		t.Fatal("custom TTL reuse failed")
	}
	fail = true
	items, notes = a.news(context.Background(), &cache, []string{"docker"}, a.Now().Add(24*time.Hour), true, false)
	if len(notes) != 1 || len(items) != 1 || !items[0].Cached {
		t.Fatal("custom failure discarded cache")
	}
	items, notes = a.news(context.Background(), &cache, []string{"aws"}, a.Now(), true, false)
	if len(items) != 0 || len(notes) != 0 || calls != 2 {
		t.Fatal("unselected custom topic fetched")
	}
}
