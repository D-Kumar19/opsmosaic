# Implementation plan and TODOs

## Phase 1: context and contracts

- [x] T001: Record feature specification, principles, architecture and backlog.
- [x] T002: Write tests first for profiles, feeds, selection, CLI and storage.

## Phase 2: working first release

- [x] T003: Implement validated onboarding and atomic local storage.
- [x] T004: Implement tips, RSS/Atom/HN adapters and safe normalization.
- [x] T005: Implement caching, deduplication, fair selection and recent history.
- [x] T006: Wire CLI options, greeting, rendering, warnings and offline mode.

## Phase 3: verify and document

- [x] T007: Pass tests/race detector, vet, build and isolated smoke checks.
- [x] T008: Document usage/test cases and record implementation traceability.

Dependencies: T002 follows T001; T003/T004 follow T002; T005 follows T003/T004;
T006 follows T005; T007 follows T006; T008 follows T007.

## Requirement mapping

| Requirement | Tasks | Evidence |
|---|---|---|
| REQ-001 | T002, T003, T006 | onboarding/profile and CLI tests |
| REQ-002 | T002, T003, T006 | invalid/duplicate topics and override tests |
| REQ-003 | T002, T004, T006 | tip coverage and tips-only tests |
| REQ-004 | T002, T004, T005 | RSS/Atom/HN, fairness and URL dedup tests |
| REQ-005 | T002, T005, T006 | TTL, failure fallback, offline/cache tests |
| REQ-006 | T002, T005, T006 | selection/history, limits and CLI tests |
| REQ-007 | T002, T003, T004 | sanitization, link validation, permissions |
| REQ-008 | T002, T007, T008 | test/race/vet/build/smoke evidence |
| REQ-009 | T019, T020, T021, T022 | Unicode layout, output policy and CLI display tests |
| REQ-010 | T025, T026, T027, T028, T029, T030, T033 | Version tests, local CI and isolated release tests; remote runner/publication gates blocked |
| REQ-011 | T034, T035 | Actual isolated terminal GIF/transcript, demo-tool tests, project summary and draft notes; release gate remains blocked |
| REQ-012 | T023 | Hook lifecycle, backup/preservation, skip/error behavior and real bash/zsh PTY tests |
| REQ-013 | T024 | Three-hour plist, simulated launchctl lifecycle, cache-only fetch/failure/locking tests |
| REQ-015 | T037 | Read-only diagnostics, redaction/JSON contract, cache ages and standalone-command tests |
| REQ-016 | T036 | Additive feed commands, migration/cache/offline/concurrency tests and guarded TLS/DNS/redirect fixtures |
| REQ-017 | T038 | Version/tag classification, actual feed fixture, custom offline/refresh cache preservation, empty-result and installed flag tests |
| REQ-018 | T039 | Complete saved snapshots, numbering/duplicates/limits, offline replay, sanitization, locking, doctor privacy and installed commands |
| REQ-019 | T040 | PTY arrow/Space/Enter selection, live resize/scroll, cancellation, termios restoration and numbered fallback |

## Future backlog

The task list below supersedes the earlier informal backlog. Checkboxes track
progress; recording tasks does not approve a design, publication, or release.

### Working agreement

- Work on tasks only when requested. On 2026-10-19 the user authorized the
  first milestone (Reliability, T015-T018), then the second milestone
  (Display polish, T019-T022), and then release foundations
  (T025-T030, T033), then presentation materials (T034-T035).
  Optional tasks are authorized individually.
- Shell edits and scheduler activation on the actual machine remain opt-in.
  Homebrew and production signing are outside the project scope and are not
  tracked as future work.
- Before implementing a task, ask the user about unresolved behavior, scope,
  or design decisions using the clarification form. Do not silently choose
  between materially different approaches.
- Ask the task's listed questions when that task is selected, not all at once.
  Record the answers here and update the relevant specification before coding.
- Every implementation task includes focused regression/acceptance tests and
  directly related documentation. Verify the actual behavior, not only coverage.
- Keep one content item by default; never execute displayed commands.
- Do not publish, push, create release artifacts remotely, or install scheduled
  jobs without explicit approval. Never assume ownership/publication permission.

