# Test cases

Run `go test ./...` for hermetic tests, `go test -race -cover ./...` for
race detection/coverage and `go vet ./...` for static checks.
Run `make test-install` to verify copying to a custom prefix, executable
permissions, PATH lookup outside the checkout, staged installation with
`DESTDIR`, first-run setup, explicit count, and invalid-option exit status.
Install tests use disposable directories, including a prefix containing spaces.
Run `make ci` for the combined formatting/module/race/vet/build/install checks.
Run `make test-release` for an isolated committed fixture: four target archives,
embedded metadata in every binary, exact bundled documentation/dependency
licenses, checksums and deliberate tampering, native version/onboarding smoke,
invalid versions, dirty-tree/overwrite guards, failed-build cleanup and a
deliberately unformatted file rejected by the CI entry point.

Phase 3 local verification used Go 1.27.2 on macOS arm64: 107 passing Go test
cases (including subtests), zero failures/skips, 94.1% core statement coverage.
Only macOS arm64 execution was verified; other targets were cross-built.
Go 1.24 and the remote Go/OS matrix remain unverified pending tooling/runners.
See [CI readiness](ci.md); do not interpret local success as remote CI.

Hosted run 39752877495 subsequently passed Linux Go 1.24.x/stable. Its macOS
failure revealed an invalid timing assumption in the concurrent bookmark test:
contenders may legitimately receive `errStateBusy` after the bounded wait.
The corrected test checks retention of every successful write, resubmits
explicitly rejected writes after contention ends, verifies all expected bookmark
identities, and requires a held lock to report `errStateBusy`. It passed 50
race-enabled local repetitions and the full local CI entry point. A fully green
hosted matrix is still required; production lock timeouts were not increased.

Phase 4 adds `make test-demo` (five Python standard-library cases) and
`make demo` (macOS-only media generation). The recorder tests prompt/reply
handling, command failures, missing prompts, timeouts and output validation.
Actual captures assert one tip, example name/UTC, no escape sequences and no
repeat onboarding offline. The renderer checks text fits, GIF finalization,
three decoded 1100x880 frames and playback timing. These tests are separate
from the 107-case Go baseline. A real GIF/transcript was generated and reviewed
using disposable state; no personal profile or live feed requests were used.
See [demo instructions](demo/README.md).

## Shell, scheduling and signing follow-up

Latest local result: **119 passing Go cases**, no failures/skips, race/vet passed,
**90.4% core statement coverage**. The prior 94.1% figure above is the historical
Phase 3 result, not the current expanded codebase's coverage.

- Hook tests: repeated install/removal, original backup/content/permissions,
  malformed markers, symlink rejection, profile-aware skip and visible errors.
- `make test-shell`: real bash/zsh interactive and noninteractive PTYs,
  missing profile/executable, visible process failures and uninstall, all
  with temporary shell files/state. Startup measurements are local samples.
- Scheduler tests: macOS `plutil` syntax, three-hour interval/private umask,
  escaped paths and pinned state, simulated launchctl enable/status/disable,
  bootstrap rollback/bootout failure, unmanaged-file rejection.
- Cache-only scheduled refresh: local HTTP success/failure/cache retention,
  unchanged history/greeting visits, busy lock, cancellation, corrupt/empty
  state and no onboarding.
- `make test-signing`: temporary keys, trusted signature, unsigned/wrong-key
  rejection, manifest/archive tampering, unsafe path rejection and overwrite
  protection. No real release key or signature is created.
- `make ci`, `make test-release` and standalone reinstall passed.

Real launchd activation was intentionally not tested or enabled on the shared
machine. Remote CI/native non-macOS-arm64 verification and production signing
remain outstanding; these are not represented by the simulated tests.

## Doctor follow-up

