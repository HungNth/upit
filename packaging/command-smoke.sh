#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Run explicitly: this check removes generated output, never source or configuration.
make clean
make build
case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) cli=bin/upit.exe ;;
    *) cli=bin/upit ;;
esac
"./$cli" --help
shopt -s nullglob dotglob
outputs=(bin/*)
if [[ ${#outputs[@]} != 1 || ${outputs[0]} != "$cli" ]]; then
    echo "bin must contain only $cli" >&2
    exit 1
fi
mkdir -p .build/package/interrupted dist/command-smoke
if [[ "$(uname -s)" == Linux ]]; then
    if diagnostic=$(make package 2>&1); then
        echo 'Linux Desktop packaging unexpectedly succeeded.' >&2
        exit 1
    fi
    if [[ "$diagnostic" != *'Desktop packaging is not supported on Linux'* ]]; then
        printf 'Unsupported-platform diagnostic missing: %s\n' "$diagnostic" >&2
        exit 1
    fi
fi
printf 'sentinel\n' > .build/package/interrupted/sentinel
printf 'sentinel\n' > dist/command-smoke/sentinel
make clean
for output in bin .build dist; do
    if [[ -e "$output" ]]; then
        echo "make clean left generated output: $output" >&2
        exit 1
    fi
done
make clean
echo 'CLI build and idempotent cleanup smoke passed.'
