# Optional shell greeting and scheduled refresh

Neither installation nor normal use edits shell configuration or enables jobs.
These commands are explicit opt-ins. Your existing personal hook is unchanged.

## Shell greeting (zsh/bash)

First complete onboarding by running `opsmosaic` manually. Then choose the file
your interactive shell actually reads. For example:

```sh
opsmosaic shell install zsh "$HOME/.zshrc"
# OR for interactive non-login bash:
opsmosaic shell install bash "$HOME/.bashrc"
# Login bash users may instead choose ~/.bash_profile.
```

The command requires an explicit shell and file; it does not guess or source it.
The same guarded hook works in both shells. It runs `opsmosaic shell greeting`
only in interactive shells with terminal stdout and an executable on PATH.
That command skips missing profiles without onboarding, otherwise uses offline
mode. Corrupt state and startup failures remain visible; it never clears output
or fetches feeds. Every new terminal gets a greeting; there is no hourly/daily
suppression. Explicit `opsmosaic` commands keep their normal behavior.

Before a changed existing file is first written, its original content is saved
as `FILE.opsmosaic-backup` (mode 0600). That backup is never overwritten.
Files are replaced atomically with original permissions. Repeated installs
produce one managed block; uninstall removes only that block:

```sh
opsmosaic shell uninstall zsh "$HOME/.zshrc"
```

Symlinks, duplicate/unbalanced markers and files without a final newline are
rejected rather than edited ambiguously. Existing unmanaged OpsMosaic hooks
are not removed. If you already added one manually, review it yourself before
opting in, to avoid duplicate greetings. Do not run simultaneous installers or
edit the same configuration while installing; backups can contain private
settings, so never commit or upload them.

Verification: `make test-shell` uses isolated files and real bash/zsh PTYs.
Local missing-profile startup samples were approximately 19–22 ms including
shell/process startup, not a performance guarantee. A separate Go test logs
the in-process check cost; no machine-independent time threshold is claimed.
Python 3 is a test helper, not an installed CLI requirement.

## macOS background refresh (every three hours)

Use the installed standalone executable, not a temporary build. Complete
onboarding and ensure your chosen topics/providers are permitted on your network.
Only run enable when you actually want background requests:

```sh
opsmosaic schedule enable
opsmosaic schedule status
opsmosaic schedule disable
```

Enable creates a per-user LaunchAgent and loads it using `launchctl bootstrap`.
It schedules a 10,800-second interval, not an immediate run and not a wall-clock
guarantee. launchd controls timing while the user session is available; do not
expect refresh while the machine is asleep or logged out. There is no Linux
scheduler in this scope.

Paths:

- Agent: `~/Library/LaunchAgents/io.github.d-kumar19.opsmosaic.refresh.plist`
- Combined stdout/stderr log: `~/Library/Logs/opsmosaic/refresh.log`

Logs remain on disable; inspect and rotate/remove the log manually as needed.
No automatic log rotation is configured. Logs may contain feed URLs/headlines;
keep them private. The job uses umask 077 for newly created files.

The job pins the executable and current absolute config/cache paths (including
custom `OPSMOSAIC_HOME`/XDG paths). Internal paired
`OPSMOSAIC_CONFIG_PATH`/`OPSMOSAIC_CACHE_PATH` overrides preserve those locations
in launchd's minimal environment; both must be absolute. To move the executable
or change profiles, disable the old job before enabling from the new setup.
No shell aliases/functions or interactive credentials are inherited.

Scheduled refresh requires a valid existing profile, uses the normal bounded
state lock and source-fetch safeguards, and updates cache without printing a
briefing or changing recent-item history/greeting rotation. Busy/cancelled
runs fail explicitly; failed sources preserve usable cache, log warnings and
produce a nonzero result. Your shell hook stays network-free.

Repeated enable checks the existing loaded job rather than duplicating it;
repeated disable is a no-op after successful removal. An existing but unloaded
or otherwise broken job causes an explicit launchctl error instead of a false
success. Diagnose with `launchctl print gui/$(id -u)/io.github.d-kumar19.opsmosaic.refresh`
and the log. A foreign/symlink agent is never overwritten or removed. For an
unloaded stale managed file, inspect it before manually removing that exact
file and enabling again.

Tests validate plist syntax with macOS `plutil`, simulate launchctl lifecycle
and errors, and exercise real local HTTP/cache and concurrent-lock behavior.
No real LaunchAgent was enabled during development verification.