Latest baseline: **126 passing Go cases**, zero failures/skips, **91.0% core
coverage**, race/vet/install checks and four-target release tests passed.
Seven doctor tests cover missing/corrupt/unreadable state, schema versions,
stale/future/unknown and partial-merge ages, exact JSON top-level keys,
privacy sentinels, home-path boundaries, errors/cancellation/help/output
failures, no network and no lock/state writes. Tests run against isolated
state, including while another transaction holds its lock. Standalone install
tests cover text and JSON diagnostics without onboarding.

## Custom-feed follow-up

Latest baseline: **137 passing Go cases**, no failures/skips, **90.7% core
coverage**; race/vet/install, four-target release, real shell and signing tests
passed. Eleven custom-feed tests cover commands/config/schema, duplicate/count
limits, permissions and legacy profile preservation, concurrent edits, full
definition cache identity, removal pruning, shared TTL/failure/topic behavior,
offline custom news/releases and doctor privacy.

Public URL/address tests include local/private/special-use destinations,
credentials, queries, fragments and alternate ports. The connection tests use
a local trusted TLS server with injected public DNS/dial boundaries; they check
IP pinning, empty/mixed/private DNS, redirect downgrade/private targets/query
strings/loops, connection/DNS failures and cancellation. RSS/Atom parsing and
unsafe-link/sanitization use local fixtures. There are no live custom-source
requests or private-network probes in tests.

Standalone installation tests exercise feed add/list/remove and diagnostic
validation using a disposable profile; those commands do not fetch.

## Stable-release filtering follow-up

Latest baseline: **142 passing Go cases**, no failures/skips, **90.9% core
coverage**, race/vet/install and four-target release checks passed.
Five focused tests cover stable SemVer and build metadata, supported component
prefixes, alpha/beta/RC/nightly and unknown labels, explicit tag precedence
through actual Atom parsing (including escaped component tags), custom offline
filtering and unchanged defaults/cache, news/tip preservation, empty-result
errors/fallback, invalid flag combinations and refresh cache retention.
The standalone installation test exercises `--tips --stable-only`.

| Area | Cases |
|---|---|
| Onboarding | Name + multiple interests, invalid input retries, deduplicated selections, EOF |
| Profiles | Empty/long/unsafe names, missing/unknown/duplicate topics, schema version |
| Storage | Round trip, nested directories, 0600 permissions, corrupt/trailing JSON |
| RSS/Atom | Headline, publication time, alternate links, unsafe links, malformed XML, unsupported format |
| HTTP | Status errors, response cap, cancellation/timeouts |
| Hacker News | AI keyword matching, unrelated/dead/deleted stories, malformed data, item failures |
| Tips | Every topic has a sourced tip; no automatic execution |
| Selection | Random kind/item selection, every kind reachable, single-item default, mode/topic filters, explicit multi-item count, fair topic order, history rotation, URL deduplication, unseen preference, 100-item history cap |
| CLI | First/repeat run, local time greeting, config replacement safety, help without profile |
| Options | Unknown topic/flag/command, conflicting modes, 1-20 limits, topic override persistence |
| Cache | Six-hour boundary, future clock, empty-feed TTL, fresh cache, refresh, failed-source retention |
| Offline | No network for tips/offline, explicit empty-news failure, corrupt-cache failure |
| Rendering | Control/ANSI/OSC stripping, markup decoding, safe URLs, output-writer failures |
| Installation | Standalone PATH command, custom/spaced prefix, DESTDIR staging, no setup on help, onboarding and count/error options |
| Greetings | All hour boundaries, device timezone, sanitized name, three-comment rotation and wrap, persisted visits across App instances, legacy-cache compatibility |
| Partial HN | Successful items survive HTTP/malformed/timeout failures, merge deduplication and 100-item cap, preserved old ages, 15-minute retry boundary, explicit refresh bypass, empty partials, full recovery and total-failure retention |
| Kind variety | Seen tips stay eligible alongside unseen news; unseen preference applies within a kind; equal kind selection remains deterministic in tests |
| State locking | Concurrent CLI/test subprocesses preserve updates and source data; interrupted lock holder releases lock; 250 ms busy fallback never fetches/writes; busy config/setup errors and cancellation |
| Freshness | Before/at six-hour boundary, old/missing/future retrieval dates, truthful labels after failed refresh, exactly one selected-stale hint, no hint for unselected stale items |
| Display | Compact/detailed shape, default one card, Unicode wrapping at 4/12/20/40/80/100 columns, long names, CJK/combining/ZWJ emoji, unchanged commands and URLs, 80-column fallback |
| Output policy | TTY/non-TTY/dumb/unknown terminals; plain overrides color; nonempty NO_COLOR (including zero) beats always; never disables color; redirected output keeps text greetings without ANSI/decorations |
| Version | Dev defaults, explicit release metadata, no profile/cache/lock creation, no network, invalid combinations, output-writer failures |
| Release | Four archives, metadata in all targets, native command smoke, documentation/license fidelity, checksum verification/tampering, invalid version/dirty tree/overwrite rejection, failed-build cleanup |

