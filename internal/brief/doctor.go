package brief

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type doctorCheck struct {
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type doctorSource struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ItemCount    int    `json:"item_count"`
	AgeSeconds   *int64 `json:"oldest_age_seconds"`
	RetryPending bool   `json:"retry_pending"`
}

type doctorPaths struct {
	Executable string `json:"executable"`
	Config     string `json:"config"`
	Cache      string `json:"cache"`
}

type doctorReport struct {
	SchemaVersion  int            `json:"schema_version"`
	Status         string         `json:"status"`
	Version        string         `json:"version"`
	Commit         string         `json:"commit"`
	BuildTime      string         `json:"build_time"`
	OS             string         `json:"os"`
	Arch           string         `json:"arch"`
	GoVersion      string         `json:"go_version"`
	Paths          doctorPaths    `json:"paths"`
	Checks         []doctorCheck  `json:"checks"`
	Sources        []doctorSource `json:"sources"`
	UnknownSources int            `json:"unknown_cached_sources"`
}

func doctorPath(path, home string) string {
	path = filepath.Clean(path)
	if home != "" {
		home = filepath.Clean(home)
		if relative, err := filepath.Rel(home, path); err == nil {
			if relative == "." {
				return "~"
			}
			if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return cleanText(filepath.Join("~", relative))
			}
		}
	}
	return cleanText(path)
}

func (r *doctorReport) check(code, status, message string) {
	r.Checks = append(r.Checks, doctorCheck{code, status, message})
	if status == "error" || (status == "warning" && r.Status == "ok") {
		r.Status = status
	}
}

func doctorStateFile(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "missing"
	}
	if err != nil || !info.Mode().IsRegular() {
		return "unreadable"
	}
	file, err := os.Open(path)
	if err != nil {
		return "unreadable"
	}
	if err := file.Close(); err != nil {
		return "unreadable"
	}
	return "present"
}

func sourceDiagnosis(id string, entry SourceCache, now time.Time) doctorSource {
	source := doctorSource{ID: id, Status: "fresh", ItemCount: len(entry.Items),
		RetryPending: entry.RetryAfter.After(now)}
	timestamps := []time.Time{entry.Fetched}
	if len(entry.Items) != 0 {
		timestamps = make([]time.Time, 0, len(entry.Items))
		for _, item := range entry.Items {
			fetched := item.FetchedAt
			if fetched.IsZero() {
				fetched = entry.Fetched
			}
			timestamps = append(timestamps, fetched)
		}
	}
	var oldest int64
	for _, fetched := range timestamps {
		if fetched.IsZero() || fetched.After(now) {
			source.Status = "unknown"
			return source
		}
		age := now.Sub(fetched)
		if age >= cacheTTL {
			source.Status = "stale"
		}
		if seconds := int64(age / time.Second); seconds > oldest {
			oldest = seconds
		}
	}
	source.AgeSeconds = &oldest
	return source
}

