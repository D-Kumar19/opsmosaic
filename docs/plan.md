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

## Future backlog

The task list below supersedes the earlier informal backlog. Checkboxes track
progress; recording tasks does not approve a design, publication, or release.

### Working agreement

- Work on tasks only when requested. On 2026-10-19 the user authorized the
  first milestone (Reliability, T015-T018), then the second milestone
  (Display polish, T019-T022); later milestones remain unstarted.
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

- [ ] **T023 [P3] Provide opt-in shell integration.**
  Document or generate an interactive-terminal hook that stays network-free
  and does not clear output or unexpectedly start onboarding.
  Test repeated setup, missing executable/profile, noninteractive shells,
  startup errors, and measured startup cost.
  **Ask first:** documentation-only versus an install/uninstall command;
  supported shells and acceptable startup-time budget?
  Dependencies: T020, T021.
- [ ] **T024 [P3] Offer optional scheduled refresh and briefing frequency.**
  Keep terminal startup independent from network refresh; make scheduling
  explicit, removable, and observable rather than silently launching jobs.
  Test schedule configuration, repeated enable/disable, refresh failures,
  offline operation, and concurrent terminal access.
  **Ask first:** manual-only versus opt-in macOS launchd/Linux scheduling;
  frequency, network policy, logs, and whether to suppress repeated greetings
  across closely spaced terminal launches?
  Dependencies: T017, T018.

### Publication and release foundations

- [ ] **T025 [P1] Confirm publication rights and repository destination.**
  Confirm code ownership, permission to publish publicly,
  intended audience, and public/private destination. Update clone/support
  links only after approval and verify access for the intended audience.
  **Ask first:** does the user have permission to publish, and what host,
  account/organization, repository URL, and visibility should be used?
  Dependencies: none.
- [ ] **T026 [P1] Choose and add a license.**
  Add the approved license with correct holder/year; clarify third-party
  feed/content terms separately from the source-code license.
  Verify license identification and README links.
  **Ask first:** MIT, Apache-2.0, or another license; who is the copyright
  holder, and are there maintainer requirements?
  Dependencies: T025.
- [ ] **T027 [P1] Add continuous integration.**
  Automate formatting checks, unit tests, race tests, vet, build, and isolated
  installation tests. Set a supported Go/OS matrix and useful failure output.
  Verify the chosen runner executes the commands and a deliberate failure
  fails the job; do not claim CI is live before its remote run is checked.
  **Ask first:** which CI provider is supported by the final repository,
  runner availability, Go versions, and OS matrix?
  Dependencies: T025.
- [ ] **T028 [P1] Add version/build metadata.**
  Implement `opsmosaic --version` with a development fallback and release
  version/commit/build information. Keep help/version free of onboarding,
  state writes, and network requests.
  Test local builds, release metadata, invalid arguments, and missing profile.
  **Ask first:** version output format and initial version (`v0.1.0` proposed)?
  Dependencies: none.
- [ ] **T029 [P1] Document support and release policy.**
  Define supported macOS/Linux architectures and minimum Go version for
  contributors; add a changelog and tag/version conventions.
  Verify the support claims match actual build/test coverage.
  **Ask first:** initial platform scope, prerelease policy, and changelog style?
  Dependencies: T025.
- [ ] **T030 [P1] Automate downloadable releases and checksums.**
  Build versioned macOS arm64/amd64 and approved Linux binaries, package them
  with relevant documentation/license, and generate checksums.
  Test metadata, archive layout, checksum verification, and standalone
  execution on native runners where available; distinguish cross-compilation
  from tested runtime support. Require approval before publishing/tagging.
  **Ask first:** release tool versus scripts, artifact host, naming, Linux
  architectures, and manual versus automatic publishing approval?
  Dependencies: T026, T027, T028, T029.
- [ ] **T031 [P2] Publish a real Homebrew tap.**
  Create a formula backed by accessible tagged source or release artifacts,
  verify checksums, and document install/upgrade/uninstall.
  Run formula audit/install tests, verify the installed version, and smoke
  test a fresh user with no Go runtime dependency for binary distribution.
  **Ask first:** tap owner/repository, public access, source versus binary
  formula, and who maintains future releases?
  Dependencies: T030.
- [ ] **T032 [P3] Add signing/provenance where appropriate.**
  Evaluate release signatures, provenance, and macOS signing/notarization;
  implement only approved measures with credentials kept out of the repo.
  Test verification steps, tampered artifacts, and unsigned failure modes.
  **Ask first:** desired trust guarantees, certificate/service availability,
  acceptable cost, and whether this is needed for the initial release?
  Dependencies: T030.

### Community and portfolio presentation

