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
