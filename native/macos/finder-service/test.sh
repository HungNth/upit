#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_BINARY="$(mktemp "${TMPDIR:-/tmp}/upit-finder-selection-test.XXXXXX")"
trap 'rm -f "$TEST_BINARY"' EXIT

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Finder Service native tests require macOS." >&2
    exit 1
fi

clang -fobjc-arc -mmacosx-version-min=14.0 -framework Foundation \
    "$SOURCE_DIRECTORY/selection.m" \
    "$SOURCE_DIRECTORY/selection_test.m" \
    -o "$TEST_BINARY"
"$TEST_BINARY"
echo "Finder Service native selection tests passed."
