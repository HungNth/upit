#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

if [[ "$(uname -s)" != Darwin || "$(uname -m)" != arm64 ]]; then
    echo 'This artifact smoke requires a native macOS arm64 host.' >&2
    exit 1
fi
if GOFLAGS=-upit-invalid-package-smoke-flag make package VERSION="${1:-0.0.0}"; then
    echo 'Packaging unexpectedly accepted invalid Go build flags.' >&2
    exit 1
fi
shopt -s nullglob dotglob
failed_staging=(.build/package/*)
[[ ${#failed_staging[@]} == 0 ]]
version="${1:-0.0.0}"
make package VERSION="$version"
artifact="$PWD/dist/macos/upit-macos-arm64-$version.dmg"
read -r expected _ < "$artifact.sha256"
actual="$(shasum -a 256 "$artifact")"
[[ "${actual%% *}" == "$expected" ]]
[[ "$(plutil -extract version raw "$artifact.metadata.json")" == "$version" ]]
[[ "$(plutil -extract signed raw "$artifact.metadata.json")" == false ]]
[[ "$(plutil -extract notarized raw "$artifact.metadata.json")" == false ]]

work="$(mktemp -d "${TMPDIR:-/tmp}/upit-package-smoke.XXXXXX")"
mounted=0
cleanup() {
    if [[ "$mounted" == 1 ]]; then hdiutil detach "$work/mount" >/dev/null; fi
    rm -rf "$work"
}
trap cleanup EXIT
mkdir "$work/mount"
hdiutil attach -readonly -nobrowse -mountpoint "$work/mount" "$artifact" >/dev/null
mounted=1
bash packaging/macos/validate.sh --app "$work/mount/Upit.app"
"$work/mount/Upit.app/Contents/Helpers/upit" --help
[[ "$(plutil -extract CFBundleShortVersionString raw "$work/mount/Upit.app/Contents/Info.plist")" == "$version" ]]
shopt -s nullglob dotglob
staging=(.build/package/*)
[[ ${#staging[@]} == 0 ]]
for output in bin/*; do
    [[ "$output" == bin/upit ]]
done
echo 'macOS DMG checksum, metadata, packaged CLI, bundle, and staging smoke passed.'
