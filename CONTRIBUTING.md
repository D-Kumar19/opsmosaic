# Contributing

OpsMosaic stays on the project-hosted repository. Contributions and reports are for
people who already have access; do not mirror source or diagnostics to external
services. No external publication or release upload is authorized.

## Development

Install Go 1.24 or newer, Git, make, and standard Unix tools. On macOS, see
the [installation instructions](README.md#install-on-macos).

```sh
git clone https://github.com/D-Kumar19/opsmosaic.git
cd opsmosaic
git switch -c fix/short-description
make ci
make test-release
```

`make ci` checks formatting, module checksums, race-tested unit/integration
tests, vet, the build and standalone installation. It fails on the first
failure. `make test-release` creates an isolated committed fixture and checks
all four archives, checksums, metadata, clean-tree guards and a native binary.
Neither command modifies your actual profile or publishes anything.

Use `OPSMOSAIC_HOME` pointing at a temporary directory for manual experiments.
Never execute commands merely because they appear in feed content. Keep one
content item as the default; preserve offline/no-network behavior.

## Changes and tests

- Prefer small, focused changes with regression tests and related documentation.
- Ask about unresolved product or compatibility decisions before implementing.
- Format Go files with `gofmt`; add no dependency without a concrete need.
- Use local HTTP fixtures, not live internet, for automated feed tests.
- Add sourced, read-only tips in `internal/brief/tips.go`, noting prerequisites.
- Feed adapters must sanitize titles, validate links, bound requests, and expose
  failures. Test partial/total failure, cache fallback and source attribution.
- Treat terminal controls and private diagnostic values as untrusted input.

## Reports and review

Use [repository issues](https://github.com/D-Kumar19/opsmosaic/issues) for bugs.
Include command, `opsmosaic --version`, OS/architecture, terminal type and a
minimal reproduction. Redact names, private URLs, tokens, and other sensitive
data; do not attach your profile, cache or shell configuration by default.

Submit changes through the repository's pull-request interface. Explain behavior
changes, test evidence, and compatibility risks. Host-specific issue/PR templates
are deferred. Passing local checks is not a substitute for review or a live CI
result; see [CI status](docs/ci.md).

## Releases

See [release policy](docs/releases.md). Keep notes under Unreleased until an
approved release. Creating a branch or building local artifacts does not
authorize tagging, pushing, publishing, changing visibility, or provisioning
runners.