Priority: P1 = correctness/release foundations; P2 = release polish;
P3 = optional enhancements. Dependencies name actual prerequisites, not a
mandatory global execution order. Alternatives can remain unimplemented.

### Reliability and correctness

- [x] **T015 [P1] Preserve partial Hacker News results.**
  Keep valid stories when individual requests fail, expose a partial-failure
  warning, and retain cached data appropriately on total failure.
  Test mixed success/failure, malformed stories, timeout/cancellation, all
  failures, and cache behavior.
  **Approved 2026-10-19:** merge with older cache, deduplicate and cap at 100;
  retry partial results after 15 minutes; retain cache on complete failure.
  Dependencies: none.
- [x] **T016 [P1] Preserve random content-kind variety.**
  Choose a kind before preferring unseen items within that kind, so consumed
  tips remain eligible while unseen news exists. Keep mode/topic filters,
  deduplication, and explicit multi-item counts correct.
  Test exhausted tip history with fresh news, every kind reachable, all-seen
  pools, empty kinds, filters, and deterministic random branches.
  **Approved 2026-10-19:** equal probability for available kinds first, then
  unseen preference within the selected kind; repeats allowed for exhausted kinds.
  Dependencies: none.
- [x] **T017 [P1] Protect shared state across processes.**
  Prevent lost history/greeting/cache updates when several terminals launch.
  Use bounded locking and preserve atomic writes; define failure behavior.
  Test separate concurrent processes, lock timeout, interrupted writers,
  retained source data, and no partial JSON.
  **Approved 2026-10-19:** macOS/Linux; wait up to 250 ms, then show a read-only
  snapshot with a warning, no network and no state writes. Busy edits/setup
  cannot save and fail explicitly. Use a persistent kernel-locked file.
  Dependencies: none.
- [x] **T018 [P1] Expose stale news and cache freshness.**
  Distinguish fresh cache from stale cache; show useful age and a restrained
  refresh hint without networking in offline mode. Do not present old
  announcements as current news.
  Test TTL boundaries, missing dates, future timestamps, old offline cache,
  failed-refresh fallback, and hint frequency.
  **Approved 2026-10-19:** retain old news; show retrieval age and mark stale
  at six hours. Show exactly one refresh hint on stderr per briefing selecting
  stale news. Unknown/future retrieval timestamps are explicitly unknown/stale.
  Dependencies: none.

Reliability milestone verified on 2026-10-19: 88 passing test cases including
subtests, zero failures/skips, 94.0% core statement coverage, race detector,
vet, and installation tests passed. Timing-sensitive tests passed five repeated
runs. Separate CLI subprocesses retained all state updates; an abruptly
terminated lock holder did not leave a stale lock or corrupt the saved cache.
Linux amd64 executable/tests cross-compiled successfully; native Linux runtime
validation remains for CI. The standalone local executable was rebuilt and
reinstalled without modifying the personal profile/cache during verification.

### Terminal experience

- [x] **T019 [P2] Make layout terminal-width-aware.**
  Wrap prose and size dividers for narrow terminals. Handle Unicode/emoji
  display widths and long names; preserve copyable commands and links.
  Test small/normal widths, CJK/combining characters, emoji, long titles,
  commands, URLs, and unknown terminal width.
  **Approved 2026-10-19:** use small terminal-size/display-width dependencies
  (`x/term`, `uniseg`); wrap prose by grapheme width, keep commands/URLs
  unbroken, and fall back to 80 columns when detection is unavailable.
  Dependencies: none.
- [x] **T020 [P2] Add compact shell-startup presentation.**
  Show a greeting, one content item, and attribution/link without the long
  interests footer; retain an explicit detailed view.
  Test compact/detailed output shapes and the one-item default.
  **Approved 2026-10-19:** compact by default, keeping greeting/comment/local
  time/item/source/link; `--detailed` restores heading, borders and interests
  footer. Counts stay unchanged and default to one content item.
  Dependencies: none.
- [x] **T021 [P2] Add plain and redirected-output behavior.**
  Provide a plain mode without emoji/decorative borders, with defined behavior
  for pipes, redirected output, noninteractive shells, and unsupported terminals.
  Test TTY/non-TTY detection, explicit overrides, and escape-free output.
  **Approved 2026-10-19:** non-TTY output automatically becomes plain;
  `--plain` removes decorative emoji/borders/color but keeps text greetings
  and time. Dumb/missing TERM also uses plain output; preserve Unicode content.
  Dependencies: none.
