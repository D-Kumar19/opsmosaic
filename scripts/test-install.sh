#!/bin/sh
set -eu

project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "${TMPDIR:-/tmp}/opsmosaic-install.XXXXXX")
cleanup() {
	rm -f "$temporary/install prefix/bin/opsmosaic" "$temporary/staging/install prefix/bin/opsmosaic"
	rm -f "$temporary/state/config.json" "$temporary/state/config.json.lock" "$temporary/state/cache.json" "$temporary/output.txt"
	for directory in "$temporary/install prefix/bin" "$temporary/install prefix" \
		"$temporary/staging/install prefix/bin" "$temporary/staging/install prefix" \
		"$temporary/staging" "$temporary/state" "$temporary"; do
		if test -d "$directory"; then
			rmdir "$directory"
		fi
	done
}
trap cleanup EXIT

cd "$project"
make install PREFIX="$temporary/install prefix"
make install DESTDIR="$temporary/staging" PREFIX="/install prefix"
test -x "$temporary/staging/install prefix/bin/opsmosaic"

PATH="$temporary/install prefix/bin:$PATH"
OPSMOSAIC_HOME="$temporary/state"
export PATH OPSMOSAIC_HOME

cd "$temporary"
test "$(command -v opsmosaic)" = "$temporary/install prefix/bin/opsmosaic"
opsmosaic --help > output.txt
grep -q 'default 1' output.txt
test ! -e "$OPSMOSAIC_HOME/config.json"

printf 'Alex\n3,5\n' | opsmosaic --tips > output.txt
grep -q ', Alex!' output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 1
if grep -q 'Interests:' output.txt; then
	printf 'Installation test failed: default output is not compact.\n' >&2
	exit 1
fi
opsmosaic --tips --count 3 > output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 3
opsmosaic --tips --detailed --plain > output.txt
grep -q 'Showing 1 items' output.txt

if opsmosaic --count 0 > output.txt 2>&1; then
	printf 'Installation test failed: invalid count accepted.\n' >&2
	exit 1
fi
printf 'Installation tests passed: standalone command, spaced prefix, staging, onboarding, and options.\n'
