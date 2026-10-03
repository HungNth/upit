#!/usr/bin/env bash
set -euo pipefail

SOURCE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_DIRECTORY="$SOURCE_ROOT/dist/macos"
VERSION=""
BUILD_VERSION=""
SIGNING_IDENTITY=""
NOTARY_PROFILE=""
PROTECTED_TAG=""

usage() {
    cat <<'EOF'
Usage: packaging/macos/package.sh --version X.Y.Z [options]

Options:
  --source-root PATH       Repository root (default: detected repository root)
  --output-directory PATH  Artifact directory (default: dist/macos)
  --build-version VALUE    CFBundleVersion (default: semantic version)
  --signing-identity NAME  Developer ID Application identity for protected releases
  --notary-profile NAME    xcrun notarytool keychain profile for protected releases
  --protected-tag TAG      Protected SemVer tag, for example v0.7.0
EOF
}

while (($# > 0)); do
    case "$1" in
        --source-root) SOURCE_ROOT="$2"; shift 2 ;;
        --output-directory) OUTPUT_DIRECTORY="$2"; shift 2 ;;
        --version) VERSION="$2"; shift 2 ;;
        --build-version) BUILD_VERSION="$2"; shift 2 ;;
        --signing-identity) SIGNING_IDENTITY="$2"; shift 2 ;;
        --notary-profile) NOTARY_PROFILE="$2"; shift 2 ;;
        --protected-tag) PROTECTED_TAG="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "--version must be a semantic version X.Y.Z." >&2
    exit 2
fi
if [[ -z "$BUILD_VERSION" ]]; then
    BUILD_VERSION="$VERSION"
fi
if [[ ! "$BUILD_VERSION" =~ ^[0-9]+(\.[0-9]+){0,3}$ ]]; then
    echo "--build-version must contain only dot-separated numeric components." >&2
    exit 2
fi
if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
    echo "macOS File Manager Upload packages require a macOS arm64 host." >&2
    exit 1
fi

DESKTOP_BINARY="$SOURCE_ROOT/bin/upit-desktop"
FILE_MANAGER_BINARY="$SOURCE_ROOT/bin/upit-file-manager"
SERVICE_SOURCE="$SOURCE_ROOT/native/macos/finder-service/main.m"
SELECTION_SOURCE="$SOURCE_ROOT/native/macos/finder-service/selection.m"
ENTITLEMENTS="$SOURCE_ROOT/packaging/macos/entitlements.plist"
for input in "$DESKTOP_BINARY" "$FILE_MANAGER_BINARY" "$SERVICE_SOURCE" "$SELECTION_SOURCE" "$ENTITLEMENTS"; do
    if [[ ! -f "$input" ]]; then
        echo "Required package input is missing: $input" >&2
        exit 1
    fi
done

