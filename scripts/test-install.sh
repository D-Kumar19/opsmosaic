#!/bin/sh
set -eu

project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "${TMPDIR:-/tmp}/opsmosaic-install.XXXXXX")
cleanup() {
	rm -f "$temporary/install prefix/bin/opsmosaic" "$temporary/staging/install prefix/bin/opsmosaic"
	rm -f "$temporary/state/config.json" "$temporary/state/config.json.lock" "$temporary/state/cache.json" "$temporary/state/feeds.json" "$temporary/state/saved.json" "$temporary/output.txt" "$temporary/doctor.json"
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
opsmosaic doctor --json > doctor.json
grep -q '"schema_version": 1' doctor.json
grep -q '"code": "profile_missing"' doctor.json
test ! -e "$OPSMOSAIC_HOME/config.json.lock"
opsmosaic shell greeting > output.txt
test ! -s output.txt
test ! -e "$OPSMOSAIC_HOME/config.json.lock"

printf 'Alex\n3,5\n' | opsmosaic --tips > output.txt
grep -q ', Alex!' output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 1
if grep -q 'Interests:' output.txt; then
	printf 'Installation test failed: default output is not compact.\n' >&2
	exit 1
fi
opsmosaic replay --plain > output.txt
grep -q 'Saved content' output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 1
opsmosaic bookmarks add > output.txt
bookmark_id=$(sed -n 's/^Bookmark add complete: //p' output.txt)
test "${#bookmark_id}" -eq 64
opsmosaic bookmarks list > output.txt
grep -q 'captured' output.txt
opsmosaic bookmarks show "$bookmark_id" --plain > output.txt
grep -q 'not live news' output.txt
opsmosaic bookmarks remove "$bookmark_id" > output.txt
opsmosaic bookmarks list > output.txt
grep -q 'None saved.' output.txt
opsmosaic --tips --count 3 > output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 3
opsmosaic shell greeting > output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 1
opsmosaic --tips --detailed --plain > output.txt
grep -q 'Showing 1 items' output.txt
opsmosaic --tips --stable-only > output.txt
test "$(grep -c '^TIP |' output.txt)" -eq 1
opsmosaic doctor > output.txt
grep -q 'profile_valid' output.txt
opsmosaic feeds add --id example --name "Example updates" --topic aws --kind news --url https://example.org/feed > output.txt
opsmosaic feeds list > output.txt
grep -q 'Example updates' output.txt
opsmosaic doctor --json > doctor.json
grep -q 'feeds_valid' doctor.json
opsmosaic feeds remove example > output.txt
opsmosaic feeds list > output.txt
grep -q 'None configured' output.txt

if opsmosaic --count 0 > output.txt 2>&1; then
	printf 'Installation test failed: invalid count accepted.\n' >&2
	exit 1
fi
printf 'Installation tests passed: standalone command, spaced prefix, staging, onboarding, replay/bookmarks, and options.\n'
