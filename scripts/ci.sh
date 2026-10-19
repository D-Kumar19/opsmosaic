#!/bin/sh
set -eu

cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
go=${GO:-go}
unformatted=$(find cmd internal -name '*.go' -type f -exec gofmt -l {} +)
if test -n "$unformatted"; then
	printf 'Go formatting check failed:\n%s\n' "$unformatted" >&2
	exit 1
fi
printf 'Checking with %s\n' "$("$go" version)"
"$go" mod verify
"$go" test -race -cover ./...
"$go" vet ./...
make build GO="$go"
make test-install GO="$go"
