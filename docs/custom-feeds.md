# Custom RSS/Atom feeds

Custom feeds supplement the built-in sources; they never replace or disable
them. Each feed maps to one existing topic and is either `news` or `release`.
All valid entries are included without keyword filtering. Only feeds belonging
to the briefing's selected topics are fetched. Topic selection still happens
through onboarding/config or a per-run `--topic` override.

## Commands

```sh
# Illustrative URL only: replace it with an actual public RSS/Atom endpoint.
opsmosaic feeds add --id team-updates --name "Example project" \
  --topic kubernetes --kind news --url https://example.org/feed.xml
opsmosaic feeds list
opsmosaic feeds remove team-updates
opsmosaic feeds --help
```

Add/remove/list do not fetch or start onboarding. Add validates syntax, not
DNS availability, feed existence or content; run `opsmosaic refresh` explicitly
when ready to fetch. `feeds list` intentionally displays configured names,
topics and URLs, so review it before sharing. Never enter private or
token-bearing URLs, including secret path segments.

IDs are unique lowercase slugs starting with a letter (1–40 characters).
Names are 1–100 bytes of safe attribution text. Topics are
`docker,kubernetes,aws,security,helm,argocd,kargo,crossplane,ai`.
Kinds must be `news` or `release`. At most 20 custom feeds are allowed.
Duplicate IDs, duplicate canonical URLs and URLs duplicating built-ins are
rejected. To change a feed, remove and re-add it.

## Public HTTPS policy

- HTTPS only on port 443, with normal certificate verification.
- No URL credentials, query strings (even empty `?`) or fragments.
- Public fully qualified ASCII/punycode DNS names or public IP addresses only.
  Local/special-use names and private/reserved addresses are rejected.
- Every connection resolves and checks **all** returned addresses, rejecting
  mixed public/private results. The connection is pinned to an approved IP;
  it does not resolve that hostname a second time during dialing.
- Redirects are revalidated, limited to five requests in the redirect chain,
  and cannot downgrade HTTPS or reach a private/local target.
- Custom feed connections are direct: environment proxies are not used,
  because they would bypass destination-IP enforcement. Networks requiring an
  HTTP proxy may therefore be unable to fetch custom feeds. Built-ins keep
  their existing HTTP-client behavior.
- No cookies, authentication headers, client certificates or token storage.
  This is not support for authenticated/internal feeds. URL validation cannot
  infer whether an arbitrary public URL path contains a secret; do not supply one.

These checks apply to feed fetches. Article links are displayed only and retain
the existing HTTP(S), no-credentials and terminal-sanitization rules; OpsMosaic
never automatically follows an article link or executes its commands.

## State and caching

`feeds.json` is a separate version-1 file beside the profile, including when
`OPSMOSAIC_HOME` or explicit scheduler paths are used. Existing profiles and
caches need no migration. Missing configuration means zero custom feeds.
Unsupported/corrupt configuration fails explicitly rather than silently
dropping feeds; `--tips` can still use bundled tips.

Example shape:

```json
{
  "version": 1,
  "feeds": [
    {
      "id": "team-updates",
      "name": "Example project",
      "topic": "kubernetes",
      "kind": "news",
      "url": "https://example.org/feed.xml"
    }
  ]
}
```

Prefer the CLI to editing the file manually. Writes are atomic, mode 0600,
under the existing bounded cross-process state lock. Source cache identities
hash the whole definition, preventing changed URLs/topics/kinds from reusing
incorrect old entries. Manual edits can leave unused cache entries until a
manual cache rebuild; they are not selected merely because they remain stored.
Feed removal prunes its own cache entry without resetting profile, history,
greeting rotation or other sources. Cache pruning and configuration replacement
are two atomic file writes, not one cross-file transaction: an interrupted
removal may clear that source's cache while retaining its feed definition.
No preference data is lost; the next online run can refetch.

The common cache policy applies: six-hour TTL, explicit `refresh` bypass,
last-known cache on failure, age/stale labels, topic filtering and article URL
deduplication at selection. Offline runs never resolve/fetch feeds and only use
their saved entries. Each feed has the usual 2 MiB body/100-entry bound and
timeouts; at most nine sources fetch concurrently across built-ins and customs.
Scheduled refresh loads custom feeds from the same configuration and does not
advance display history.

Custom `release` items also honor the optional per-run `--stable-only` flag,
using the same conservative tag/title rules as built-ins. Unknown-format
release titles are excluded in that mode; custom `news` entries are unaffected.
No items are deleted from the source cache by filtering.

`doctor` checks configuration locally and reports custom cache ages as
`custom-1`, `custom-2`, etc., in saved configuration order. It does not expose
their names, IDs, topics or URLs; invalid configuration is a nonzero error.
These aliases are diagnostic positions, not stable source IDs.

## Verification

Tests use local RSS/TLS fixtures and injected DNS/dial boundaries, not public
feed availability or actual private-network requests. They verify CLI
persistence/removal, bounds/duplicate handling, legacy profile preservation,
concurrent writes, cache identity/TTL/failure/offline behavior, doctor privacy,
sanitization, DNS pinning, mixed/private-address rejection, redirect policy,
network failures and cancellation. Production HTTPS certificate validation is
unchanged; the local TLS fixture supplies only its own test trust configuration.
