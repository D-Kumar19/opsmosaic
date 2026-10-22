package brief

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

type doctorNoNetwork struct{ t *testing.T }

func (n doctorNoNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	n.t.Fatal("doctor attempted a network request")
	return nil, nil
}

func TestDoctorMissingStateIsReadOnly(t *testing.T) {
	a := newTestApp(t)
	a.Sources = DefaultSources()
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	for _, args := range [][]string{{"doctor"}, {"doctor", "--json"}} {
		out, warnings, err := runApp(a, args, "")
		if err != nil || warnings != "" || !strings.Contains(out, "profile_missing") || !strings.Contains(out, "cache_missing") {
			t.Fatalf("missing state: %q %q %v", out, warnings, err)
		}
		for _, path := range []string{a.Store.ConfigPath, a.Store.CachePath, a.Store.ConfigPath + ".lock"} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("doctor created %s", path)
			}
		}
	}
}

func TestDoctorHealthyJSONAndPrivacy(t *testing.T) {
	a := newTestApp(t)
	home := filepath.Dir(a.Store.ConfigPath)
	t.Setenv("HOME", home)
	a.Sources = DefaultSources()
	a.Client = &http.Client{Transport: doctorNoNetwork{t}}
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "PRIVATE_NAME", Topics: []string{"docker"}})
	cache := Cache{Version: 1, Sources: map[string]SourceCache{
		"docker": {Fetched: a.Now().Add(-time.Hour), Items: []Item{{
			Title: "PRIVATE_HEADLINE", URL: "https://private.invalid/token", FetchedAt: a.Now().Add(-time.Hour),
		}}},
		"PRIVATE_SOURCE": {Fetched: a.Now()},
	}, GreetingVisits: 7, Recent: []string{"PRIVATE_HISTORY"}}
	writeJSON(a.Store.CachePath, cache)
	before, _ := os.ReadFile(a.Store.CachePath)
	for _, args := range [][]string{{"doctor"}, {"doctor", "--json"}} {
		out, _, err := runApp(a, args, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"PRIVATE_", "private.invalid", home, `"topics"`, "token"} {
			if strings.Contains(out, secret) {
				t.Fatalf("private value leaked (%s): %s", secret, out)
			}
		}
		if len(args) == 2 {
			var report doctorReport
			if err := json.Unmarshal([]byte(out), &report); err != nil {
				t.Fatal(err)
			}
			if report.SchemaVersion != 1 || report.Status != "ok" || report.Paths.Config != "~/config.json" ||
				len(report.Sources) != len(DefaultSources()) || report.UnknownSources != 1 ||
				report.Sources[0].Status != "fresh" || report.Sources[0].AgeSeconds == nil ||
				*report.Sources[0].AgeSeconds != 3600 || report.Sources[0].ItemCount != 1 {
				t.Fatalf("unexpected JSON contract: %+v", report)
			}
			var fields map[string]json.RawMessage
			json.Unmarshal([]byte(out), &fields)
			var keys []string
			for key := range fields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			want := []string{"arch", "build_time", "checks", "commit", "go_version", "os", "paths", "schema_version", "sources", "status", "unknown_cached_sources", "version"}
			if !reflect.DeepEqual(keys, want) {
				t.Fatalf("JSON fields changed: %v", keys)
			}
		}
	}
	after, _ := os.ReadFile(a.Store.CachePath)
	if string(before) != string(after) {
		t.Fatal("doctor changed cache/history")
	}
	if _, err := os.Stat(a.Store.ConfigPath + ".lock"); !os.IsNotExist(err) {
		t.Fatal("doctor acquired a state lock")
	}
}

func TestDoctorInvalidStateDoesNotLeakValues(t *testing.T) {
	for _, content := range []string{
		`{"version":1,"name":"PRIVATE_NAME","topics":["PRIVATE_TOPIC"]}`,
		`{"version":42,"name":"PRIVATE_NAME","topics":["docker"]}`,
		`{"version":1, "PRIVATE_SECRET":`,
	} {
		a := newTestApp(t)
		os.WriteFile(a.Store.ConfigPath, []byte(content), 0600)
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		if err == nil || !strings.Contains(out, "profile_invalid") || strings.Contains(out, "PRIVATE_") {
			t.Fatalf("invalid profile: %s %v", out, err)
		}
		var report doctorReport
		if json.Unmarshal([]byte(out), &report) != nil || report.Status != "error" {
			t.Fatal("error report is not valid JSON")
		}
	}
	a := newTestApp(t)
	os.Mkdir(a.Store.CachePath, 0700)
	out, _, err := runApp(a, []string{"doctor"}, "")
	if err == nil || !strings.Contains(out, "cache_unreadable") {
		t.Fatalf("non-regular cache: %q %v", out, err)
	}
}