func (a *App) diagnose() doctorReport {
	report := doctorReport{
		SchemaVersion: 1, Status: "ok",
		Version: cleanText(Version), Commit: cleanText(Commit), BuildTime: cleanText(BuildTime),
		OS: runtime.GOOS, Arch: runtime.GOARCH, GoVersion: runtime.Version(),
		Checks: []doctorCheck{}, Sources: []doctorSource{},
	}
	home, err := os.UserHomeDir()
	if err != nil {
		report.check("home_unavailable", "error", "Cannot locate home directory for path redaction.")
	}
	executable, err := os.Executable()
	if err != nil {
		report.check("executable_unavailable", "error", "Cannot locate the running executable.")
	} else {
		report.Paths.Executable = doctorPath(executable, home)
	}
	report.Paths.Config = doctorPath(a.Store.ConfigPath, home)
	report.Paths.Cache = doctorPath(a.Store.CachePath, home)
	if _, err := a.Store.saved(); err != nil {
		report.check("saved_invalid", "error", "Saved-item state is invalid or unreadable; inspect saved.json locally.")
	}
	custom, customErr := a.Store.customFeeds()
	if customErr != nil {
		report.check("feeds_invalid", "error", "Custom feed configuration is invalid or unreadable; inspect feeds.json locally.")
	} else if len(custom.Feeds) != 0 {
		report.check("feeds_valid", "ok", fmt.Sprintf("%d custom feeds configured; names, topics, URLs and IDs omitted.", len(custom.Feeds)))
	}
	profileStatus := doctorStateFile(a.Store.ConfigPath)
	switch profileStatus {
	case "missing":
		report.check("profile_missing", "warning", "No profile; run opsmosaic manually to complete onboarding.")
	case "unreadable":
		report.check("profile_unreadable", "error", "Profile is unreadable or not a regular file; inspect its permissions and type.")
	default:
		if _, err := a.Store.profile(); err != nil {
			// Never print validation/parser errors: they can contain private input.
			report.check("profile_invalid", "error", "Profile is invalid or became unreadable; inspect locally before resetting it.")
		} else {
			report.check("profile_valid", "ok", "Profile schema and values are valid; personal values omitted.")
		}
	}
	switch doctorStateFile(a.Store.CachePath) {
	case "missing":
		report.check("cache_missing", "warning", "No cache; bundled tips still work. Refresh manually when online.")
		return report
	case "unreadable":
		report.check("cache_unreadable", "error", "Cache is unreadable or not a regular file; inspect its permissions and type.")
		return report
	}
	cache, err := a.Store.cache()
	if err != nil {
		report.check("cache_invalid", "error", "Cache is invalid or became unreadable; inspect locally before rebuilding it.")
		return report
	}
	report.check("cache_valid", "ok", "Cache schema is valid; item content and URLs omitted.")
	now := a.Now()
	known := map[string]bool{}
	// Report the fixed built-in inventory, not profile interests or arbitrary stored IDs.
	for _, source := range DefaultSources() {
		known[source.ID] = true
		entry, exists := cache.Sources[source.ID]
		diagnosis := doctorSource{ID: source.ID, Status: "not_cached"}
		if exists {
			diagnosis = sourceDiagnosis(source.ID, entry, now)
			if diagnosis.Status == "stale" || diagnosis.Status == "unknown" {
				report.check("cache_"+diagnosis.Status, "warning", "A built-in source has stale or unknown-age cache; refresh manually when online.")
			}
		}
		report.Sources = append(report.Sources, diagnosis)
	}
	if customErr == nil {
		for i, feed := range custom.Feeds {
			id := feed.source().ID
			known[id] = true
			safeID := fmt.Sprintf("custom-%d", i+1)
			diagnosis := doctorSource{ID: safeID, Status: "not_cached"}
			if entry, exists := cache.Sources[id]; exists {
				diagnosis = sourceDiagnosis(safeID, entry, now)
				if diagnosis.Status == "stale" || diagnosis.Status == "unknown" {
					report.check("cache_"+diagnosis.Status, "warning", "A custom source has stale or unknown-age cache; refresh manually when online.")
				}
			}
			report.Sources = append(report.Sources, diagnosis)
		}
	}
	for id := range cache.Sources {
		if !known[id] {
			report.UnknownSources++
		}
	}
	if len(cache.Sources) == 0 {
		report.check("cache_empty", "warning", "No cached sources; bundled tips remain available.")
	}
	return report
}

func (a *App) doctor(ctx context.Context, args []string, out, warnings io.Writer) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(warnings)
	jsonOutput := flags.Bool("json", false, "emit a versioned JSON report")
	var usageErr error
	flags.Usage = func() {
		_, usageErr = fmt.Fprintln(out, "Usage: opsmosaic doctor [--json]\nRead-only local checks; no network, setup, lock, repair or state writes.")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return usageErr
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("doctor accepts only --json; use opsmosaic doctor --help")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	report := a.diagnose()
	if *jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return fmt.Errorf("write doctor report: %w", err)
		}
	} else {
		var text strings.Builder
		fmt.Fprintf(&text, "OpsMosaic doctor: %s (local checks only)\n", report.Status)
		fmt.Fprintf(&text, "Version: %s | commit %s | built %s\n", report.Version, report.Commit, report.BuildTime)
		fmt.Fprintf(&text, "Platform: %s/%s | %s\n", report.OS, report.Arch, report.GoVersion)
		fmt.Fprintf(&text, "Executable: %s\nConfig: %s\nCache: %s\n", report.Paths.Executable, report.Paths.Config, report.Paths.Cache)
		for _, check := range report.Checks {
			fmt.Fprintf(&text, "%s [%s]: %s\n", check.Status, check.Code, check.Message)
		}
		for _, source := range report.Sources {
			age := "unknown"
			if source.AgeSeconds != nil {
				age = fmt.Sprintf("%ds", *source.AgeSeconds)
			}
			fmt.Fprintf(&text, "Source %s: %s | items %d | oldest age %s | retry pending %t\n",
				source.ID, source.Status, source.ItemCount, age, source.RetryPending)
		}
		fmt.Fprintf(&text, "Unknown cached source IDs (names omitted): %d\n", report.UnknownSources)
		fmt.Fprintln(&text, "Cached status is not a live source-health check. Paths outside home may reveal directory names.")
		if _, err := io.WriteString(out, text.String()); err != nil {
			return fmt.Errorf("write doctor report: %w", err)
		}
	}
	if report.Status == "error" {
		return fmt.Errorf("doctor found local state errors; see report (nothing changed)")
	}
	return nil
}
