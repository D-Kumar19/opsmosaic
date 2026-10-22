# Release policy

## Scope and support

Project repository: `https://github.com/D-Kumar19/opsmosaic`.
The source is MIT-licensed to Dheeraj Kumar (2026). This license does not grant
rights to third-party article bodies. Releases require maintainer approval.

Target initial version: `v0.1.0`. No tag or release exists merely because that
version appears in documentation or local test artifacts.

| Target | Packaging target | Runtime verification |
|---|---|---|
| macOS arm64 | Yes | Local development/test host |
| macOS amd64 | Yes | Cross-build only; native runner pending |
| Linux arm64 | Yes | Cross-build only; native runner pending |
| Linux amd64 | Yes | Cross-build only; native runner pending |

The platform scope is a release target, not a claim that untested targets are
certified. Go 1.24 is the contributor minimum. The intended CI matrix tests
1.24.x and latest stable; until runners are assigned, record the actual local
compiler in verification evidence. Installed binaries require no Go runtime.

## Version output

```sh
opsmosaic --version
# opsmosaic dev (commit unknown; built unknown)
```

Source builds default to `dev`. Release tooling embeds the requested version,
full commit and UTC build time through Go linker variables. `--version` makes
no network requests or state writes and needs no profile. It cannot be combined
with briefing options/subcommands.

## Local packaging

From a clean, committed checkout:

```sh
make ci
make test-release
make release VERSION=v0.1.0
```

This creates `dist/v0.1.0/` containing four
`opsmosaic_v0.1.0_<os>_<arch>.tar.gz` archives and `SHA256SUMS`.
Each archive contains the executable, README, LICENSE, CHANGELOG and dependency
license notices. `dist/` is ignored by Git. Packaging never creates a tag,
contacts a release API, or uploads artifacts. Output is staged before being
renamed into place; an existing version directory is never overwritten.

The build needs the approved Go modules in its cache or reachable from its
configured module proxy. No release-host integration is configured.

Verify on macOS with `shasum -a 256 -c SHA256SUMS`, or on Linux with
`sha256sum -c SHA256SUMS`, inside the release directory. Extract the matching
archive and install its executable into a directory on PATH. Test `--version`
and a disposable profile before distribution. Checksums detect corruption;
they are not signatures or publisher authentication. Optional
[OpenSSH signing/verification tools](signing.md) have disposable-key tests but
no approved production signing identity. Builds include a real
build timestamp; byte-for-byte reproducible archives are not claimed.

## Approval gates

Before an actual release:

1. Confirm repository access and the maintainer-approved artifact destination.
2. Unblock [CI](ci.md), run the Go/OS matrix, and review failing checks.
3. Validate archives, dependency notices, checksums and native platform behavior.
4. Move appropriate changelog entries into a dated version section.
5. Get explicit approval, then tag the reviewed commit using SemVer.
6. Separately approve the push and upload to the chosen release destination.

Use `vMAJOR.MINOR.PATCH` tags; supported prereleases are `-alpha.N`, `-beta.N`,
or `-rc.N` with positive N. Do not change an existing release tag; publish a
new patch version. Until 1.0, document breaking changes explicitly.
The Homebrew tap was declined; `make install` remains the supported path.
Signing/provenance, artifact-host integration and publication remain separate,
unapproved tasks. A clean-tree requirement prevents packaging uncommitted code.

[Draft release notes](release-notes.md) and a [project summary](project-summary.md)
are prepared for maintainer review, with the [terminal demo](demo/README.md).
These materials do not satisfy the outstanding runner/native-test gates or
authorize a release.
