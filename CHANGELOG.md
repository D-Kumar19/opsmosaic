# Changelog

Notable changes are recorded here in a Keep a Changelog-style format.
Versions follow Semantic Versioning. No release has been tagged yet.

## [Unreleased]

### Added

- Personalized local-time greetings and a single random tip, headline or release.
- Docker, Kubernetes, AWS, Security, Helm, Argo CD, Kargo, Crossplane and AI topics.
- Local onboarding, offline caching, recent-item history and atomic state storage.
- Partial Hacker News recovery, short retry intervals and stale-cache age labels.
- Cross-process locking with a bounded, read-only startup fallback.
- Compact Unicode-aware layout, detailed/plain modes and optional color.
- Version/build information, local release packaging and SHA-256 checksums.
- Portable local CI checks and contributor documentation.
- Explicit zsh/bash hook setup/removal and offline, no-onboarding startup.
- Opt-in macOS three-hour cache refresh with logs and unchanged display history.
- Optional local OpenSSH checksum-manifest signing and trusted-key verification.
- Read-only local `doctor` diagnostics with privacy-conscious text/JSON reports.
- Additive custom RSS/Atom feeds with topic/kind mapping, cache/offline support
  and public-HTTPS connection/redirect safeguards.
- Opt-in `--stable-only` filtering of prerelease/unknown release labels without
  changing default content, news/tips or saved source entries.
- Persistent complete-briefing replay and bookmark add/list/show/remove, with
  original source/capture metadata, no-network saved output and no eviction.
- Interactive arrow-key topic selection with Space/Enter, live resize support,
  cancellation without profile loss, and a numbered non-TTY fallback.

### Changed

- Use personal-project repository links, contribution and release documentation.
- Use `io.github.d-kumar19.opsmosaic.refresh` for new macOS refresh agents.
  Disable any differently named existing job with its previous build before
  enabling the new identifier.

### Planned release

- Target: `v0.1.0`, pending runner verification and manual release approval.