- [x] **T022 [P2] Add tasteful optional color.**
  Respect `NO_COLOR`, terminal capabilities and plain mode; keep emphasis
  accessible rather than using color as the only information channel.
  Test color precedence, disabled colors, unsupported terminals, and pipes.
  **Approved 2026-10-19:** auto cyan headings, warm yellow greeting and dim
  metadata on capable terminals; `--color=auto|always|never`. Plain output
  and nonempty `NO_COLOR` override even `always`; empty `NO_COLOR` does not.
  Dependencies: T021.

Display milestone verified on 2026-10-19: 105 passing cases including subtests,
zero failures/skips, 94.1% core statement coverage, race/vet/install tests passed.
A real 40-column PTY verified terminal sizing, styled detailed output,
`--plain` and `NO_COLOR` precedence; redirected output stayed plain even with
`--color=always`. Unicode tests cover narrow widths, long names, CJK, combining
marks and ZWJ emoji. Commands/URLs stay unbroken. Linux amd64 cross-compilation
passed (native Linux runtime validation remains for CI). Reinstalled the
standalone command; personal profile/cache were not used for verification.

- [x] **T023 [P3] Provide opt-in shell integration.**
  Document or generate an interactive-terminal hook that stays network-free
  and does not clear output or unexpectedly start onboarding.
  Test repeated setup, missing executable/profile, noninteractive shells,
  startup errors, and measured startup cost.
  **Approved 2026-10-19:** explicit install/uninstall for zsh/bash with a
  user-selected file; do not modify the user's existing configuration now.
  Every new terminal greets offline, skips missing profile/executable and
  preserves errors. Measured sample missing-profile real-shell startup ~19–22 ms;
  no portable fixed latency guarantee. See `docs/shell-integration.md`.
  Dependencies: T020, T021.
- [x] **T024 [P3] Offer optional scheduled refresh and briefing frequency.**
  Keep terminal startup independent from network refresh; make scheduling
  explicit, removable, and observable rather than silently launching jobs.
  Test schedule configuration, repeated enable/disable, refresh failures,
  offline operation, and concurrent terminal access.
  **Approved 2026-10-19:** opt-in macOS launchd only, every three hours;
  every new terminal greets (no suppression). Explicit enable/disable/status,
  private local logs, existing profile and pinned executable/state paths.
  Cache-only background refresh preserves display history and exposes errors.
  No actual job enabled; plist/lifecycle tested with simulated launchctl and
  local HTTP/lock tests. Native launchd activation requires user opt-in.
  Dependencies: T017, T018.

### Publication and release foundations

- [x] **T025 [P1] Confirm publication rights and repository destination.**
  Confirm code ownership, permission to publish,
  intended audience, and public/private destination. Update clone/support
  links only after approval and verify access for the intended audience.
  **Current project destination (2026-10-22):**
  `https://github.com/D-Kumar19/opsmosaic`. Repository visibility and remote
  release availability must be checked rather than inferred from this plan.
  Dependencies: none.
- [x] **T026 [P1] Choose and add a license.**
  Add the approved license with correct holder/year; clarify third-party
  feed/content terms separately from the source-code license.
  Verify license identification and README links.
  **Approved 2026-10-19:** MIT, Copyright (c) 2026 Dheeraj Kumar;
  source licensing does not grant rights to third-party article bodies or
  replace release approval.
  Dependencies: T025.
- [x] **T027 [P1] Add continuous integration.**
  Automate formatting checks, unit tests, race tests, vet, build, and isolated
  installation tests. Set a supported Go/OS matrix and useful failure output.
  Verify the chosen runner executes the commands and a deliberate failure
  fails the job; do not claim CI is live before its remote run is checked.
  **Current scope:** configure GitHub Actions for the personal repository.
  **Implemented:** pinned, read-only GitHub Actions workflow with Linux
  Go 1.24.x/stable and macOS stable; 15-minute job timeouts, concurrency
  cancellation, Linux packaging/signing-tool/demo tests and macOS PTY tests.
  The minimum Go version runs on Linux only to conserve included minutes.
  Actions is enabled according to the maintainer.
  **Verified:** hosted run 39753472332 passed all three jobs on commit
  65b9a64. Its Linux release tests deliberately introduce unformatted code in
  an isolated fixture and require CI to return nonzero. Run 39752877495 also
  demonstrated that a real test failure fails the hosted job. No additional
  failing pull request is needed. See `docs/ci.md`.
  Dependencies: T025.
