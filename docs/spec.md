# OpsMosaic: project context

OpsMosaic is a personal terminal briefing for people working in DevOps, cloud,
and security. It combines a mosaic of interests into a small, useful daily
read: a practical tip, relevant news, and project releases with original links.

## First-release requirements

- REQ-001: Ask for a name and multiple interests on first run. Persist locally;
  subsequent runs welcome the user back with local date, time, and timezone.
  `config` repeats onboarding without destroying an existing profile on error.
  Use emoji greetings based on local hour (morning 05-11, afternoon 12-16,
  evening 17-21, night owl 22-04) with three rotating comments per period.
  Persist rotation after successful displays; accept older caches.
- REQ-002: Support Docker, Kubernetes, AWS, Security, Helm, Argo CD, Kargo,
  Crossplane, and AI. Validate selections and `--topic` overrides.
- REQ-003: Ship reviewed, read-only tips with documentation links for every
  topic. Tips are distinct from news and releases and work without networking.
- REQ-004: Fetch public RSS/Atom feeds and Hacker News JSON. Show headline,
  source, publication time when available, and HTTP(S) article link.
  Filter general feeds, deduplicate articles, and balance topics.
- REQ-005: Cache each source for six hours. `refresh` explicitly fetches.
  `--offline` never performs network requests. Preserve last-known items on
  source failure, label cached content, and report errors on stderr.
  Partial HN results merge with older entries, deduplicated and capped at 100,
  and retry after 15 minutes; complete failures preserve the cache.
  Show retrieval age and mark cached items stale at six hours, including
  unknown/future timestamps. Preserve age for older merged entries. Keep stale
  items visible and emit one refresh hint on stderr per briefing selecting them.
- REQ-006: Default to exactly one randomly chosen tip, news item, or release
  matching selected interests. Choose randomly among available kinds, then
  among items within that kind, preferring unseen content within that kind.
  Kinds have equal probability even when all their items have been seen.
  Support `--tips`,
  `--news`, `--topic`, and `--count` (1-20 total items). Track the last 100
  shown items, repeating only when necessary. Only an explicit `--count`
  greater than one enables multiple items; preserve balanced selection for
  these larger briefings.
- REQ-007: Treat external text as untrusted: remove terminal escape/control
  sequences, decode markup entities, and reject non-HTTP(S) links.
  Never execute fetched commands. Store local state atomically with mode 0600.
  On macOS/Linux, protect transactions with a cross-process lock; wait at most
  250 ms, then render a read-only/no-network snapshot with a warning if busy.
  Busy preference edits or first-time setup fail explicitly without prompting.
- REQ-008: Provide hermetic unit/integration tests and reproducible build,
  vet, race-test, and smoke-check commands. No mandatory hosted services,
  credentials, telemetry, AI model, or external runtime. Small approved Go
  dependencies support terminal-size detection and Unicode display widths.
- REQ-009: Default to compact output retaining greeting/comment/local time,
  item and source/link; `--detailed` restores dividers and interests/count.
  Wrap prose to terminal width with grapheme-aware widths (80-column fallback),
  but never insert wraps into commands or URLs. `--plain` suppresses decorative
  emoji, borders and color; non-TTY/dumb/unknown terminals use plain output
  automatically while keeping textual greetings. `--color=auto|always|never`
  uses subtle cyan headings, warm greetings and dim metadata; plain output and
  nonempty `NO_COLOR` override color, even `always`. Empty `NO_COLOR` does not.
- REQ-010: Use the personal project repository at
  `https://github.com/D-Kumar19/opsmosaic`. Retain the MIT source license,
  contributor guidance, SemVer/changelog and truthful support documentation.
  `--version` reports dev/release version, commit and UTC build time without
  onboarding, network or state changes. Provide portable local CI checks and
  isolated four-target release/archive/checksum tests. Package dependency
  licenses and require clean committed source; never overwrite a version.
  Remote CI requires configured runners and verified success/failure evidence.
  Tagging, pushing and uploading require separate explicit approval.
- REQ-011: Prepare an animated terminal demo, accessible text
  transcript, project summary and draft release notes. Capture real CLI output
  with a disposable example profile, one random item and offline return;
  do not capture the user's shell or state. Clearly label draft release status
  and distinguish measured evidence from unverified claims. A Homebrew tap is
  not planned; retain standalone source installation.

- REQ-012: Explicit shell install/uninstall commands support zsh/bash and an
  explicitly chosen regular configuration file. Preserve unrelated content and
  permissions, back up the original, reject ambiguous markers/symlinks and
  avoid duplicate managed blocks. Hooks run only in interactive terminal
  sessions; missing executable/profile skips, failures stay visible. Greeting
  every terminal is offline and never starts onboarding.
