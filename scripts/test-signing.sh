#!/bin/sh
set -eu
project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "${TMPDIR:-/tmp}/opsmosaic-sign-test.XXXXXX")
trap 'find "$temporary" -depth -delete' EXIT
mkdir "$temporary/release"
ssh-keygen -q -t ed25519 -N '' -C 'disposable-test-only' -f "$temporary/key"
ssh-keygen -q -t ed25519 -N '' -C 'wrong-test-only' -f "$temporary/wrong"
printf 'opsmosaic-test %s\n' "$(cat "$temporary/key.pub")" > "$temporary/signers"
printf 'opsmosaic-test %s\n' "$(cat "$temporary/wrong.pub")" > "$temporary/wrong-signers"
printf 'fixture archive\n' > "$temporary/release/example.tar.gz"
(
	cd "$temporary/release"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum ./example.tar.gz > SHA256SUMS
	else
		shasum -a 256 ./example.tar.gz > SHA256SUMS
	fi
)
verify() { sh "$project/scripts/verify-release.sh" "$temporary/release" "$1" opsmosaic-test; }
if verify "$temporary/signers" > "$temporary/error" 2>&1; then
	printf 'Unsigned release accepted.\n' >&2; exit 1
fi
sh "$project/scripts/sign-release.sh" "$temporary/release" "$temporary/key"
verify "$temporary/signers"
if verify "$temporary/wrong-signers" > "$temporary/error" 2>&1; then
	printf 'Wrong key accepted.\n' >&2; exit 1
fi
if sh "$project/scripts/sign-release.sh" "$temporary/release" "$temporary/key" > "$temporary/error" 2>&1; then
	printf 'Existing signature overwritten.\n' >&2; exit 1
fi
cp "$temporary/release/SHA256SUMS" "$temporary/original"
printf 'changed\n' >> "$temporary/release/SHA256SUMS"
if verify "$temporary/signers" > "$temporary/error" 2>&1; then
	printf 'Changed manifest accepted.\n' >&2; exit 1
fi
cp "$temporary/original" "$temporary/release/SHA256SUMS"
sed 's|example.tar.gz|../example.tar.gz|' "$temporary/original" > "$temporary/release/SHA256SUMS"
if sh "$project/scripts/check-release.sh" "$temporary/release" > "$temporary/error" 2>&1; then
	printf 'Out-of-directory manifest path accepted.\n' >&2; exit 1
fi
cp "$temporary/original" "$temporary/release/SHA256SUMS"
printf 'tampered\n' >> "$temporary/release/example.tar.gz"
if verify "$temporary/signers" > "$temporary/error" 2>&1; then
	printf 'Tampered archive accepted.\n' >&2; exit 1
fi
printf 'Signing tests passed: trusted key, unsigned/wrong key/manifest/archive rejection, overwrite guard. Test keys deleted on exit.\n'