- [x] **T028 [P1] Add version/build metadata.**
  Implement `opsmosaic --version` with a development fallback and release
  version/commit/build information. Keep help/version free of onboarding,
  state writes, and network requests.
  Test local builds, release metadata, invalid arguments, and missing profile.
  **Approved 2026-10-19:** target `v0.1.0`; output includes version, full commit
  and UTC build time, with explicit `dev`/`unknown` source-build defaults.
  Dependencies: none.
- [x] **T029 [P1] Document support and release policy.**
  Define supported macOS/Linux architectures and minimum Go version for
  contributors; add a changelog and tag/version conventions.
  Verify the support claims match actual build/test coverage.
  **Approved 2026-10-19:** macOS/Linux arm64/amd64 targets, Go 1.24 minimum,
  SemVer with alpha/beta/rc prereleases and Keep a Changelog-style notes.
  Local runtime verified only on macOS arm64 with Go 1.27.2; other targets
  cross-built, Go 1.24/native matrix not yet verified.
  Dependencies: T025.
- [x] **T030 [P1] Automate downloadable releases and checksums.**
  Build versioned macOS arm64/amd64 and approved Linux binaries, package them
  with relevant documentation/license, and generate checksums.
  Test metadata, archive layout, checksum verification, and standalone
  execution on native runners where available; distinguish cross-compilation
  from tested runtime support. Require approval before publishing/tagging.
  **Approved 2026-10-19:** portable local scripts, four targets, tar.gz archives,
  SHA-256 checksums, embedded metadata and dependency licenses. Manual
  publication approval required; artifact-host integration deferred.
  **Locally implemented and verified:** `make test-release`, including all
  archive contents/metadata, native smoke, tampering and failure guards.
  **Approved 2026-10-22:** manual Actions dispatch creates a draft GitHub
  Release only; the maintainer publishes after review. Workflow implementation
  checks successful CI for the exact commit, rejects reused versions, packages
  and downloads four archives/checksums, and verifies the native Linux binary.
  Commit/push and one remote draft verification are approved.
  **Verified:** run 39756397057 created draft `v0.1.0` for commit 98fd72e,
  downloaded all five assets, verified checksums and native Linux amd64 version.
  CI run 39756247194 passed all three jobs. No public release is authorized.
  Dependencies: T026, T027, T028, T029.

### Community and portfolio presentation

- [x] **T033 [P2] Add contributor guidance.**
  Explain local setup, tests, source/tip additions, review expectations and
  where to report bugs; add concise issue/PR templates if useful.
  Verify instructions from a clean checkout and use the chosen platform's
  supported template format.
  **Current scope:** contributors with repository access, GitHub issues/PRs,
  concise setup/test/source/tip/review instructions. Templates
  deferred. Build/install/release instructions verified using isolated fixtures.
  Dependencies: T025.
- [x] **T034 [P2] Add a short terminal demo.**
  Show onboarding, time-aware greeting, one random item, and offline behavior.
  Use an example profile with no personal data; keep media accessible and
  include a text equivalent.
  **Approved 2026-10-19:** animated terminal GIF and text transcript, stored in
  `docs/demo/`, using disposable Alex state, not the user's shell/profile.
  Three actual-output scenes show onboarding, one tip and offline return;
  captions explain UTC, disabled color and the intentionally empty news cache.
  Five capture-tool tests pass; the renderer verifies three decodable frames
  with no text overflow. Regeneration uses Python stdlib and macOS Swift/AppKit,
  adding no production dependency. See `docs/demo/README.md`.
  Dependencies: T019, T020, T021.