if [[ -n "$SIGNING_IDENTITY" || -n "$NOTARY_PROFILE" || -n "$PROTECTED_TAG" ]]; then
    if [[ "$PROTECTED_TAG" != v[0-9]*.[0-9]*.[0-9]* ]]; then
        echo "Signing and notarization require --protected-tag vX.Y.Z." >&2
        exit 1
    fi
    if [[ ! "$PROTECTED_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Unsupported protected tag: $PROTECTED_TAG" >&2
        exit 1
    fi
    if [[ "${UPIT_PROTECTED_RELEASE:-}" != "true" ]]; then
        echo "UPIT_PROTECTED_RELEASE=true is required for signing or notarization." >&2
        exit 1
    fi
    if [[ -z "$SIGNING_IDENTITY" || -z "$NOTARY_PROFILE" ]]; then
        echo "Protected packages require both --signing-identity and --notary-profile." >&2
        exit 1
    fi
fi

mkdir -p "$OUTPUT_DIRECTORY"
work_directory="$(mktemp -d "${TMPDIR:-/tmp}/upit-macos-package.XXXXXX")"
trap 'rm -rf "$work_directory"' EXIT

render_plist() {
    local template="$1"
    local destination="$2"
    local content
    content="$(<"$template")"
    content="${content//@VERSION@/$VERSION}"
    content="${content//@BUILD_VERSION@/$BUILD_VERSION}"
    printf '%s\n' "$content" > "$destination"
}

app="$work_directory/Upit.app"
service_app="$app/Contents/Helpers/UpitFinderService.app"
file_manager_app="$service_app/Contents/Helpers/UpitFileManager.app"
mkdir -p \
    "$app/Contents/MacOS" \
    "$app/Contents/Resources" \
    "$service_app/Contents/MacOS" \
    "$service_app/Contents/Helpers" \
    "$file_manager_app/Contents/MacOS"

cp "$DESKTOP_BINARY" "$app/Contents/MacOS/upit-desktop"
cp "$FILE_MANAGER_BINARY" "$file_manager_app/Contents/MacOS/upit-file-manager"
render_plist "$SOURCE_ROOT/packaging/macos/desktop-Info.plist.in" "$app/Contents/Info.plist"
render_plist "$SOURCE_ROOT/native/macos/finder-service/Info.plist.in" "$service_app/Contents/Info.plist"
render_plist "$SOURCE_ROOT/packaging/macos/file-manager-Info.plist.in" "$file_manager_app/Contents/Info.plist"

clang -fobjc-arc -mmacosx-version-min=14.0 -framework Cocoa \
    "$SERVICE_SOURCE" \
    "$SELECTION_SOURCE" \
    -o "$service_app/Contents/MacOS/UpitFinderService"
chmod +x "$app/Contents/MacOS/upit-desktop" \
    "$service_app/Contents/MacOS/UpitFinderService" \
    "$file_manager_app/Contents/MacOS/upit-file-manager"

sign_binary() {
    local path="$1"
    codesign --force --options runtime --timestamp --sign "$SIGNING_IDENTITY" \
        --entitlements "$ENTITLEMENTS" "$path"
}

if [[ -n "$SIGNING_IDENTITY" ]]; then
    sign_binary "$file_manager_app/Contents/MacOS/upit-file-manager"
    sign_binary "$service_app/Contents/MacOS/UpitFinderService"
    sign_binary "$file_manager_app"
    sign_binary "$service_app"
    sign_binary "$app/Contents/MacOS/upit-desktop"
    sign_binary "$app"
    codesign --verify --deep --strict --verbose=2 "$app"
fi
validate_args=(--app "$app")
if [[ -n "$SIGNING_IDENTITY" ]]; then
    validate_args+=(--require-signature)
fi
bash "$SOURCE_ROOT/packaging/macos/validate.sh" "${validate_args[@]}"

image_root="$work_directory/dmg-root"
mkdir -p "$image_root"
cp -R "$app" "$image_root/Upit.app"
ln -s /Applications "$image_root/Applications"
artifact="$OUTPUT_DIRECTORY/upit-macos-arm64-$VERSION.dmg"
hdiutil create -volname "Upit $VERSION" -srcfolder "$image_root" -ov -format UDZO "$artifact" >/dev/null

if [[ -n "$NOTARY_PROFILE" ]]; then
    xcrun notarytool submit "$artifact" --keychain-profile "$NOTARY_PROFILE" --wait
    xcrun stapler staple "$artifact"
    xcrun stapler validate "$artifact"
fi

shasum -a 256 "$artifact" > "$artifact.sha256"
cat > "$artifact.metadata.json" <<EOF
{
  "name": "upit",
  "version": "$VERSION",
  "buildVersion": "$BUILD_VERSION",
  "platform": "macos",
  "architecture": "arm64",
  "minimumSystemVersion": "14.0",
  "signed": $([[ -n "$SIGNING_IDENTITY" ]] && echo true || echo false),
  "notarized": $([[ -n "$NOTARY_PROFILE" ]] && echo true || echo false),
  "serviceBundle": "Upit.app/Contents/Helpers/UpitFinderService.app",
  "serviceName": "Upload with Upit"
}
EOF

echo "Built $artifact"