- [ ] **T033 [P2] Add contributor guidance.**
  Explain local setup, tests, source/tip additions, review expectations and
  where to report bugs; add concise issue/PR templates if useful.
  Verify instructions from a clean checkout and use the chosen platform's
  supported template format.
  **Ask first:** contribution scope, support expectations and issue platform?
  Dependencies: T025.
- [ ] **T034 [P2] Add a short terminal demo.**
  Show onboarding, time-aware greeting, one random item, and offline behavior.
  Use an example profile with no personal data; keep media accessible and
  include a text equivalent.
  **Ask first:** GIF/video/static screenshots, visual style, and where assets
  should live? Do not record the user's real shell/profile without approval.
  Dependencies: T019, T020, T021.
- [ ] **T035 [P2] Prepare the initial release and portfolio summary.**
  Write concise release notes, architecture/design trade-offs, and an accurate
  CV/project description. Include only measured test/performance/adoption data.
  Verify install links and declared features; tag/publish only on approval.
  **Ask first:** intended audience, release readiness criteria, and whether
  to publish a CV bullet, blog post, or project-page summary?
  Dependencies: T015, T016, T017, T018, T030, T033, T034.

### Optional product enhancements

- [ ] **T036 [P3] Support user-configurable feeds.**
  Design a small validated local configuration for custom RSS/Atom sources,
  topic mapping, and source attribution; preserve built-in defaults and
  migrate configuration without losing preferences.
  Test invalid/duplicate feeds, HTTP failures, schema compatibility, filtering,
  deduplication, terminal sanitization, and portable/offline behavior.
  **Ask first:** add versus replace built-ins, supported source types,
  configuration format, URL policy, and whether authenticated feeds are
  explicitly out of scope? No plugin system unless separately requested.
  Dependencies: none.
- [ ] **T037 [P3] Add a diagnostic command.**
  `opsmosaic doctor` should explain executable/version, state paths, profile
  health, cache ages, and source status without leaking private values.
  Test missing/corrupt state and useful exit codes.
  **Ask first:** local-only by default or active source probes? Which
  diagnostic details may be displayed, and is JSON output wanted?
  Dependencies: T018, T028.
- [ ] **T038 [P3] Offer stable-release-only filtering.**
  Keep upstream labels visible and make prerelease filtering explicit.
  Test RC/alpha/beta/nightly tags, unknown version formats, and empty results.
  **Ask first:** default include/exclude prereleases, opt-in flag/config,
  and treatment of non-semver tags and nested component releases?
  Dependencies: none.
- [ ] **T039 [P3] Add bookmarks and replay.**
  Save an interesting item and show the last displayed item again without
  fetching; keep state independent from expiring feed cache.
  Test persistence, empty history, removed source items, duplicate bookmarks,
  and concurrent writers.
  **Ask first:** command names, retention, storage, and whether replay advances
  content history or greeting rotation?
  Dependencies: T017.
- [ ] **T040 [P3] Offer an optional full-screen topic picker.**
  Retain the line-based picker as a non-TTY/SSH fallback; keep keyboard
  navigation, accessibility and cancellation predictable.
  Test selection, cancellation without profile loss, resize, and fallback.
  **Ask first:** desired interaction, dependency budget, and whether the
  existing numbered picker is already sufficient?
  Dependencies: T019, T021.
- [ ] **T041 [P3] Evaluate optional AI summaries.**
  Only proceed with explicit interest: preserve headline/link mode and label
  generated summaries. Define publisher-content handling, privacy, provider,
  cost controls, credentials, and failure behavior before implementation.
  Test opt-in/out, timeouts, rate/cost limits, attribution, and untrusted output.
  **Ask first:** is AI needed at all, local versus hosted model, data-sharing
  permission, budget and credential storage? Never make it mandatory.
  Dependencies: T025.

### Suggested milestones (not authorization to start)

1. **Reliability:** T015-T018.
2. **Display polish:** T019-T022.
3. **Release foundations:** T025-T030 and T033.
4. **Public launch:** T031, T034-T035 when publication is approved.
5. **Optional work:** T023-T024, T032 and T036-T041, based on user priorities.

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
- Homebrew tap publication remains a release task, not an existing service.

## Follow-up: personalized greetings (2026-10-14)

- [x] T013 (REQ-001): Add emoji morning/afternoon/evening/night-owl greetings
  and three rotating comments for each local-time period. Persist successful
  greeting visits with backward-compatible cache state.
- [x] T014 (REQ-008): Test hour boundaries, local timezone, sanitization,
  persisted rotation, legacy caches, installation and existing CLI behavior.
- T013 depends on T006; T014 depends on T013. Document the greeting periods
  and reinstall the verified standalone executable.