- REQ-013: Opt-in macOS launchd refresh every three hours, enable/disable/status
  with private logs and pinned executable/state paths. Do not enable any real
  job during implementation. Refresh uses existing profile and bounded locking,
  preserves cache on failures, reports errors, and never advances greeting or
  recent-item history. No Linux scheduler or suppression window in this scope.
- REQ-014: Prepare optional local OpenSSH signatures over release checksums,
  verifying independently trusted identity/namespace before archive hashes.
  Refuse overwrite and unsafe manifest paths; test unsigned/wrong-key/tampered
  failures using disposable keys only. These existing local tools are retained;
  production signing/provenance is outside scope and is not a release gate.
  No Apple signing is implied.

- REQ-015: `doctor [--json]` reports local executable/build/platform, redacted
  state paths, profile validity and cached-source ages/status without network,
  onboarding, locking or writes. Omit profile values, headlines/URLs and raw
  parser errors. Replace home path prefixes with `~`; document remaining path
  disclosure. Missing/stale state warns with success; invalid/unreadable state
  returns an error after a structured report. JSON has a versioned tested shape.

- REQ-016: Add/list/remove up to 20 custom RSS/Atom feeds in separate versioned
  local JSON; supplement built-ins, preserve profiles and map each source to
  one existing topic and news/release. Reject duplicate definitions/URLs,
  invalid schema and unsafe URLs explicitly. Public HTTPS/443 only, no
  credentials/query strings/fragments, authenticated/internal feeds or plugins.
  Enforce public DNS/IP pinning and safe redirects; bound concurrency and reuse
  cache/offline/failure/sanitization/selection behavior across normal and
  scheduled runs. Doctor validates privately with positional custom-source IDs.

- REQ-017: Keep existing defaults; `--stable-only` filters only RELEASE items
  to recognized stable SemVer labels including supported component prefixes.
  Explicit release-tag URLs take precedence over titles; reject prereleases
  and unknown/ambiguous formats. Apply after cache loading/fetching and before
  selection, including customs/offline/refresh; preserve all source cache
  entries, news and tips. Retain explicit empty-news failure and mixed tip
  fallback with an informative filtering message. No persisted filter setting.

- REQ-018: Persist the complete last normal briefing independently of cache.
  `replay` and `bookmarks show ID` render original content/source/link and
  capture/publication times as saved, not live, without greeting, network,
  history/rotation changes or writes. `bookmarks add [ITEM_NUMBER]` requires a
  1-based index for multi-item briefings; list/remove use stable IDs. Keep up to
  100 until manual removal, reuse duplicates and reject full additions without
  eviction. Lock mutations, preserve bookmarks on later briefings, validate
  private atomic state, and surface errors without silent resets.
- REQ-019: On interactive supported terminals, offer an accessible full-screen
  topic picker using arrow keys to move, Space to toggle, Enter to confirm,
  and `q`/Ctrl-C to cancel without saving. Use no new dependency. Re-evaluate
  terminal dimensions on redraw and scroll the topic list when needed. For
  piped/non-interactive input or unsupported terminals, retain the numbered
  prompt. Preserve validation and existing-profile contents on cancellation.

## Acceptance scenarios

1. A new user enters a name and comma-separated topic numbers and sees a
   briefing. The next run uses the saved profile without prompting.
2. Invalid names/selections are explained and re-prompted; EOF fails clearly.
3. A failed feed uses its older cache with a warning. No cached news yields
   an explicit empty state; news-only mode fails rather than claiming success.
4. Offline mode uses cache and bundled tips, even without internet.
5. An article appearing in multiple feeds appears only once in a briefing.
6. Titles containing terminal escape sequences cannot control the terminal.
7. A default run renders exactly one card, which can be a tip, news, or release.
   Explicit mode/topic filters restrict that choice; `--count 3` permits three.
8. One failed HN story does not discard successful stories; partial results
   retain older entries with truthful per-item retrieval ages and a short retry.
9. Concurrent processes do not lose saved state updates; busy snapshot readers
   do not fetch or mutate state. An interrupted lock holder leaves no stale lock.
10. Offline five-day-old news shows its age/stale label and one refresh hint.
11. A new user can run `--version` without creating state or making requests.
    Release archives carry embedded metadata, documentation and licenses;
    tampered archives fail checksum verification. Failed packaging leaves no
    completed output. Cross-builds are not claimed as native runtime tests.

## Deliberate first-release scope

Use line-based, numbered multi-selection rather than a full-screen TUI:
accessible over SSH and scriptable. No shell-startup hook
is installed automatically. No summaries or copied article bodies.
Names/preferences stay on the device; feed providers still see ordinary HTTP
requests. Public feeds can change or disappear; their availability is not
guaranteed. Source terms apply; links and attribution do not establish a
license for republishing full articles.

## Later possibilities

Arrow-key multi-selection, bookmarks, remote release publishing, and optional
AI summaries. These are backlog items, not first-release requirements.
