package brief

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestShellHookLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shell config")
	original := "# personal settings\nexport EXAMPLE=yes\n"
	if err := os.WriteFile(path, []byte(original), 0640); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := editHook(path, true); err != nil {
			t.Fatal(err)
		}
	}
	content, _ := os.ReadFile(path)
	if strings.Count(string(content), hookStart) != 1 || !strings.HasPrefix(string(content), original) {
		t.Fatalf("duplicate hook or lost content: %s", content)
	}
	backup, _ := os.ReadFile(path + ".opsmosaic-backup")
	if string(backup) != original {
		t.Fatal("original backup missing")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0640 {
		t.Fatal("permissions changed")
	}
	for i := 0; i < 2; i++ {
		if err := editHook(path, false); err != nil {
			t.Fatal(err)
		}
	}
	content, _ = os.ReadFile(path)
	if string(content) != original {
		t.Fatalf("uninstall changed original: %q", content)
	}
}

func TestShellHookRejectsUnsafeFiles(t *testing.T) {
	for _, content := range []string{hookStart, hookEnd, hookEnd + "\n" + hookStart, hookStart + hookStart + hookEnd, "no newline"} {
		path := filepath.Join(t.TempDir(), "rc")
		os.WriteFile(path, []byte(content), 0600)
		if err := editHook(path, true); err == nil {
			t.Fatalf("accepted malformed input %q", content)
		}
		after, _ := os.ReadFile(path)
		if string(after) != content {
			t.Fatal("modified rejected input")
		}
	}
	path := filepath.Join(t.TempDir(), "rc")
	os.Symlink("missing-target", path)
	if err := editHook(path, true); err == nil {
		t.Fatal("followed shell config symlink")
	}
}

func TestShellGreetingWithoutProfile(t *testing.T) {
	a := newTestApp(t)
	out, warnings, err := runApp(a, []string{"shell", "greeting"}, "")
	if err != nil || out != "" || warnings != "" {
		t.Fatalf("missing profile should skip: %q %q %v", out, warnings, err)
	}
	if _, err := os.Stat(a.Store.ConfigPath + ".lock"); !os.IsNotExist(err) {
		t.Fatal("skip created lock")
	}
	if err := writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}}); err != nil {
		t.Fatal(err)
	}
	out, _, err = runApp(a, []string{"shell", "greeting"}, "")
	if err != nil || strings.Count(out, "TIP |") != 1 {
		t.Fatalf("offline greeting: %s %v", out, err)
	}
}

func TestShellHookNoninteractiveAndMissingExecutable(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		binary, err := exec.LookPath(shell)
		if err != nil {
			t.Skipf("%s unavailable", shell)
		}
		for _, args := range [][]string{{"--norc", "-c"}, {"--norc", "-i", "-c"}} {
			if shell == "zsh" {
				args[0] = "-f"
			}
			command := exec.Command(binary, append(args, shellHook())...)
			command.Env = []string{"PATH=/nonexistent", "HOME=" + t.TempDir()}
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("%s missing executable/non-TTY: %s %v", shell, output, err)
			}
		}
	}
}

