#!/usr/bin/env bash
set -euo pipefail

APP_PATH=""
REQUIRE_SIGNATURE=0

usage() {
    cat <<'EOF'
Usage: packaging/macos/validate.sh --app PATH [--require-signature]

Validates the macOS 14+ Apple Silicon bundle layout, Finder Service metadata, and optional signatures.
EOF
}

while (($# > 0)); do
    case "$1" in
        --app) APP_PATH="$2"; shift 2 ;;
        --require-signature) REQUIRE_SIGNATURE=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$APP_PATH" || ! -d "$APP_PATH" ]]; then
    echo "--app must point to an existing Upit.app bundle." >&2
    exit 2
fi
if [[ "$(basename "$APP_PATH")" != "Upit.app" ]]; then
    echo "The app bundle must be named Upit.app." >&2
    exit 2
fi

service_app="$APP_PATH/Contents/Helpers/UpitFinderService.app"
file_manager_app="$service_app/Contents/Helpers/UpitFileManager.app"
root_info="$APP_PATH/Contents/Info.plist"
service_info="$service_app/Contents/Info.plist"
file_manager_info="$file_manager_app/Contents/Info.plist"
service_binary="$service_app/Contents/MacOS/UpitFinderService"
desktop_binary="$APP_PATH/Contents/MacOS/upit-desktop"
file_manager_binary="$file_manager_app/Contents/MacOS/upit-file-manager"
cli_binary="$APP_PATH/Contents/Helpers/upit"
icon_resource="$APP_PATH/Contents/Resources/AppIcon.icns"

for input in "$root_info" "$service_info" "$file_manager_info" "$service_binary" "$desktop_binary" "$file_manager_binary" "$cli_binary" "$icon_resource"; do
    if [[ ! -e "$input" ]]; then
        echo "Required package member is missing: $input" >&2
        exit 1
    fi
done

for plist in "$root_info" "$service_info" "$file_manager_info"; do
    plutil -lint "$plist" >/dev/null
done

product_version="$(plutil -extract CFBundleShortVersionString raw -o - "$root_info")"
build_version="$(plutil -extract CFBundleVersion raw -o - "$root_info")"
if [[ ! "$product_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ || ! "$build_version" =~ ^[0-9]+(\.[0-9]+){0,3}$ ]]; then
    echo "Package version metadata must be numeric and fully rendered." >&2
    exit 1
fi
for plist in "$root_info" "$service_info" "$file_manager_info"; do
    if [[ "$(plutil -extract CFBundleName raw -o - "$plist")" != "Upit" ||
          "$(plutil -extract CFBundleShortVersionString raw -o - "$plist")" != "$product_version" ||
          "$(plutil -extract CFBundleVersion raw -o - "$plist")" != "$build_version" ||
          "$(plutil -extract LSMinimumSystemVersion raw -o - "$plist")" != "14.0" ]]; then
        echo "Package components must share the Upit name, version, and minimum platform." >&2
        exit 1
    fi
done

if [[ "$(plutil -extract CFBundleIconFile raw -o - "$root_info")" != "AppIcon" ]]; then
    echo "Desktop bundle must declare CFBundleIconFile as AppIcon." >&2
    exit 1
fi
if [[ ! -s "$icon_resource" ]]; then
    echo "Desktop bundle AppIcon.icns must exist and be non-empty." >&2
    exit 1
fi

service_name="$(plutil -extract 'NSServices.0.NSMenuItem.default' raw -o - "$service_info")"
if [[ "$service_name" != "Upload with Upit" ]]; then
    echo "Finder Service name is incorrect." >&2
    exit 1
fi
message="$(plutil -extract 'NSServices.0.NSMessage' raw -o - "$service_info")"
if [[ "$message" != "uploadFileService" ]]; then
    echo "Finder Service message selector is incorrect." >&2
    exit 1
fi
# Any extra URL type (e.g. public.file-url) makes System Settings list the service under Internet.
send_file_types="$(plutil -extract 'NSServices.0.NSSendFileTypes' json -o - "$service_info" 2>/dev/null || true)"
if [[ "$send_file_types" != '["public.item"]' ]]; then
    echo "Finder Service must accept exactly public.item to stay in Files and Folders." >&2
    exit 1
fi
for plist in "$service_info" "$file_manager_info"; do
    background="$(plutil -extract 'LSBackgroundOnly' raw -o - "$plist")"
    if [[ "$background" != "true" ]]; then
        echo "Background-only metadata is missing from $plist." >&2
        exit 1
    fi
done

for binary in "$service_binary" "$desktop_binary" "$file_manager_binary" "$cli_binary"; do
    if [[ ! -x "$binary" ]]; then
        echo "Package member is not executable: $binary" >&2
        exit 1
    fi
    if [[ "$(lipo -archs "$binary")" != *arm64* ]]; then
        echo "Package member is not an Apple Silicon Mach-O: $binary" >&2
        exit 1
    fi
    build_info="$(vtool -show-build "$binary")"
    if [[ ! "$build_info" =~ platform[[:space:]]+MACOS || ! "$build_info" =~ minos[[:space:]]+14\.0([[:space:]]|$) ]]; then
        echo "Package member must target macOS 14.0: $binary" >&2
        exit 1
    fi
done

if [[ "$REQUIRE_SIGNATURE" == 1 ]]; then
    codesign --verify --strict --verbose=2 "$cli_binary" >/dev/null
    for bundle in "$file_manager_app" "$service_app" "$APP_PATH"; do
        codesign --verify --deep --strict --verbose=2 "$bundle" >/dev/null
    done
    if codesign -d --entitlements :- "$APP_PATH" 2>/dev/null | grep -q 'com.apple.security.app-sandbox'; then
        echo "App Sandbox is not supported for the fixed Configuration Set contract." >&2
        exit 1
    fi
fi

echo "Valid macOS 14+ arm64 Upit bundle: $APP_PATH"
