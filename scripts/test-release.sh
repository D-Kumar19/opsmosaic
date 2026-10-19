#!/bin/sh
set -eu

project=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "${TMPDIR:-/tmp}/opsmosaic-release-test.XXXXXX")
cleanup() {
	# This directory is exclusively the fixture created by mktemp above.
	find "$temporary" -depth -delete
}
trap cleanup EXIT
mkdir "$temporary/fixture"
for directory in cmd internal scripts; do
	cp -R "$project/$directory" "$temporary/fixture/"
done
for file in go.mod go.sum Makefile .gitignore LICENSE README.md CHANGELOG.md; do
	cp "$project/$file" "$temporary/fixture/"
done
cd "$temporary/fixture"
git init -q
git add .
git -c user.name=OpsMosaic-Test -c user.email=opsmosaic-test@example.invalid \
	-c commit.gpgsign=false commit -qm "Release test fixture"
commit=$(git rev-parse HEAD)
go=${GO:-go}

for invalid in '' 0.1.0 v01.0.0 v0.1.0-rc.0 v0.1.0-rc.01 v0.1.0/extra; do
	if GO="$go" sh scripts/release.sh "$invalid" > "$temporary/error.txt" 2>&1; then
		printf 'Invalid version was accepted: %s\n' "$invalid" >&2
		exit 1
	fi
	grep -q 'Usage:' "$temporary/error.txt"
done
printf 'dirty\n' >> README.md
if GO="$go" sh scripts/release.sh v0.1.0 > "$temporary/error.txt" 2>&1; then
	printf 'Dirty source was accepted.\n' >&2
	exit 1
fi
grep -q 'clean committed' "$temporary/error.txt"
git show HEAD:README.md > README.md
GO="$go" sh scripts/release.sh v0.1.0
if GO="$go" sh scripts/release.sh v0.1.0 > "$temporary/error.txt" 2>&1; then
	printf 'Existing version output was overwritten.\n' >&2
	exit 1
fi
grep -q 'already exists' "$temporary/error.txt"
if GO=false sh scripts/release.sh v0.1.1 > "$temporary/error.txt" 2>&1; then
	printf 'Release build failure was hidden.\n' >&2
	exit 1
fi
test ! -e dist/v0.1.1
test -z "$(find dist -name '.release.*' -type d -print)"
(
	cd dist/v0.1.0
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum -c SHA256SUMS
	else
		shasum -a 256 -c SHA256SUMS
	fi
)
test "$(find dist/v0.1.0 -name '*.tar.gz' -type f | wc -l | tr -d ' ')" -eq 4
mkdir "$temporary/extracted"
for archive in dist/v0.1.0/*.tar.gz; do
	test "$(tar -tzf "$archive" | wc -l | tr -d ' ')" -eq 7
	for entry in opsmosaic README.md LICENSE CHANGELOG.md LICENSE.uniseg LICENSE.term LICENSE.sys; do
		tar -tzf "$archive" | grep -qx "$entry"
	done
	tar -xzf "$archive" -C "$temporary/extracted"
	for entry in README.md LICENSE CHANGELOG.md; do
		cmp "$entry" "$temporary/extracted/$entry"
	done
	for module in github.com/rivo/uniseg golang.org/x/term golang.org/x/sys; do
		directory=$("$go" list -m -f '{{.Dir}}' "$module")
		license="$directory/LICENSE"
		if test "$module" = github.com/rivo/uniseg; then license="$directory/LICENSE.txt"; fi
		cmp "$license" "$temporary/extracted/LICENSE.${module##*/}"
	done
	"$go" version -m "$temporary/extracted/opsmosaic" > "$temporary/build.txt"
	target=${archive#dist/v0.1.0/opsmosaic_v0.1.0_}
	target=${target%.tar.gz}
	grep -Fq "GOOS=${target%_*}" "$temporary/build.txt"
	grep -Fq "GOARCH=${target#*_}" "$temporary/build.txt"
	# -trimpath can omit linker flags from build info; inspect embedded values.
	LC_ALL=C grep -aFq 'v0.1.0' "$temporary/extracted/opsmosaic"
	LC_ALL=C grep -aFq "$commit" "$temporary/extracted/opsmosaic"
	LC_ALL=C grep -aEq '[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z' "$temporary/extracted/opsmosaic"
done
os=$("$go" env GOHOSTOS)
arch=$("$go" env GOHOSTARCH)
tar -xzf "dist/v0.1.0/opsmosaic_v0.1.0_${os}_${arch}.tar.gz" -C "$temporary/extracted"
test -x "$temporary/extracted/opsmosaic"
export OPSMOSAIC_HOME="$temporary/state"
"$temporary/extracted/opsmosaic" --version > "$temporary/version.txt"
grep -q "opsmosaic v0.1.0 (commit $commit; built " "$temporary/version.txt"
grep -Eq 'built [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z\)' "$temporary/version.txt"
test ! -e "$OPSMOSAIC_HOME"
printf 'Alex\n3\n' | "$temporary/extracted/opsmosaic" --tips --plain > "$temporary/briefing.txt"
test "$(grep -c '^TIP |' "$temporary/briefing.txt")" -eq 1

# Deliberate checksum corruption must be detected.
printf 'tampered' >> dist/v0.1.0/opsmosaic_v0.1.0_linux_arm64.tar.gz
if (
	cd dist/v0.1.0
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum -c SHA256SUMS
	else
		shasum -a 256 -c SHA256SUMS
	fi
) > "$temporary/checksums.txt" 2>&1; then
	printf 'Checksum verification accepted tampered archive.\n' >&2
	exit 1
fi

# Exercise the CI formatting failure gate before any builds/tests run.
printf 'package brief\nfunc unformatted( ) { }\n' > internal/brief/format_probe.go
if GO="$go" sh scripts/ci.sh > "$temporary/ci.txt" 2>&1; then
	printf 'CI accepted unformatted code.\n' >&2
	exit 1
fi
grep -q 'formatting check failed' "$temporary/ci.txt"
printf 'Release tests passed: four archives, notices, checksums/tamper detection, version metadata, native smoke, clean-tree/overwrite guards, and CI failure gate.\n'
