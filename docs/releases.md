# Release policy

## Scope and support

Project repository: `https://github.com/D-Kumar19/opsmosaic`.
The source is MIT-licensed to Dheeraj Kumar (2026). This license does not grant
rights to third-party article bodies. Releases require maintainer approval.

Target initial version: `v0.1.0`. No tag or release exists merely because that
version appears in documentation or local test artifacts.

Verified draft `v0.1.0` targets commit `98fd72e`; [workflow run
39756397057](https://github.com/D-Kumar19/opsmosaic/actions/runs/39756397057)
uploaded four archives and `SHA256SUMS`, downloaded and checked them, and ran
the Linux amd64 version smoke test. It remains a draft, not a public release.
Final release materials and publication approval are tracked by T035.

| Target | Packaging target | Runtime verification |
|---|---|---|
| macOS arm64 | Yes | Local development/test host and hosted macOS CI |
| macOS amd64 | Yes | Cross-build only; native runner pending |
| Linux arm64 | Yes | Cross-build only; native runner pending |
| Linux amd64 | Yes | Hosted Linux CI and native archive smoke |

The platform scope is a release target, not a claim that untested targets are
certified. Go 1.24 is the contributor minimum. CI tests 1.24.x on Linux and
latest stable on Linux/macOS. Installed binaries require no Go runtime.

## Manual GitHub draft releases

The [Draft release workflow](../.github/workflows/release.yml) is manually
triggered; pushes and tags do not publish anything.

1. Commit and push the intended source to `main`; wait for its CI to pass.
2. Open **Actions -> Draft release -> Run workflow**, select `main` and enter a
   new version such as `v0.1.0`.
3. The workflow requires successful CI for that exact commit, rejects existing
   tags/releases, reruns source/packaging checks, and builds four archives plus
   `SHA256SUMS`.
4. It creates a **draft** GitHub Release targeting the exact tested commit,
   downloads its five assets, verifies checksums, and smoke-tests Linux amd64.
   Prerelease versions are marked as prereleases. A failed verification leaves
   the draft for inspection, never publishes it, and does not overwrite it.
5. Review the assets, release notes, changelog and target commit. Only after
   approval, click **Publish release** in GitHub. Publishing makes the version
   and its tag public; draft creation does not create a public release tag.

The release job needs `contents: write` for draft assets and `actions: read`
to verify CI. It uses GitHub's temporary job token, not a personal access token.
No production signing, scheduled release, or automatic public publishing is
configured. If GitHub permissions reject draft creation, fix the reported
permission rather than adding a token to source.

Equivalent manual dispatch after setup:

```sh
GH_HOST=github.com gh workflow run release.yml \
  --repo D-Kumar19/opsmosaic --ref main -f version=v0.1.0
```

Local packaging and manual GitHub upload remain alternatives; the workflow is
convenience automation around the same scripts.

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
configured module proxy.

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
5. Prepare and verify a draft using the manual workflow above.
6. Get explicit publication approval before publishing the draft and its tag.

Use `vMAJOR.MINOR.PATCH` tags; supported prereleases are `-alpha.N`, `-beta.N`,
or `-rc.N` with positive N. Do not change an existing release tag; publish a
new patch version. Until 1.0, document breaking changes explicitly.
The Homebrew tap was declined; `make install` remains the supported path.
Production signing/provenance is outside the project scope and is not a release
gate. Public publication still requires approval after draft verification.
A clean-tree requirement prevents packaging uncommitted code.

[Draft release notes](release-notes.md) and a [project summary](project-summary.md)
are prepared for maintainer review, with the [terminal demo](demo/README.md).
These materials do not satisfy the outstanding runner/native-test gates or
authorize a release.
