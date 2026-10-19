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

## Deliberate first-release scope

Use line-based, numbered multi-selection rather than a full-screen TUI:
accessible over SSH and scriptable. No shell-startup hook
is installed automatically. No summaries or copied article bodies.
Names/preferences stay on the device; feed providers still see ordinary HTTP
requests. Public feeds can change or disappear; their availability is not
guaranteed. Source terms apply; links and attribution do not establish a
license for republishing full articles.

## Later possibilities

Arrow-key multi-selection, opt-in
shell greeting, user-supplied feeds, bookmarks, release packaging, and optional
AI summaries. These are backlog items, not first-release requirements.