func TestDoctorCacheValidationAndAge(t *testing.T) {
	for _, age := range []time.Duration{0, cacheTTL - time.Second, cacheTTL, 5 * 24 * time.Hour, -time.Hour} {
		a := newTestApp(t)
		a.Sources = []Source{{ID: "docker"}}
		writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
		writeJSON(a.Store.CachePath, Cache{Version: 1, Sources: map[string]SourceCache{
			"docker": {Fetched: a.Now().Add(-age), RetryAfter: a.Now().Add(time.Minute), Items: []Item{{}}},
		}})
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		var report doctorReport
		if err != nil || json.Unmarshal([]byte(out), &report) != nil {
			t.Fatalf("cache age: %q %v", out, err)
		}
		expected := "fresh"
		if age < 0 {
			expected = "unknown"
		} else if age >= cacheTTL {
			expected = "stale"
		}
		if report.Sources[0].Status != expected || !report.Sources[0].RetryPending {
			t.Fatalf("age %s: %+v", age, report.Sources[0])
		}
	}
	for _, content := range []string{`not json`, `{"version":2,"sources":{}}`, `{"version":1,"sources":null}`, `{"version":1,"sources":{}} {}`} {
		a := newTestApp(t)
		os.WriteFile(a.Store.CachePath, []byte(content), 0600)
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		if err == nil || !strings.Contains(out, "cache_invalid") {
			t.Fatalf("accepted invalid cache: %s %v", out, err)
		}
	}
}

func TestDoctorPartialCacheOldestItemAndUnknownTime(t *testing.T) {
	a := newTestApp(t)
	a.Sources = []Source{{ID: "docker"}}
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
	cache := Cache{Version: 1, Sources: map[string]SourceCache{"docker": {
		Fetched: a.Now(), Items: []Item{{FetchedAt: a.Now()}, {FetchedAt: a.Now().Add(-24 * time.Hour)}},
	}}}
	for _, unknown := range []bool{false, true} {
		if unknown {
			entry := cache.Sources["docker"]
			entry.Fetched = time.Time{}
			entry.Items = []Item{{}}
			cache.Sources["docker"] = entry
		}
		writeJSON(a.Store.CachePath, cache)
		out, _, err := runApp(a, []string{"doctor", "--json"}, "")
		var report doctorReport
		if err != nil || json.Unmarshal([]byte(out), &report) != nil || report.Status != "warning" {
			t.Fatalf("partial age: %s %v", out, err)
		}
		expected := "stale"
		if unknown {
			expected = "unknown"
		}
		if report.Sources[0].Status != expected {
			t.Fatalf("wrong per-item age: %+v", report.Sources[0])
		}
	}
}

func TestDoctorArgumentsOutputAndPathRedaction(t *testing.T) {
	for _, args := range [][]string{{"doctor", "extra"}, {"doctor", "--online"}, {"doctor", "--tips"}, {"doctor", "--json", "extra"}} {
		if _, _, err := runApp(newTestApp(t), args, ""); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}

	a := newTestApp(t)
	if err := a.Run(context.Background(), []string{"doctor", "--json"}, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
		t.Fatal("output failure hidden")
	}
	out, _, err := runApp(a, []string{"doctor", "--help"}, "")
	if err != nil || !strings.Contains(out, "--json") {
		t.Fatalf("doctor help: %s %v", out, err)
	}
	for _, tc := range []struct{ path, home, want string }{
		{"/home/alex/config", "/home/alex", "~/config"},
		{"/home/alex", "/home/alex", "~"},
		{"/home/alexander/config", "/home/alex", "/home/alexander/config"},
		{"/tmp/config", "/home/alex", "/tmp/config"},
		{"/tmp/config", "", "/tmp/config"},
	} {
		if got := doctorPath(tc.path, tc.home); got != tc.want {
			t.Errorf("redaction %q: got %q want %q", tc.path, got, tc.want)
		}
	}
}

func TestDoctorBusyStateAndCancellation(t *testing.T) {
	a := newTestApp(t)
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, _, err := runApp(a, []string{"doctor"}, ""); err != nil {
		t.Fatalf("doctor should not acquire the held lock: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var output strings.Builder
	if err := a.Run(ctx, []string{"doctor"}, strings.NewReader(""), &output, &output); err == nil || output.Len() != 0 {
		t.Fatal("cancelled doctor did not stop without output")
	}
	for _, args := range [][]string{{"doctor"}, {"doctor", "--help"}} {
		if err := a.Run(context.Background(), args, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
			t.Fatalf("writer failure hidden for %v", args)
		}
	}
}
