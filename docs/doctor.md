# Local diagnostics

```sh
opsmosaic doctor
opsmosaic doctor --json
opsmosaic doctor --help
```

`doctor` is read-only and local-only. It never fetches sources, starts onboarding,
acquires a state lock, repairs files, changes preferences/history or creates a
cache. It can run while another process owns the state lock. Atomic state files
are read independently; the report is a snapshot, not a transaction across both
files or a guarantee that another process cannot change them during inspection.

## Checks and privacy

- Version, commit, build time, OS/architecture, Go version and executable path.
- Config/cache paths, with the home directory replaced by `~`.
- Profile existence/readability/validation without names or selected interests.
- Cache readability/schema, built-in source entry counts, oldest retrieval age,
  stale/unknown timestamps and pending partial-fetch retry.
- Unknown cached source IDs counted without printing their names.
- Custom feed configuration validity and cache status under positional
  `custom-N` aliases, without feed names, IDs, topics or URLs.
- Saved briefing/bookmark schema validity, without saved content or identifiers.
  Missing saved state is normal; corrupt/unreadable state emits `saved_invalid`.

Headlines, item URLs, commands, history, profile values and raw parser/validation
errors are never printed. Paths outside home can still reveal directory names;
review the output before sharing it. Do not assume a diagnostic report is
anonymous. Terminal controls in displayed metadata/paths are sanitized.

The fixed built-in inventory plus anonymized custom-feed positions are shown,
not the profile's selected-topic list. `not_cached` means no saved entry, not a broken live source. An empty feed
can be freshly cached with zero items. Ages use the oldest item retrieval time,
falling back to the last complete source fetch for legacy items. Missing/future
retrieval times have unknown age; six hours or older is stale. Pending retry
means its cached deadline is still in the future, not that a background job exists.
Source reachability, scheduler state and network availability are not probed.

## Exit codes

- **0:** no errors. Missing profile/cache, empty cache, stale and unknown-age
  data are warnings and do not fail the command.
- **1:** corrupt/unsupported/unreadable state, inability to locate required
  local metadata, invalid arguments/configuration or output failure.

Reports explain suggested manual actions; nothing is reset automatically.
Invalid environment path configuration can fail during application startup,
before a report is produced. State errors discovered by doctor still produce
the complete report; the process then exits nonzero with a generic stderr
message. JSON stdout stays valid and does not include that stderr message.

## JSON contract

Schema version 1 has these top-level fields:

| Field | Shape |
|---|---|
| `schema_version` | Integer, currently `1` |
| `status` | `ok`, `warning`, or `error` |
| `version`, `commit`, `build_time` | Sanitized build metadata strings |
| `os`, `arch`, `go_version` | Runtime strings |
| `paths` | `executable`, `config`, `cache` strings |
| `checks` | Array of `{code, status, message}` |
| `sources` | Array of `{id, status, item_count, oldest_age_seconds, retry_pending}` |
| `unknown_cached_sources` | Count of cache keys not in the configured built-in/custom inventory |

Source status is `not_cached`, `fresh`, `stale` or `unknown`.
`oldest_age_seconds` is an integer or `null` when unknown/not cached.
Both arrays are always arrays, including when empty. Corrupt/missing cache
does not produce invented source health entries. Check codes such as
`profile_missing`, `profile_invalid`, `cache_missing`, `cache_unreadable` and
`cache_stale` identify findings; human message wording is not an API.

Tests cover exact top-level JSON fields, privacy sentinels, writer errors,
argument/cancellation behavior, held locks, stale/unknown boundaries and
unchanged state. The standalone installation test also exercises `doctor`.
