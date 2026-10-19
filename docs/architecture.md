# Architecture

`cmd/opsmosaic` provides process exit handling. `internal/brief` owns CLI
orchestration, models, onboarding, storage, feed adapters, selection, and
rendering. Core orchestration uses the Go standard library. The display layer
uses `golang.org/x/term` for terminal capability/size detection and
`github.com/rivo/uniseg` for Unicode widths and grapheme iteration. `x/sys` is
a transitive dependency of `x/term`. All compile into the executable.

## Display

Resolve output options from CLI flags, output descriptor, `TERM`, and
`NO_COLOR`. Non-TTY/dumb/unknown terminals imply plain output. Plain output
and nonempty `NO_COLOR` take priority over forced color. Compact mode omits
heading/dividers/footer; detailed mode includes these, but plain detailed
output still has no decorative borders. Greetings remain in all modes.

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
cache and recent history.

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
gated on verified project runners and approval; see [CI](ci.md) and [releases](releases.md).