- [x] **T035 [P2] Prepare the initial release and portfolio summary.**
  Write concise release notes, architecture/design trade-offs, and an accurate
  CV/project description. Include only measured test/performance/adoption data.
  Verify install links and declared features; tag/publish only on approval.
  **Scope:** README/project summary and draft release notes;
  no unsupported CV claims or automatic publishing.
  **Prepared locally:** `docs/project-summary.md`, `docs/release-notes.md`,
  README demo links and honest measurements/platform limitations.
  **Completed 2026-10-22:** final release notes, dated changelog, source/binary
  install guidance and measured CI/platform claims are prepared. T027 and T030
  are complete. Commit these final materials, rebuild the unpublished draft
  with them, and review before maintainer publication. No public release claimed.
  Dependencies: T015, T016, T017, T018, T030, T033, T034.

### Optional product enhancements

- [x] **T036 [P3] Support user-configurable feeds.**
  Design a small validated local configuration for custom RSS/Atom sources,
  topic mapping, and source attribution; preserve built-in defaults and
  migrate configuration without losing preferences.
  Test invalid/duplicate feeds, HTTP failures, schema compatibility, filtering,
  deduplication, terminal sanitization, and portable/offline behavior.
  **Approved 2026-10-19:** CLI `feeds add/list/remove`, separate versioned
  JSON, supplement built-ins, one existing topic and news/release per feed,
  all valid entries without keyword filtering. Public HTTPS/443 only, no
  authentication, private/local targets, query strings or fragments.
  Maximum 20 custom feeds; guard DNS/IP at connect and redirect time.
  Reuse cache/offline/scheduled refresh; doctor validates with private aliases.
  No plugins or new topics. See `docs/custom-feeds.md`.
  Dependencies: none.
- [x] **T037 [P3] Add a diagnostic command.**
  `opsmosaic doctor` should explain executable/version, state paths, profile
  health, cache ages, and source status without leaking private values.
  Test missing/corrupt state and useful exit codes.
  **Approved 2026-10-19:** local-only, human and `--json`, home path prefix
  replaced by `~`; no names/interests/headlines/URLs. No network, state lock,
  onboarding, repair or writes. Missing/stale state warns; corrupt/unreadable
  state or invalid arguments/configuration returns nonzero. Cache status is
  explicitly not a live health probe. See `docs/doctor.md`.
  Dependencies: T018, T028.
- [x] **T038 [P3] Offer stable-release-only filtering.**
  Keep upstream labels visible and make prerelease filtering explicit.
  Test RC/alpha/beta/nightly tags, unknown version formats, and empty results.
  **Approved and implemented 2026-10-19:** keep defaults; per-run
  `--stable-only` excludes prereleases/unknown RELEASE formats only.
  Explicit release-tag links take precedence; otherwise require a complete
  SemVer label with optional supported component prefix or `Release ` prefix.
  Tips/news unaffected; filter never discards cache. Empty news-only results
  fail; mixed mode retains tip fallback. Arbitrarily nested/ambiguous labels
  excluded conservatively. Five regression tests and standalone flag checks pass.
  Dependencies: none.
- [x] **T039 [P3] Add bookmarks and replay.**
  Save an interesting item and show the last displayed item again without
  fetching; keep state independent from expiring feed cache.
  Test persistence, empty history, removed source items, duplicate bookmarks,
  and concurrent writers.
  **Approved and implemented 2026-10-19:** independent `saved.json` remembers
  the complete last normal briefing; `replay` and `bookmarks add/list/show/remove`.
  Multi-item adds require a 1-based number. Keep up to 100 bookmarks until
  removal, reject full additions, reuse duplicates without overwriting capture
  metadata. Replay/show use original capture/source/link, label saved/not live,
  and never fetch, greet, write or advance history/rotation.
  Mutations use the shared state lock; busy normal snapshots do not replace
  replay. Eleven regression tests plus standalone command checks pass.
  See `docs/bookmarks.md` for size/locking/partial-save behavior.
  Dependencies: T017.
- [x] **T040 [P3] Offer an optional full-screen topic picker.**
  Retain the line-based picker as a non-TTY/SSH fallback; keep keyboard
  navigation, accessibility and cancellation predictable.
  Test selection, cancellation without profile loss, resize, and fallback.
  **Approved 2026-10-22:** arrow keys move, Space toggles, Enter confirms; no
  new dependencies; retain numbered prompt for non-TTY/unsupported terminals.
  `q`/Ctrl-C cancel before saving. Picker redraw reads current terminal height
  and scrolls on resize; terminal mode is restored on every exit.
  **Implemented and tested:** state/key/render tests plus real PTY selection,
  live resize, cancellation with existing-profile preservation, and canonical
  mode/echo restoration. See `make test-shell`.
  Dependencies: T019, T021.