func TestScheduleLifecycleAndFailures(t *testing.T) {
	home := t.TempDir()
	store := Store{filepath.Join(home, "profile.json"), filepath.Join(home, "cache.json")}
	writeJSON(store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
	var calls []string
	fail := false
	runner := func(_ context.Context, args ...string) error {
		calls = append(calls, args[0])
		if fail {
			return errors.New("launchd unavailable")
		}
		return nil
	}
	var output bytes.Buffer
	run := func(action string) error {
		return manageSchedule(context.Background(), action, home, "/Applications/A & B/opsmosaic", store, 501, &output, runner)
	}
	path := filepath.Join(home, "Library", "LaunchAgents", scheduleLabel+".plist")
	if err := run("status"); err != nil || len(calls) != 0 {
		t.Fatal("unconfigured status invoked launchd")
	}
	fail = true
	if err := run("enable"); err == nil {
		t.Fatal("bootstrap failure hidden")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("failed bootstrap left configuration")
	}
	fail = false
	for _, action := range []string{"enable", "enable", "status"} {
		if err := run(action); err != nil {
			t.Fatal(err)
		}
	}
	content, _ := os.ReadFile(path)
	for _, expected := range []string{"<integer>10800</integer>", "A &amp; B", store.ConfigPath, store.CachePath} {
		if !strings.Contains(string(content), expected) {
			t.Fatalf("missing plist field %q", expected)
		}
	}
	fail = true
	if err := run("disable"); err == nil {
		t.Fatal("bootout failure hidden")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("failed disable lost configuration")
	}
	fail = false
	for _, action := range []string{"disable", "disable"} {
		if err := run(action); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Join(calls, ",") != "bootstrap,bootstrap,print,print,bootout,bootout" {
		t.Fatalf("unexpected launchctl calls: %v", calls)
	}
}

func TestIntegrationArgumentErrors(t *testing.T) {
	for _, args := range [][]string{{"shell"}, {"shell", "install", "fish", "rc"}, {"shell", "greeting", "extra"}, {"schedule"}, {"schedule", "unknown"}, {"schedule", "refresh"}} {
		a := newTestApp(t)
		if _, _, err := runApp(a, args, ""); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestScheduledStatePaths(t *testing.T) {
	t.Setenv("OPSMOSAIC_CONFIG_PATH", "/tmp/custom config.json")
	t.Setenv("OPSMOSAIC_CACHE_PATH", "/tmp/custom cache.json")
	store, err := DefaultStore()
	if err != nil || store.ConfigPath != "/tmp/custom config.json" {
		t.Fatalf("scheduled paths: %+v %v", store, err)
	}
	t.Setenv("OPSMOSAIC_CACHE_PATH", "")
	if _, err := DefaultStore(); err == nil {
		t.Fatal("accepted incomplete state override")
	}
}

func TestShellGreetingStartupCost(t *testing.T) {
	a := newTestApp(t)
	start := time.Now()
	for i := 0; i < 100; i++ {
		if _, _, err := runApp(a, []string{"shell", "greeting"}, ""); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("missing-profile greeting: mean %s over 100 runs (no fixed machine-dependent threshold)", time.Since(start)/100)
}

func TestScheduledRefreshPreservesDisplayState(t *testing.T) {
	a := newTestApp(t)
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
	cache := Cache{Version: 1, Sources: map[string]SourceCache{}, GreetingVisits: 12, Recent: []string{"previous"}}
	writeJSON(a.Store.CachePath, cache)
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`<rss><channel><item><title>Docker example</title><link>https://example.invalid/docker</link></item></channel></rss>`))
	}))
	defer server.Close()
	a.Sources = []Source{{ID: "docker", Name: "Docker fixture", Topic: "docker", Kind: "news", URL: server.URL}}
	out, warnings, err := runApp(a, []string{"schedule", "refresh"}, "")
	if err != nil || warnings != "" || !strings.Contains(out, "Scheduled refresh complete") || strings.Contains(out, "Alex") {
		t.Fatalf("refresh-only result: %q %q %v", out, warnings, err)
	}
	after, err := a.Store.cache()
	if err != nil || after.GreetingVisits != 12 || strings.Join(after.Recent, ",") != "previous" || len(after.Sources["docker"].Items) != 1 {
		t.Fatalf("refresh altered history or lost news: %+v %v", after, err)
	}
	fail = true
	_, warnings, err = runApp(a, []string{"schedule", "refresh"}, "")
	if err == nil || !strings.Contains(warnings, "503") {
		t.Fatalf("failed source hidden: %q %v", warnings, err)
	}
	after, _ = a.Store.cache()
	if len(after.Sources["docker"].Items) != 1 || after.GreetingVisits != 12 {
		t.Fatal("failed refresh lost cache/display state")
	}
	lock, err := a.Store.lock(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	_, _, err = runApp(a, []string{"schedule", "refresh"}, "")
	if !errors.Is(err, errStateBusy) {
		t.Fatalf("busy scheduler did not fail explicitly: %v", err)
	}
}

func TestSchedulePlistAndOwnership(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "refresh.plist")
	content := schedulePlist("/tmp/space & quotes\"/opsmosaic", Store{"/tmp/config", "/tmp/cache"}, "/tmp/log")
	if !strings.Contains(content, "<key>Label</key><string>io.github.d-kumar19.opsmosaic.refresh</string>") {
		t.Fatal("launch agent must use the personal project identifier")
	}
	os.WriteFile(path, []byte(content), 0600)
	if err := validateScheduleFile(path); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		if output, err := exec.Command("/usr/bin/plutil", "-lint", path).CombinedOutput(); err != nil {
			t.Fatalf("invalid launchd plist: %s %v", output, err)
		}
	}
	os.WriteFile(path, []byte("<plist>unmanaged</plist>"), 0600)
	if err := validateScheduleFile(path); err == nil {
		t.Fatal("accepted unmanaged launch agent")
	}
}

func TestIntegrationCLILifecycleAndErrors(t *testing.T) {
	a := newTestApp(t)
	path := filepath.Join(t.TempDir(), "rc")
	for _, action := range []string{"install", "install", "uninstall", "uninstall"} {
		if _, _, err := runApp(a, []string{"shell", action, "zsh", path}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.Run(context.Background(), []string{"shell", "install", "bash", path},
		strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
		t.Fatal("shell output failure hidden")
	}
	os.WriteFile(a.Store.ConfigPath, []byte("invalid"), 0600)
	if _, _, err := runApp(a, []string{"shell", "greeting"}, ""); err == nil {
		t.Fatal("corrupt profile hidden")
	}
	t.Setenv("HOME", t.TempDir())
	for _, action := range []string{"status", "disable"} {
		_, _, err := runApp(a, []string{"schedule", action}, "")
		if runtime.GOOS == "darwin" && err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "darwin" && err == nil {
			t.Fatal("unsupported scheduler accepted")
		}
	}
	if _, _, err := runApp(a, []string{"schedule", "enable"}, ""); err == nil {
		t.Fatal("scheduler accepted corrupt/missing profile")
	}
}

func TestScheduledRefreshEmptyCorruptAndCancelled(t *testing.T) {
	a := newTestApp(t)
	writeJSON(a.Store.ConfigPath, Profile{Version: 1, Name: "Alex", Topics: []string{"docker"}})
	if _, _, err := runApp(a, []string{"schedule", "refresh"}, ""); err == nil {
		t.Fatal("empty news treated as successful refresh")
	}
	os.WriteFile(a.Store.CachePath, []byte("invalid"), 0600)
	if _, _, err := runApp(a, []string{"schedule", "refresh"}, ""); err == nil {
		t.Fatal("corrupt cache hidden")
	}
	before, _ := os.ReadFile(a.Store.CachePath)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Run(ctx, []string{"schedule", "refresh"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("cancelled scheduler succeeded")
	}
	after, _ := os.ReadFile(a.Store.CachePath)
	if string(before) != string(after) {
		t.Fatal("cancelled scheduler modified cache")
	}
}
