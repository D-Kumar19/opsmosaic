#!/bin/sh
set -eu
if test "$#" -ne 2; then
	printf 'Usage: sh scripts/sign-release.sh RELEASE_DIRECTORY PRIVATE_KEY\n' >&2
	exit 1
fi
directory=$(CDPATH= cd -- "$1" && pwd)
key=$2
test -f "$key" || { printf 'Signing key not found.\n' >&2; exit 1; }
test -f "$directory/SHA256SUMS" || { printf 'SHA256SUMS not found.\n' >&2; exit 1; }
test ! -e "$directory/SHA256SUMS.sig" || { printf 'Signature already exists; refusing overwrite.\n' >&2; exit 1; }
project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
sh "$project/scripts/check-release.sh" "$directory"
ssh-keygen -Y sign -f "$key" -n opsmosaic-release "$directory/SHA256SUMS"
printf 'Signed checksum manifest only. Nothing tagged or published.\n'
