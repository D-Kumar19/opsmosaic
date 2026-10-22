# Architecture

`cmd/opsmosaic` provides process exit handling. `internal/brief` owns CLI
orchestration, models, onboarding, storage, feed adapters, selection, and
rendering. Core orchestration uses the Go standard library. The display layer
uses `golang.org/x/term` for terminal capability/size detection and
`github.com/rivo/uniseg` for Unicode widths and grapheme iteration. `x/sys` is
a transitive dependency of `x/term`. All compile into the executable.

On macOS/Linux interactive onboarding, the topic picker uses `x/term` raw mode
and the already-present `x/sys/unix` poll support; no dependency is added.
Other platforms and non-TTY input use the numbered prompt. The picker restores
terminal input state and alternate-screen visibility on success, cancellation,
and failures.

## Display

Resolve output options from CLI flags, output descriptor, `TERM`, and
`NO_COLOR`. Non-TTY/dumb/unknown terminals imply plain output. Plain output
and nonempty `NO_COLOR` take priority over forced color. Compact mode omits
heading/dividers/footer; detailed mode includes these, but plain detailed
output still has no decorative borders. Normal briefings retain greetings;
saved replay/show replace them with a capture-time, not-live banner.

`display.go` wraps sanitized prose before applying ANSI styling. It measures
grapheme clusters, never splits a cluster, and leaves commands/URLs unwrapped.
Unknown width falls back to 80. Very narrow widths cannot contain a cluster
wider than the entire terminal; preserve it rather than corrupting Unicode.
The sanitizer preserves emoji ZWJ sequences while rejecting terminal controls
and other format characters. Plain mode removes only application decorations,
not Unicode user names or article content. Tests resolve policy deterministically
without requiring a real terminal; release verification also probes a real PTY.

## Data flow

Parse command -> early help/version exit -> acquire bounded state lock -> load or create profile ->
resolve topics -> load cache ->
fetch stale selected sources concurrently -> filter and deduplicate items ->
randomly choose one available kind -> prefer unseen content within that kind ->
randomly choose one item ->
print greeting/items/warnings -> atomically save
cache and recent history, then the complete last briefing snapshot.

The default count is one. Mode/topic flags narrow the random candidate pool.
The standard library's `math/rand/v2` provides randomness; tests inject an
index picker to deterministically cover every kind and candidate. An explicit
count greater than one retains balanced multi-item selection and topic rotation.

Tips-only mode makes no HTTP requests. Offline mode uses cached data without
refreshing it. Source errors do not erase successful older cache entries.
Empty news-only output is an error; the default briefing may still show a tip
and an explicit news empty state.

HN adapters return successful items alongside a typed `PartialFetchError` when
some story requests fail. A total failure returns only an error. The source
cache merges partial items before older entries, deduplicates canonical URLs,
caps the result at 100, and sets `retry_after` to 15 minutes later. Complete
success replaces the cache and clears that deadline; complete failure preserves
it. `refresh` bypasses the deadline, and future-clock cache state triggers a
refetch. No external request errors are hidden.

## Local model

- Profile: schema version, name, selected topic slugs.
- Item: stable ID, kind, topic, title, source, URL, optional command and
  publication timestamp. Optional `fetched_at` preserves per-item retrieval
  time across partial merges. Missing values fall back to the source's last
  complete fetch time. Cached status is presentation metadata.
- Source cache: last complete fetch timestamp, normalized items, optional
  `retry_after` for partial results. Optional fields preserve v1 compatibility.
- Cache: schema version, source-ID map, last 100 displayed IDs, optional
  `greeting_visits` counter (zero for older caches). Greeting period uses the
  local hour; counter modulo three picks the comment. Advance after rendering
  succeeds and atomically save with history.

Config: OS user-config directory / opsmosaic / config.json.
Cache: OS user-cache directory / opsmosaic / cache.json.
`OPSMOSAIC_HOME` overrides both roots for portable/isolated operation.
Writes use a temporary file in the same directory followed by rename.

`config.json.lock` is a stable, never-unlinked file beside the profile.
macOS/Linux `flock` protects the whole read/modify/write transaction (including
onboarding and fetches); closing the descriptor or process exit releases it.
The 250 ms acquisition deadline falls back to read-only cached rendering with a
warning and no requests or writes. Busy `config`/missing-profile runs fail
explicitly. Other platforms report unsupported locking rather than silently
running without protection. Snapshot readers rely on atomic JSON replacement.

Cached item age uses retrieval time, never publication time. Six-hour-old
items and unknown/future timestamps are stale. Stale items remain selectable,
with one stderr refresh hint per briefing only if selected.

