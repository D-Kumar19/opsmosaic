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

## Future backlog

- [ ] Full-screen topic picker and optional color.
- [ ] Opt-in shell startup hook that does not block on network.
- [ ] Custom feeds, saved articles and configurable briefing frequency.
- [ ] Signed binaries and cross-platform release automation.
- [ ] Publish a Homebrew tap/formula backed by accessible versioned releases.
- [ ] Optional, clearly labeled AI summaries with privacy/cost controls.

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
