# Draft release notes: v0.1.0

**DRAFT — not tagged, uploaded or released.**
Do not describe this version as available for download.

## Overview

OpsMosaic is a standalone terminal briefing for DevOps, cloud, security and AI.
Choose a name and interests once, then get one randomly selected practical tip,
news headline or project release, with a source link.

## Highlights

- Nine topics: Docker, Kubernetes, AWS, Security, Helm, Argo CD, Kargo,
  Crossplane and AI.
- Local-time greetings with rotating comments; compact, detailed and plain
  output, optional color and Unicode-aware prose wrapping.
- Bundled read-only tips, RSS/Atom headlines, title-filtered Hacker News stories
  and upstream release links. No copied article bodies or generated summaries.
- Six-hour local caching, offline operation, partial Hacker News recovery,
  stale-content age labels and bounded cross-process state locking.
- Standalone installation with `make install`; no account, API key, backend or
  Go runtime required after building.
- Version metadata and local four-target packaging with dependency licenses
  and SHA-256 checksums.
- Explicit zsh/bash hook installation/removal, plus opt-in macOS three-hour
  cache refresh that preserves greeting rotation and display history.
- Prepared local OpenSSH checksum-manifest signing and trusted-key verification,
  tested only with disposable keys.
- Local-only, read-only `doctor` checks with human/JSON reports and redacted home paths.
- Additive custom RSS/Atom feed configuration with existing-topic mapping,
  cache/offline integration and public-HTTPS DNS/redirect safeguards.
- Optional stable-release-only selection that retains news/tips and all cached
  entries, with unchanged default prerelease visibility.
- Complete last-briefing replay and persistent bookmarks, labeled saved/not live,
  without network, greeting or history changes.

## Installation and demo

Users with repository access can follow the
[source installation instructions](../README.md#install-on-macos).
`make install` is the supported installation path; no Homebrew tap is planned.
See the [animated terminal demo and transcript](demo/README.md).

## Verification and limitations

- Latest local result: 157 passing Go cases including subtests, no failures/skips,
  race detector and vet passed; core statement coverage 87.7%.
- Five demo-tool tests pass; the generated GIF contains three verified frames.
- Local archive tests cover macOS/Linux arm64/amd64 builds, metadata,
  documentation/licenses, checksums/tampering and packaging failure guards.
- Runtime testing is currently macOS arm64 on Go 1.27.2. Other target binaries
  were cross-built, not native-runtime-certified. Go 1.24 is the declared
  minimum, but its matrix run remains unverified.
- Remote CI configuration and verified runs remain outstanding. Feed availability is not
  guaranteed, and AI coverage is a lightweight headline keyword filter.
- Checksums detect corruption, not publisher authenticity; production signing
  identity/trust and real release signatures are not configured.
- Real bash/zsh PTYs and simulated launchctl lifecycle tests passed. No actual
  background job was enabled during development verification.
- No adoption or general performance claims are made.

## Before these notes can become a real release

Complete the [CI and release approval gates](releases.md#approval-gates),
verify the native target matrix, date the changelog, and obtain explicit
tag/push/upload approval and a confirmed release artifact destination.
T035 remains blocked until its release prerequisites are met.
