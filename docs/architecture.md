# Architecture

`cmd/opsmosaic` provides process exit handling. `internal/brief` owns CLI
orchestration, models, onboarding, storage, feed adapters, selection, and
rendering. All components use the Go standard library.

## Data flow

Parse command -> load or create profile -> resolve topics -> load cache ->
fetch stale selected sources concurrently -> filter and deduplicate items ->
prefer unseen content -> randomly choose one available kind and one item ->
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

## Local model

- Profile: schema version, name, selected topic slugs.
- Item: stable ID, kind, topic, title, source, URL, optional command and
  publication timestamp. Cached status is presentation metadata.
- Source cache: last successful fetch timestamp and normalized items.
- Cache: schema version, source-ID map, last 100 displayed IDs.

Config: OS user-config directory / opsmosaic / config.json.
Cache: OS user-cache directory / opsmosaic / cache.json.
`OPSMOSAIC_HOME` overrides both roots for portable/isolated operation.
Writes use a temporary file in the same directory followed by rename.

## Sources

Official Docker, Kubernetes and AWS RSS; CISA advisories RSS for Security;
GitHub release Atom for Helm, Argo CD, Kargo and Crossplane; Hacker News JSON
for AI. General feeds are title-filtered and are not comprehensive indexes.
HTTP has a five-second client timeout, a 2 MiB response cap, and bounded
parallel source fetches (maximum nine). HN scans at most 30 top-story items,
with bounded parallel requests and a whole-source timeout.
