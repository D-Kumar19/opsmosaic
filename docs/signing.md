# Optional local artifact signatures

This is prepared tooling, **not a signed release or an established publisher
identity**. T032 stays blocked on release readiness and an approved trusted-key
process. No production key, external signing service, Apple certificate,
notarization or build-provenance attestation has been configured.

## What is signed

OpenSSH `ssh-keygen -Y sign` signs the exact `SHA256SUMS` manifest using the
namespace `opsmosaic-release`. Verification first checks the manifest signature
against an independently trusted allowed-signers file, then verifies each listed
archive checksum. A signature proves possession of the trusted signing key,
not that the build is safe or that CI ran.

Requirements: an OpenSSH build supporting `-Y sign`/`-Y verify`, standard Unix
tools and `sha256sum` or macOS `shasum`. No Go dependency was added.

## Approved future use

After all release gates and key-management decisions are approved:

```sh
# PRIVATE_KEY remains outside the repository and release directory.
sh scripts/sign-release.sh dist/v0.1.0 /absolute/path/to/private_key

# Obtain the allowed-signers file through an independently trusted channel.
sh scripts/verify-release.sh dist/v0.1.0 /absolute/path/to/trusted_signers opsmosaic-publisher
```

An allowed-signers line has this form:

```text
opsmosaic-publisher namespaces="opsmosaic-release" ssh-ed25519 <TRUSTED_PUBLIC_KEY>
```

Do not simply trust a public key shipped alongside an untrusted archive.
Agree on publisher identity, public-key fingerprint distribution, key custody,
rotation and revocation before signing a real release. Passphrase-protected keys
may prompt through OpenSSH; use only an approved signing environment.

Signing checks archive hashes first, refuses an existing manifest signature,
and does not modify archives, tag, push or upload. Manifest entries must be
SHA-256 hashes plus simple local `.tar.gz` filenames; path traversal and
non-archive entries are rejected. This manifest is an explicit list, not a
claim that every unrelated file in a directory is trusted. Distribute only
the expected archives, manifest and signature.

## Disposable tests

```sh
make test-signing
```

Tests generate two temporary Ed25519 keys, sign fixture checksums, verify the
trusted identity, and reject unsigned input, wrong keys, modified manifests,
tampered archives, unsafe manifest paths and signature overwrite. Test keys are
removed with the isolated fixture; no real project artifact is signed.

Signing/checksum scripts run from the checkout, not from within an installed
OpsMosaic binary. The installed command needs no OpenSSH signing runtime.