## Saved briefings

`saved.go` dispatches replay/bookmark commands before profile/feed loading.
Version-1 `saved.json` beside the profile holds an optional last snapshot and
up to 100 independent single-item bookmarks. Snapshots embed full Item values
and capture time; no source/cache references are needed to replay.
Bookmark identity hashes kind plus canonical URL, or stable tip ID for tips.
Mutations reload under the same process lock and atomically write mode 0600;
read-only replay/list/show do not lock, fetch or change history/greetings.
Normal runs validate existing saved state before display and preserve bookmarks
when updating the last briefing. Failed rendering and busy fallback never save.
Scheduled refresh never updates this file. Size is bounded to the 16 MiB reader
limit. Doctor validates it without exposing values.

Cache/history writes precede snapshot writes: atomic files are not a multi-file
transaction. Partial-save errors are explicit; see [persistence details](bookmarks.md).

## Sources

Official Docker, Kubernetes and AWS RSS; CISA advisories RSS for Security;
GitHub release Atom for Helm, Argo CD, Kargo and Crossplane; Hacker News JSON
for AI. General feeds are title-filtered and are not comprehensive indexes.
HTTP has a five-second client timeout, a 2 MiB response cap, and bounded
parallel source fetches (maximum nine). HN scans at most 30 top-story items,
with bounded parallel requests and a whole-source timeout.

## Build and release boundaries

`version.go` owns linker-settable version, commit and UTC timestamp fields,
with explicit development defaults. Version output exits before profile/cache
access or lock acquisition, sanitizes metadata, and propagates output failures.
It is not combined with briefing options or subcommands.

`make ci` is a portable local entry point, not an active hosted workflow.
`scripts/release.sh` stages four cross-builds, source documentation and dependency
licenses in ignored output, computes SHA-256 checksums, then renames the finished
directory. Clean-source and overwrite guards prevent misleading packages.
`scripts/test-release.sh` runs in a disposable committed repository, verifies
every archive's metadata/licenses and a native standalone briefing, and exercises
failure guards. No scripts tag or publish. Remote CI and distribution remain
gated on verified remote CI and maintainer approval; see [CI](ci.md) and [releases](releases.md).

## Opt-in integrations

`integration.go` dispatches explicit shell/schedule subcommands before briefing
parsing. Shell setup requires an explicit file, backs up original content and
atomically edits a uniquely marked block without touching unmanaged hooks.
The guarded hook calls `shell greeting`, which checks profile existence before
using the standard offline briefing path.

macOS LaunchAgents pin the executable and paired absolute config/cache paths.
`schedule refresh` reuses source/cache logic under the normal lock, but does not
select/render items or mutate greeting/history state. A narrow launchctl runner
boundary lets tests exercise lifecycle/failure handling without enabling jobs.
No daemon or Linux scheduling abstraction is added.

Optional shell scripts sign the exact checksum manifest through OpenSSH and
verify an independently trusted signer before checking local archive hashes.
No private key is stored in project state or built into the executable.

`doctor.go` is an early read-only subcommand. It reads profile/cache snapshots
without the transaction lock, reuses their schema validation, and maps failures
to generic diagnostic codes instead of exposing private parser/validation text.
A versioned report drives text and JSON output; fixed built-in source IDs avoid
printing arbitrary saved IDs. Per-item retrieval ages preserve partial-cache
truthfulness. Diagnostic output never changes greeting/history/cache.

## Custom feeds

`custom_feeds.go` owns version-1 `feeds.json` beside the profile. CLI edits use
the shared transaction lock and atomic 0600 writes without changing profile
schema. Full-definition hashes namespace custom source cache keys. A copied App
combines configured sources with built-ins for ordinary and scheduled fetches,
leaving injected/default source definitions unchanged. Remove prunes only that
custom source's cache; see [cross-file interruption behavior](custom-feeds.md).

`public_feed.go` supplies a direct, no-proxy HTTPS transport for customs only.
It validates URLs/redirects, resolves and checks all addresses, then dials an
approved literal IP while normal TLS verifies the original host. Feed HTTP body
limits/parser sanitization remain shared. The fetch loop limits all concurrent
sources to nine; at most 20 customs and 100 entries per source are allowed.
Doctor appends positional `custom-N` cache summaries without exposing config.

`releases.go` classifies stable labels for the optional per-run
`--stable-only` flag. Explicit release-tag paths are authoritative; otherwise
only conservative whole-title version labels qualify. Filtering occurs on
loaded candidates before selection, not during source fetch or cache writes.
Thus scheduled refresh still retains all releases and later unfiltered runs
can show them. Tips/news are unaffected.
