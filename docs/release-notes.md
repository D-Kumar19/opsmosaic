# OpsMosaic v0.1.0

Initial release of OpsMosaic. These notes are prepared for the GitHub draft;
downloads become public only when the maintainer publishes it.

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
- Interactive arrow-key topic selection with Space/Enter, cancellation,
  resize handling and a numbered fallback for noninteractive input.
- Local-only, read-only `doctor` checks with human/JSON reports and redacted home paths.
- Additive custom RSS/Atom feed configuration with existing-topic mapping,
  cache/offline integration and public-HTTPS DNS/redirect safeguards.
- Optional stable-release-only selection that retains news/tips and all cached
  entries, with unchanged default prerelease visibility.
- Complete last-briefing replay and persistent bookmarks, labeled saved/not live,
  without network, greeting or history changes.

## Installation

After publication, open
[the v0.1.0 release](https://github.com/D-Kumar19/opsmosaic/releases/tag/v0.1.0).
Download `SHA256SUMS` and the archive matching `uname -s` / `uname -m`:

| System | Architecture | Archive suffix |
|---|---|---|
| macOS | Apple Silicon (`arm64`) | `darwin_arm64.tar.gz` |
| macOS | Intel (`x86_64`) | `darwin_amd64.tar.gz` |
| Linux | ARM64 (`aarch64`) | `linux_arm64.tar.gz` |
| Linux | x86-64 (`x86_64`) | `linux_amd64.tar.gz` |

For example, on Apple Silicon, in the download directory:

```sh
grep ' ./opsmosaic_v0.1.0_darwin_arm64.tar.gz$' SHA256SUMS > selected.sha256
shasum -a 256 -c selected.sha256
# Continue only if verification reports OK.
mkdir opsmosaic-v0.1.0
tar -xzf opsmosaic_v0.1.0_darwin_arm64.tar.gz -C opsmosaic-v0.1.0
mkdir -p "$HOME/.local/bin"
install -m 0755 opsmosaic-v0.1.0/opsmosaic "$HOME/.local/bin/opsmosaic"
export PATH="$HOME/.local/bin:$PATH"
opsmosaic --version
opsmosaic
```

Use your matching filename; Linux can use `sha256sum -c selected.sha256`.
Archives contain the executable, README, changelog, MIT license and dependency
licenses. No Go runtime or API key is required. Persist the PATH setting in your
shell configuration if needed. Source installation remains available:

```sh
git clone https://github.com/D-Kumar19/opsmosaic.git
cd opsmosaic
git checkout v0.1.0
make install
```

Source builds require Go 1.24+ and `make`; the version tag exists after
publication. macOS binaries are not Apple-signed or notarized. If macOS blocks
execution, review its warning; building from source is an alternative.

See the [terminal demo](https://github.com/D-Kumar19/opsmosaic#see-it-in-action).

## Verification and limitations

- Latest local result: 157 passing Go cases including subtests, no failures/skips,
  race detector and vet passed; core statement coverage 87.7%.
- Five demo-tool tests pass; the generated GIF contains three verified frames.
- Local archive tests cover macOS/Linux arm64/amd64 builds, metadata,
  documentation/licenses, checksums/tampering and packaging failure guards.
- Hosted CI passed on Linux amd64 (Go 1.24 and latest stable) and macOS arm64
  (latest stable). Linux amd64 archive download/checksum/version checks passed
  in the draft workflow. macOS amd64 and Linux arm64 are cross-build targets,
  not native-runtime-certified.
- Feed availability is not guaranteed, and AI coverage is a lightweight
  headline keyword filter, not generated summaries.
- Checksums detect corruption, not publisher authenticity; production signing
  identity/trust and real release signatures are not configured.
- Real bash/zsh PTYs and simulated launchctl lifecycle tests passed. No actual
  background job was enabled during development verification.
- No adoption or general performance claims are made.
