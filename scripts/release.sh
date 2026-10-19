#!/bin/sh
set -eu

cd "$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
version=${1:-}
if test "$#" -ne 1 || ! printf '%s\n' "$version" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta|rc)\.[1-9][0-9]*)?$'; then
	printf 'Usage: make release VERSION=vMAJOR.MINOR.PATCH[-alpha.N|-beta.N|-rc.N]\n' >&2
	exit 1
fi
if test -n "$(git status --porcelain --untracked-files=normal)"; then
	printf 'Release build requires a clean committed working tree. No tag or upload will be created.\n' >&2
	exit 1
fi
output="dist/$version"
if test -e "$output"; then
	printf 'Release output already exists: %s (refusing to overwrite)\n' "$output" >&2
	exit 1
fi
go=${GO:-go}
commit=$(git rev-parse HEAD)
timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
mkdir -p dist
temporary=$(mktemp -d dist/.release.XXXXXX)
cleanup() {
	rm -f "$temporary/package/opsmosaic" "$temporary/package/README.md" \
		"$temporary/package/LICENSE" "$temporary/package/CHANGELOG.md"
	for module in uniseg term sys; do rm -f "$temporary/package/LICENSE.$module"; done
	if test -d "$temporary/package"; then rmdir "$temporary/package"; fi
	# Remove only the explicitly named temporary release outputs.
	for target in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64; do
		rm -f "$temporary/opsmosaic_${version}_${target}.tar.gz"
	done
	rm -f "$temporary/SHA256SUMS"
	if test -d "$temporary"; then rmdir "$temporary"; fi
}
trap cleanup EXIT
mkdir "$temporary/package"
"$go" mod download
for module in github.com/rivo/uniseg golang.org/x/term golang.org/x/sys; do
	directory=$("$go" list -m -f '{{.Dir}}' "$module")
	license="$directory/LICENSE"
	if test "$module" = github.com/rivo/uniseg; then license="$directory/LICENSE.txt"; fi
	cp "$license" "$temporary/package/LICENSE.${module##*/}"
done
for target in darwin_arm64 darwin_amd64 linux_arm64 linux_amd64; do
	os=${target%_*}
	arch=${target#*_}
	CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" "$go" build -trimpath \
		-ldflags "-X opsmosaic/internal/brief.Version=$version -X opsmosaic/internal/brief.Commit=$commit -X opsmosaic/internal/brief.BuildTime=$timestamp" \
		-o "$temporary/package/opsmosaic" ./cmd/opsmosaic
	cp README.md LICENSE CHANGELOG.md "$temporary/package/"
	tar -czf "$temporary/opsmosaic_${version}_${target}.tar.gz" -C "$temporary/package" opsmosaic README.md LICENSE CHANGELOG.md LICENSE.uniseg LICENSE.term LICENSE.sys
done
(
	cd "$temporary"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum ./*.tar.gz > SHA256SUMS
	else
		shasum -a 256 ./*.tar.gz > SHA256SUMS
	fi
)
rm "$temporary/package/opsmosaic" "$temporary/package/README.md" "$temporary/package/LICENSE" "$temporary/package/CHANGELOG.md"
for module in uniseg term sys; do rm "$temporary/package/LICENSE.$module"; done
rmdir "$temporary/package"
mv "$temporary" "$output"
printf 'Local release artifacts: %s\nCommit: %s\nNothing tagged, uploaded, or published.\n' "$output" "$commit"