## Bookmarks and replay follow-up (T039)

Latest baseline: **153 passing Go cases**, no failures/skips, **91.5% core
statement coverage**. `make ci` and four-target `make test-release` pass.
Eleven saved-state tests cover complete snapshots, private permissions, single/
multi-item numbering, empty history, duplicates and the 100-item no-eviction
limit, canonical article versus tip identity, concurrent writers, held locks,
removed profile/cache/sources, preserved original timestamps/source/link, and
control-sequence sanitization. They also cover invalid schema/content/paths,
the 16 MiB bound, output/cancellation failures, unchanged state on replay,
normal bookmark retention, busy fallback and successful local scheduled refresh.
Doctor detects invalid saved state without exposing a private sentinel.
Standalone installation checks replay plus add/list/show/remove outside checkout.
All fixtures use temporary state and local HTTP only.

## Interactive topic picker follow-up (T040)

Latest baseline: **157 passing Go cases**, zero failures/skips, **87.7% core
statement coverage**. Local CI/race/vet/build/install, four-target packaging,
five demo-tool cases and real bash/zsh PTY integration passed. Picker unit tests
cover navigation/wraparound, toggling, empty-confirm validation, cancellation,
key sequences, scrolling, sanitized names, the numbered fallback and output
shape. PTY checks drive arrow/Space/Enter, resize to a short terminal mid-choice,
confirm the selected topics, cancel a later config without altering the saved
profile, and verify canonical input/echo restoration. All profiles and shell
files are disposable; no personal configuration was used.

## AI-summary evaluation follow-up (T041)

Evaluation-only decision; no AI provider, credentials, runtime behavior or
external data sharing were added. No new runtime test cases apply. Existing
headline, attribution/link, terminal-sanitization and offline tests remain the
relevant regression coverage; the measured Go baseline remains 157 passing
cases with 87.7% core statement coverage.

## Manual acceptance

Use a temporary `OPSMOSAIC_HOME`, never your real profile, for smoke tests.

1. Run `--tips`, enter a name and `2,3,4,9`; check greeting and tip.
2. Repeat `--tips` without input; no onboarding and one random unseen tip when
   available. The same topic may appear again; topics are not cycled by default.
3. Run `refresh --news --count 9`; each available selected topic has a sourced
   item and a link. Network errors must be warnings, not hidden.
4. Run `--offline --news`; cached labels appear, no network requests occur.
5. Run `--topic helm --tips`; the original saved interests remain unchanged.
6. Run `--count 0`; exit is nonzero with a useful error.
7. Run without options; exactly one tip, news item, or release appears.
   Repeat using fresh cache to check variety without requiring any particular
   random sequence. Automated tests force every random branch deterministically.

Live source availability is not a hermetic test assertion. Recheck it during
release verification, recording unavailable sources separately.
