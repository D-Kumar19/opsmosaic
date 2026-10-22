package brief

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const hookStart = "# >>> opsmosaic managed hook >>>"
const hookEnd = "# <<< opsmosaic managed hook <<<"

func shellHook() string {
	return hookStart + `
case $- in
  *i*)
    if [ -t 1 ] && command -v opsmosaic >/dev/null 2>&1; then
      opsmosaic shell greeting
    fi
    ;;
esac
` + hookEnd + "\n"
}

func editHook(path string, install bool) error {
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("shell configuration must be a regular file, not a symlink")
	}
	content := []byte{}
	mode := os.FileMode(0600)
	if err == nil {
		mode = info.Mode().Perm()
		content, err = os.ReadFile(path)
		if err != nil {
			return err
		}
	}
	text := string(content)
	start, end := strings.Index(text, hookStart), strings.Index(text, hookEnd)
	if strings.Count(text, hookStart) > 1 || strings.Count(text, hookEnd) > 1 ||
		(start >= 0) != (end >= 0) || (start >= 0 && end < start) {
		return fmt.Errorf("malformed OpsMosaic hook markers; configuration left unchanged")
	}
	if start >= 0 {
		text = text[:start] + text[end+len(hookEnd):]
		// The managed block owns its trailing newline only.
		if strings.HasPrefix(text[start:], "\n") {
			text = text[:start] + text[start+1:]
		}
	}
	if install {
		if text != "" && !strings.HasSuffix(text, "\n") {
			return fmt.Errorf("configuration needs a final newline before installing; left unchanged")
		}
		text += shellHook()
	}
	if text == string(content) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	if info != nil {
		backup := path + ".opsmosaic-backup"
		f, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err == nil {
			_, writeErr := f.Write(content)
			closeErr := f.Close()
			if writeErr != nil {
				return writeErr
			}
			if closeErr != nil {
				return closeErr
			}
		} else if !os.IsExist(err) {
			return err
		}
	}
	return atomicText(path, []byte(text), mode)
}

