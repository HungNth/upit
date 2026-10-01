#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIRECTORY="/Applications"

usage() {
    cat <<'EOF'
Usage: packaging/macos/uninstall.sh [--install-directory PATH]

Unregisters the Upit Finder Service and removes the installed Upit.app bundle.
EOF
}

while (($# > 0)); do
    case "$1" in
        --install-directory) INSTALL_DIRECTORY="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

installed_app="$INSTALL_DIRECTORY/Upit.app"
lsregister="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
if [[ ! -x "$lsregister" ]]; then
    echo "Launch Services registration tool is unavailable." >&2
    exit 1
fi

if [[ -d "$installed_app" ]]; then
    service_app="$installed_app/Contents/Helpers/UpitFinderService.app"
    "$lsregister" -u "$service_app" || true
    "$lsregister" -u "$installed_app" || true
    rm -rf "$installed_app"
fi

if [[ -e "$installed_app" ]]; then
    echo "Upit.app remains after uninstall." >&2
    exit 1
fi

echo "Removed Upit.app and unregistered Upload with Upit."
