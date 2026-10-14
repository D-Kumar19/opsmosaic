# OpsMosaic: project context

OpsMosaic is a personal terminal briefing for people working in DevOps, cloud,
and security. It combines a mosaic of interests into a small, useful daily
read: a practical tip, relevant news, and project releases with original links.

## First-release requirements

- REQ-001: Ask for a name and multiple interests on first run. Persist locally;
  subsequent runs welcome the user back with local date, time, and timezone.
  `config` repeats onboarding without destroying an existing profile on error.
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
- REQ-006: Default to exactly one randomly chosen tip, news item, or release
  matching selected interests. Choose randomly among available kinds, then
  among items within that kind, preferring unseen content. Support `--tips`,
  `--news`, `--topic`, and `--count` (1-20 total items). Track the last 100
  shown items, repeating only when necessary. Only an explicit `--count`
  greater than one enables multiple items; preserve balanced selection for
  these larger briefings.
- REQ-007: Treat external text as untrusted: remove terminal escape/control
  sequences, decode markup entities, and reject non-HTTP(S) links.
  Never execute fetched commands. Store local state atomically with mode 0600.
- REQ-008: Provide hermetic unit/integration tests and reproducible build,
  vet, race-test, and smoke-check commands. No mandatory hosted services,
  credentials, telemetry, AI model, or third-party Go dependencies.

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

## Deliberate first-release scope

Use line-based, numbered multi-selection rather than a full-screen TUI:
accessible over SSH, scriptable, and dependency-free. No shell-startup hook
is installed automatically. No summaries or copied article bodies.
Names/preferences stay on the device; feed providers still see ordinary HTTP
requests. Public feeds can change or disappear; their availability is not
guaranteed. Source terms apply; links and attribution do not establish a
license for republishing full articles.

## Later possibilities

Arrow-key multi-selection, color and terminal-width-aware layouts, opt-in
shell greeting, user-supplied feeds, bookmarks, release packaging, and optional
AI summaries. These are backlog items, not first-release requirements.
