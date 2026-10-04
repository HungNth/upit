#!/usr/bin/env bash
set -euo pipefail

SOURCE_DIRECTORY="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_BINARY="$(mktemp "${TMPDIR:-/tmp}/upit-notification-test.XXXXXX")"
trap 'rm -f "$TEST_BINARY"' EXIT

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Native notification tests require macOS." >&2
    exit 1
fi

clang -mmacosx-version-min=14.0 -framework Cocoa -framework UserNotifications \
    "$SOURCE_DIRECTORY/../../../cmd/upit-file-manager/feedback_darwin.m" \
    "$SOURCE_DIRECTORY/notification_test.m" -o "$TEST_BINARY"
"$TEST_BINARY"
