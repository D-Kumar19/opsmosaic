# OpsMosaic project summary

Personal-project overview, not a claim of a published release or adoption.
See the [demo](demo/README.md) and [draft release notes](release-notes.md).

## Purpose

OpsMosaic puts a small technical briefing in the terminal without adding a
dashboard, account or hosted backend. Users select interests and receive one
random tip, headline or release with an original source link. It is useful
intermittently and remains useful offline through bundled tips and local cache.

## Engineering decisions

| Decision | Benefit | Trade-off |
|---|---|---|
| Standalone Go CLI, standard library first | Simple installation and no runtime service | Small terminal/Unicode dependencies are still needed for robust layout |
| Dependency-free arrow-key topic picker with numbered fallback | Comfortable on interactive terminals while retaining pipe/SSH fallback | Raw terminal handling and resize restoration need dedicated tests |
| Equal available-kind selection before unseen preference | Seen tips stay eligible beside busy news feeds | A kind can repeat after its own items are exhausted |
| Per-source caching and partial-result merge | Feed failures do not discard useful previous content | Older items need explicit age/stale labels |
| Atomic state writes plus bounded process locking | Prevent lost concurrent updates | A busy process gets a read-only snapshot; first-time setup may need retry |
| Sanitized headlines and original links, no summaries | Limits untrusted terminal output and content copying | Users open the source to read the full story |
| Network-free terminal startup option | Keeps startup independent of live feeds | Fresh news needs a later explicit or ordinary online run |
| Local packaging without automatic publishing | Separates build verification from release approval | Runner and distribution setup are still outstanding |
| Independent saved snapshots and bookmarks | Replay remains available after cache/source removal | Explicit retention limits and separate-file partial saves need clear errors |

## Evidence, not promises

The latest recorded local result is 157 passing Go cases and 87.7% core statement
coverage, with race/vet/install checks and isolated release-package tests.
Five separate demo-tool tests validate the recording helper. Tests use temporary
state and local fixtures, not the user's profile or live feed availability.
See [test evidence](testing.md) for scope and commands.

Only macOS arm64 has native runtime verification here. Four cross-builds do not
establish full platform support, an operational remote pipeline or a published
release. Local startup samples are not general performance benchmarks; the
project has no measured adoption or productivity metrics.

## Current scope

Project repository: [D-Kumar19/opsmosaic](https://github.com/D-Kumar19/opsmosaic).
`make install` remains the supported installation method; the user explicitly
declined a Homebrew tap. Release notes are drafts, `v0.1.0` is a target, and
tagging/pushing/uploading still require separate approval.