func atomicText(path string, data []byte, mode os.FileMode) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".opsmosaic-integration-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(f.Name()); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
			err = fmt.Errorf("remove integration temporary file: %w", removeErr)
		}
	}()
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (a *App) integration(ctx context.Context, args []string, in io.Reader, out, warnings io.Writer) error {
	if args[0] == "shell" {
		if len(args) == 2 && args[1] == "greeting" {
			if _, err := a.Store.profile(); os.IsNotExist(err) {
				return nil
			} else if err != nil {
				return err
			}
			return a.Run(ctx, []string{"--offline"}, in, out, warnings)
		}
		if len(args) != 4 || (args[1] != "install" && args[1] != "uninstall") ||
			(args[2] != "zsh" && args[2] != "bash") {
			return fmt.Errorf("usage: opsmosaic shell install|uninstall zsh|bash FILE; or shell greeting")
		}
		if err := editHook(args[3], args[1] == "install"); err != nil {
			return fmt.Errorf("edit shell hook: %w", err)
		}
		_, err := fmt.Fprintln(out, "Shell hook "+args[1]+" complete. Existing unmanaged hooks are not modified.")
		return err
	}
	if len(args) == 2 && args[1] == "refresh" {
		return a.scheduledRefresh(ctx, out, warnings)
	}
	if len(args) != 2 || (args[1] != "enable" && args[1] != "disable" && args[1] != "status") {
		return fmt.Errorf("usage: opsmosaic schedule enable|disable|status")
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("scheduled refresh supports macOS launchd only")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	store := a.Store
	store.ConfigPath, err = filepath.Abs(store.ConfigPath)
	if err != nil {
		return err
	}
	store.CachePath, err = filepath.Abs(store.CachePath)
	if err != nil {
		return err
	}
	return manageSchedule(ctx, args[1], home, executable, store, os.Getuid(), out,
		func(ctx context.Context, args ...string) error {
			result, err := exec.CommandContext(ctx, "/bin/launchctl", args...).CombinedOutput()
			if err != nil {
				return fmt.Errorf("launchctl %s: %w: %s", args[0], err, cleanText(string(result)))
			}
			if args[0] == "print" {
				_, err = fmt.Fprintln(out, cleanText(string(result)))
				return err
			}
			return nil
		})
}

const scheduleLabel = "io.github.d-kumar19.opsmosaic.refresh"

func schedulePlist(executable string, store Store, log string) string {
	escape := html.EscapeString
	// Explicit paths preserve XDG/custom state even in launchd's minimal environment.
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>` + scheduleLabel + `</string>
<key>ProgramArguments</key><array><string>` + escape(executable) + `</string><string>schedule</string><string>refresh</string></array>
<key>EnvironmentVariables</key><dict>
<key>OPSMOSAIC_CONFIG_PATH</key><string>` + escape(store.ConfigPath) + `</string>
<key>OPSMOSAIC_CACHE_PATH</key><string>` + escape(store.CachePath) + `</string>
</dict>
<key>StartInterval</key><integer>10800</integer>
<key>Umask</key><integer>63</integer>
<key>StandardOutPath</key><string>` + escape(log) + `</string>
<key>StandardErrorPath</key><string>` + escape(log) + `</string>
</dict></plist>
`
}

type launchRunner func(context.Context, ...string) error

func manageSchedule(ctx context.Context, action, home, executable string, store Store, uid int, out io.Writer, run launchRunner) error {
	path := filepath.Join(home, "Library", "LaunchAgents", scheduleLabel+".plist")
	domain := fmt.Sprintf("gui/%d", uid)
	if action == "status" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_, err := fmt.Fprintln(out, "Scheduled refresh is not configured.")
			return err
		} else if err != nil {
			return err
		}
		if err := validateScheduleFile(path); err != nil {
			return err
		}
		if err := run(ctx, "print", domain+"/"+scheduleLabel); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, "Scheduled refresh is loaded (every 3 hours). Logs: "+filepath.Join(home, "Library", "Logs", "opsmosaic", "refresh.log"))
		return err
	}
	if action == "disable" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_, err := fmt.Fprintln(out, "Scheduled refresh is already disabled.")
			return err
		} else if err != nil {
			return err
		}
		if err := validateScheduleFile(path); err != nil {
			return err
		}
		if err := run(ctx, "bootout", domain, path); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		_, err := fmt.Fprintln(out, "Scheduled refresh disabled; logs and preferences retained.")
		return err
	}
	if _, err := store.profile(); err != nil {
		return fmt.Errorf("complete onboarding before enabling refresh: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		if err := validateScheduleFile(path); err != nil {
			return err
		}
		return manageSchedule(ctx, "status", home, executable, store, uid, out, run)
	} else if !os.IsNotExist(err) {
		return err
	}

	log := filepath.Join(home, "Library", "Logs", "opsmosaic", "refresh.log")
	for _, directory := range []string{filepath.Dir(path), filepath.Dir(log)} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return err
		}
	}
	if err := atomicText(path, []byte(schedulePlist(executable, store, log)), 0600); err != nil {
		return err
	}
	if err := run(ctx, "bootstrap", domain, path); err != nil {
		if removeErr := os.Remove(path); removeErr != nil {
			return fmt.Errorf("%w; remove failed configuration: %v", err, removeErr)
		}
		return err
	}
	_, err := fmt.Fprintln(out, "Scheduled refresh enabled every 3 hours; no immediate refresh. Logs: "+log)
	return err
}

func validateScheduleFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing non-regular schedule configuration")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(content), "<key>Label</key><string>"+scheduleLabel+"</string>") {
		return fmt.Errorf("existing schedule configuration is not managed by OpsMosaic; left unchanged")
	}
	return nil
}

func (a *App) scheduledRefresh(ctx context.Context, out, warnings io.Writer) (runErr error) {
	lock, err := a.Store.lock(ctx, 250*time.Millisecond)
	if err != nil {
		return fmt.Errorf("scheduled refresh cannot acquire state lock: %w", err)
	}
	defer func() { runErr = errors.Join(runErr, lock.Close()) }()
	profile, err := a.Store.profile()
	if err != nil {
		return fmt.Errorf("scheduled refresh requires an existing valid profile: %w", err)
	}
	cache, err := a.Store.cache()
	if err != nil {
		return err
	}
	now := a.Now()
	app, err := a.withCustomFeeds()
	if err != nil {
		return err
	}
	items, notes := app.news(ctx, &cache, profile.Topics, now, true, false)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := writeJSON(a.Store.CachePath, cache); err != nil {
		return err
	}
	for _, note := range notes {
		if _, err := fmt.Fprintln(warnings, now.UTC().Format(time.RFC3339), "Warning:", cleanText(note)); err != nil {
			return err
		}
	}
	if len(notes) != 0 || len(items) == 0 {
		return fmt.Errorf("scheduled refresh incomplete: %d source warnings, %d available items; retained usable cache", len(notes), len(items))
	}
	_, err = fmt.Fprintln(out, now.UTC().Format(time.RFC3339), "Scheduled refresh complete; cached items:", len(items))
	return err
}
