#!/bin/sh
set -eu
if test "$#" -ne 3; then
	printf 'Usage: sh scripts/verify-release.sh RELEASE_DIRECTORY TRUSTED_ALLOWED_SIGNERS IDENTITY\n' >&2
	exit 1
fi
directory=$(CDPATH= cd -- "$1" && pwd)
signers=$2
identity=$3
ssh-keygen -Y verify -f "$signers" -I "$identity" -n opsmosaic-release \
	-s "$directory/SHA256SUMS.sig" < "$directory/SHA256SUMS"
project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
sh "$project/scripts/check-release.sh" "$directory"
printf 'Trusted signature and artifact checksums verified.\n'
