# CI readiness

## Current status (2026-10-19)

Local entry point: `make ci`. **No remote CI workflow is active.**
Repository: `github.com/D-Kumar19/opsmosaic`; stay on project infrastructure.

Read-only repository API checks found:

- Actions enabled with selected actions; GitHub-owned actions are allowed.
- No self-hosted runners visible to this repository (`total_count: 0`).
- Hosted-runner inventory endpoint returned 404, so hosted availability is
  unverified, not assumed absent.
- project examples use team-specific labels or caller-supplied `runs-on` values.
  Those examples do not establish that this repository can use those runners.
- Repository API reports `private: false`. The user explicitly chose to retain
  current visibility on the project host. This is not proof of internet visibility
  and is not permission to mirror the code externally.

T027 remains blocked on a verified runner assignment and successful remote run.
Do not add `ubuntu-latest`/`macos-latest` assumptions, provision runners, install
credentials, or send code to external CI to work around this.

## Intended matrix

- Go 1.24.x (minimum) and the latest stable Go available at setup time.
- Native macOS and Linux runners; test both approved architectures where
  available and clearly distinguish cross-builds from runtime tests.
- Execute `make ci` for each configured Go/OS combination.
- Execute `make test-release` on an appropriate native runner.
- Use read-only repository permissions, approved action versions and bounded
  timeouts. Do not upload private logs/artifacts externally.

After runner confirmation, choose approved checkout/setup actions compatible
with the GitHub version and runner runtime. Then verify a real
successful run and an intentional failing check on a disposable test branch
before marking CI operational. Never advertise an unverified green badge.

## Local failure verification

The formatting gate must return nonzero on an unformatted Go file. Unit test,
vet, build, and installation failures propagate through `set -eu`. Release
tests also verify invalid versions, dirty-tree rejection, overwrite refusal,
checksum tampering detection and metadata without onboarding.
