#!/bin/sh
set -eu
if test "$#" -ne 1; then
	printf 'Usage: sh scripts/check-release.sh RELEASE_DIRECTORY\n' >&2
	exit 1
fi
cd -- "$1"
# Accept only checksum entries for simple archive filenames in this directory.
awk '
  NF != 2 { exit 1 }
  length($1) != 64 || $1 ~ /[^0-9a-fA-F]/ { exit 1 }
  $2 !~ /^(\.\/)?[A-Za-z0-9][A-Za-z0-9_.-]*\.tar\.gz$/ { exit 1 }
  seen[$2]++ { exit 1 }
  END { if (NR == 0) exit 1 }
' SHA256SUMS || { printf 'Invalid checksum manifest; expected hashes and local archive filenames only.\n' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum -c SHA256SUMS
else
	shasum -a 256 -c SHA256SUMS
fi
