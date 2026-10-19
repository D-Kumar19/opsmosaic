package brief

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestVersionWithoutState(t *testing.T) {
	a := newTestApp(t)
	out, warnings, err := runApp(a, []string{"--version"}, "")
	if err != nil || warnings != "" || out != "opsmosaic dev (commit unknown; built unknown)\n" {
		t.Fatalf("development version: %q %q %v", out, warnings, err)
	}
	for _, path := range []string{a.Store.ConfigPath, a.Store.CachePath, a.Store.ConfigPath + ".lock"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("version created state: %s", path)
		}
	}
	if err := a.Run(context.Background(), []string{"--version"}, strings.NewReader(""), failedWriter{}, failedWriter{}); err == nil {
		t.Fatal("version output error was hidden")
	}
}

func TestVersionMetadataAndInvalidArguments(t *testing.T) {
	oldVersion, oldCommit, oldTime := Version, Commit, BuildTime
	t.Cleanup(func() { Version, Commit, BuildTime = oldVersion, oldCommit, oldTime })
	Version, Commit, BuildTime = "v0.1.0", "abc123", "2026-10-19T10:00:00Z"
	a := newTestApp(t)
	out, _, err := runApp(a, []string{"--version"}, "")
	if err != nil || out != "opsmosaic v0.1.0 (commit abc123; built 2026-10-19T10:00:00Z)\n" {
		t.Fatalf("release version: %q %v", out, err)
	}
	for _, args := range [][]string{
		{"--version", "extra"}, {"--version", "--tips"},
		{"config", "--version"}, {"refresh", "--version"}, {"--version", "--unknown"},
	} {
		if _, _, err := runApp(a, args, ""); err == nil {
			t.Errorf("accepted invalid version arguments: %v", args)
		}
	}
}