- [x] **T041 [P3] Evaluate optional AI summaries.**
  Only proceed with explicit interest: preserve headline/link mode and label
  generated summaries. Define publisher-content handling, privacy, provider,
  cost controls, credentials, and failure behavior before implementation.
  Test opt-in/out, timeouts, rate/cost limits, attribution, and untrusted output.
  **Ask first:** is AI needed at all, local versus hosted model, data-sharing
  permission, budget and credential storage? Never make it mandatory.
  **Decision 2026-10-22:** evaluation only; do not implement or call an AI
  provider. Keep headline/source/link behavior, send no content externally,
  budget $0, and require a later explicit opt-in/provider decision before any
  implementation. If revisited, only public headline/topic/source/link
  metadata is in scope unless separately approved; no article text. A hosted
  credential may only be supplied through an environment variable if later
  authorized. See `docs/ai-summaries.md`.
  **Evaluation complete:** current headline/link workflow is sufficient.
  Metadata-only generation has limited value, while article-text summarization
  introduces fetching/extraction, copyright, privacy, credential, cost and
  failure-handling obligations disproportionate to this small CLI.
  Dependencies: T025.

### Suggested milestones (not authorization to start)

1. **Reliability:** T015-T018.
2. **Display polish:** T019-T022.
3. **Release foundations:** T025-T030 and T033.
4. **Presentation/release preparation:** T034-T035 complete; final materials
   must be committed and included in refreshed draft assets before publication.
5. **Optional work:** T023-T024 and T036-T041 complete
   (T041 is evaluation-only; no AI code).

Do not add a backend, Kubernetes deployment, telemetry, or an AI subscription
solely to increase apparent project complexity. A polished `v0.1.0` is the
proposed target, subject to the release decisions above.

## Follow-up: single random item (2026-10-14)

- [x] T009 (REQ-006): Default to one random tip/news/release; honor explicit
  mode, topic and count options; prefer unseen content; update tests/docs.
- T009 depends on T006 and is verified by deterministic random-selection and
  CLI tests, race detection, vet, build and isolated binary smoke checks.
- Verification: 60 passing cases including subtests, zero failures/skips,
  core coverage 93.6%, rebuilt executable, and no personal state changes.

## Verification (2026-10-13)

- 49 passing test cases including subtests; zero failures/skips.
- Race detector and `go vet ./...` passed.
- Core package statement coverage: 92.7%; whole-project coverage: 91.5%.
- Built `bin/opsmosaic`; first-run/repeat/config/option paths covered.
- Live smoke check returned an item from all nine topic sources.
- Offline smoke check displayed cached items, and invalid count exited nonzero.
- Smoke state was temporary and removed; personal onboarding remains untouched.

## Follow-up: portable installation (2026-10-14)

- [x] T010: Replace the README's personal example with `<YOUR NAME>` and
  document macOS prerequisites, standalone installation, PATH, updates,
  custom prefixes, and uninstalling.
- [x] T011: Add `make install` copying the executable to `~/.local/bin` by
  default, with configurable `PREFIX` and `DESTDIR`.
- [x] T012: Verify installation from outside the checkout, prefixes with
  spaces, staged installs, onboarding, and explicit options; rerun race
  tests/vet; install the standalone command locally.
- T011 depends on T006; T010 documents T011; T012 depends on T010/T011.
- Historical tap proposal was cancelled on 2026-10-19; use `make install`.

## Follow-up: personalized greetings (2026-10-14)

- [x] T013 (REQ-001): Add emoji morning/afternoon/evening/night-owl greetings
  and three rotating comments for each local-time period. Persist successful
  greeting visits with backward-compatible cache state.
- [x] T014 (REQ-008): Test hour boundaries, local timezone, sanitization,
  persisted rotation, legacy caches, installation and existing CLI behavior.
- T013 depends on T006; T014 depends on T013. Document the greeting periods
  and reinstall the verified standalone executable.
