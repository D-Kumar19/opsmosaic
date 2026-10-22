# CI readiness

## Current status (2026-10-22)

Repository: [D-Kumar19/opsmosaic](https://github.com/D-Kumar19/opsmosaic).
Local entry point: `make ci`. No remote workflow is committed in the current
checkout, and remote CI success has not been verified.

GitHub-hosted runners are a candidate for this personal repository. Before
marking T027 complete, configure the workflow, confirm Actions permissions and
the account's available minutes/storage, and verify actual runs. Repository
hosting alone does not establish that CI is operational.

## Intended matrix

- Go 1.24.x (minimum) and the latest stable Go available at setup time.
- Native macOS and Linux runners; test both target architectures where
  available and distinguish cross-builds from runtime tests.
- Execute `make ci` for each configured Go/OS combination.
- Execute `make test-release` on an appropriate native runner.
- Use read-only repository permissions, pinned action versions, bounded
  timeouts and short artifact retention. Keep credentials and private state
  out of logs and artifacts.

Verify a real successful run and an intentional failing check on a disposable
test branch before marking CI operational. Never advertise an unverified badge.

## Local failure verification

The formatting gate must return nonzero on an unformatted Go file. Unit test,
vet, build, and installation failures propagate through `set -eu`. Release
tests also verify invalid versions, dirty-tree rejection, overwrite refusal,
checksum tampering detection and metadata without onboarding.
