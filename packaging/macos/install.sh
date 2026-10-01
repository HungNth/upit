#!/usr/bin/env bash
set -euo pipefail

APP_SOURCE=""
INSTALL_DIRECTORY="/Applications"

usage() {
    cat <<'EOF'
Usage: packaging/macos/install.sh --app PATH [--install-directory PATH]

Installs Upit.app, registers the nested Finder Service, and prints the System Settings enablement path.
EOF
}

while (($# > 0)); do
    case "$1" in
        --app) APP_SOURCE="$2"; shift 2 ;;
        --install-directory) INSTALL_DIRECTORY="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$APP_SOURCE" || ! -d "$APP_SOURCE" ]]; then
    echo "--app must point to an existing Upit.app bundle." >&2
    exit 2
fi
if [[ "$(basename "$APP_SOURCE")" != "Upit.app" ]]; then
    echo "The app bundle must be named Upit.app." >&2
    exit 2
fi
service_app="$APP_SOURCE/Contents/Helpers/UpitFinderService.app"
if [[ ! -d "$service_app" ]]; then
    echo "Finder Service bundle is missing from Upit.app." >&2
    exit 1
fi

mkdir -p "$INSTALL_DIRECTORY"
installed_app="$INSTALL_DIRECTORY/Upit.app"
ditto "$APP_SOURCE" "$installed_app"

lsregister="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
if [[ ! -x "$lsregister" ]]; then
    echo "Launch Services registration tool is unavailable." >&2
    exit 1
fi
"$lsregister" -f "$installed_app"
"$lsregister" -f "$installed_app/Contents/Helpers/UpitFinderService.app"

cat <<EOF
Installed: $installed_app
Finder Service: Upload with Upit
If the action is not visible immediately, enable it in System Settings → Keyboard → Keyboard Shortcuts → Services → Files and Folders.
EOF
