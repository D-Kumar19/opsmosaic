# CI readiness

## Current status (2026-10-22)

Repository: [D-Kumar19/opsmosaic](https://github.com/D-Kumar19/opsmosaic).
Local entry point: `make ci`. The [workflow](../.github/workflows/ci.yml) runs
on pushes to `main`, pull requests targeting `main`, and manual dispatch.
Actions is enabled according to the maintainer. No secrets or write token
permissions are required.

The [first hosted run](https://github.com/D-Kumar19/opsmosaic/actions/runs/39752877495)
passed both Linux jobs (Go 1.24.x and stable). The macOS stable job exposed a
bookmark-concurrency test that incorrectly required all contenders to acquire
the bounded lock within 250 ms. The test now accepts explicit busy rejections,
verifies successful writes are retained, and resubmits rejected writes only
after all concurrent writers finish. Production locking behavior is unchanged.
The corrected test passed 50 local race-enabled repetitions and `make ci`;
the [subsequent hosted run](https://github.com/D-Kumar19/opsmosaic/actions/runs/39753472332)
passed all three jobs on commit `65b9a64`. T027 is complete.

## Configured matrix

| Runner | Go | Checks |
|---|---|---|
| `ubuntu-24.04` | `1.24.x` (minimum) | `make ci` |
| `ubuntu-24.04` | Latest stable | `make ci test-release test-signing test-demo` |
| `macos-14` | Latest stable | `make ci test-shell` |

This three-job matrix tests the minimum on Linux and latest stable on both
operating systems without duplicating every expensive check. Runner architecture
is recorded in the job's Go version output; it is not a four-architecture runtime
certification. Packaging cross-builds four targets, but only the matching native
archive is smoke-tested. Remaining native architectures are release gates.

## Usage and safety

- Standard GitHub-hosted runners only; no larger paid runner types.
- Each job has a 15-minute timeout. New runs cancel superseded runs for the
  same branch or pull request; there is no scheduled recurring workload.
- Pushes to feature branches do not run CI unless a pull request is open.
- macOS jobs consume the account's allowance differently from Linux; 2,000
  included minutes are not 2,000 minutes of macOS execution. Review actual
  usage in GitHub billing and do not enable paid overage just for this workflow.
- Go module/build caching reduces repeated work. No artifact uploads, releases,
  production signing, personal profiles, or cloud credentials are involved.
- Actions are pinned to full commit hashes. Repository permissions are
  read-only and checkout does not retain credentials.

The successful Linux job also ran `make test-release`, which inserts an
unformatted Go file into an isolated fixture, requires the CI entry point to
return nonzero, and checks its formatting diagnostic. The first hosted run
demonstrated real test-failure propagation. Together these verify positive and
negative behavior without another intentionally failing pull request.

## Local failure verification

The formatting gate must return nonzero on an unformatted Go file. Unit test,
vet, build, and installation failures propagate through `set -eu`. Release
tests also verify invalid versions, dirty-tree rejection, overwrite refusal,
checksum tampering detection and metadata without onboarding.
