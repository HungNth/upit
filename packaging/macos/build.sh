#!/usr/bin/env bash
set -euo pipefail

SOURCE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_DIRECTORY="$SOURCE_ROOT/dist/macos"
VERSION=""
BUILD_VERSION=""
SIGNING_IDENTITY="${MACOS_SIGNING_IDENTITY:-}"
NOTARY_PROFILE="${MACOS_NOTARY_PROFILE:-}"
PROTECTED_TAG="${MACOS_PROTECTED_TAG:-}"

usage() {
    cat <<'EOF'
Usage: packaging/macos/build.sh [options]

Options:
  --version X.Y.Z          Product semantic version (default: 0.0.0)
  --build-version VALUE    CFBundleVersion (default: semantic version)
  --source-root PATH       Repository root (default: detected repository root)
  --output-directory PATH  Artifact directory (default: dist/macos)
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
if [[ -z "$VERSION" ]]; then
    if [[ -f "$SOURCE_ROOT/VERSION" ]]; then
        VERSION="$(tr -d '[:space:]' < "$SOURCE_ROOT/VERSION")"
    else
        VERSION="0.9.0"
    fi
fi

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "Version must follow semantic versioning: X.Y.Z" >&2
    exit 2
fi

if [[ -z "$BUILD_VERSION" ]]; then
    BUILD_VERSION="$VERSION"
fi

if [[ ! "$BUILD_VERSION" =~ ^[0-9]+(\.[0-9]+){0,3}$ ]]; then
    echo "Build version must contain 1 to 4 numeric components." >&2
    exit 2
fi

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
    echo "macOS packaging requires an Apple Silicon macOS host." >&2
    exit 1
fi

frontend_dir="$SOURCE_ROOT/cmd/upit-desktop/frontend"
if [[ ! -d "$frontend_dir" ]]; then
    echo "Frontend directory not found: $frontend_dir" >&2
    exit 1
fi

service_source="$SOURCE_ROOT/native/macos/finder-service/main.m"
selection_source="$SOURCE_ROOT/native/macos/finder-service/selection.m"
if [[ ! -f "$service_source" || ! -f "$selection_source" ]]; then
    echo "Finder service sources missing under native/macos/finder-service" >&2
    exit 1
fi

staging_root="$SOURCE_ROOT/.build/package"
mkdir -p "$staging_root"
work_directory="$(mktemp -d "$staging_root/run.XXXXXX")"
trap 'rm -rf "$work_directory"' EXIT

# Step 1: Install frontend dependencies from lockfile and build assets
npm --prefix "$frontend_dir" ci
npm --prefix "$frontend_dir" run build

# Step 2: Compile private staged Go binaries
export GOOS=darwin
export GOARCH=arm64
export CGO_ENABLED=1
export MACOSX_DEPLOYMENT_TARGET=14.0
export CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=14.0"
export CGO_CXXFLAGS="${CGO_CXXFLAGS:-} -mmacosx-version-min=14.0"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=14.0"
export GOFLAGS="${GOFLAGS:-} -ldflags=-extldflags=-mmacosx-version-min=14.0"

staged_bin="$work_directory/bin"
mkdir -p "$staged_bin"
staged_cli="$staged_bin/upit"
staged_desktop="$staged_bin/upit-desktop"
staged_file_manager="$staged_bin/upit-file-manager"

version_ldflags="-X github.com/HungNth/upit/internal/version.version=$VERSION"
go build -ldflags "$version_ldflags" -o "$staged_cli" "$SOURCE_ROOT/cmd/upit"
go build -ldflags "$version_ldflags" -o "$staged_file_manager" "$SOURCE_ROOT/cmd/upit-file-manager"
go build -ldflags "$version_ldflags" -o "$staged_desktop" "$SOURCE_ROOT/cmd/upit-desktop"

# Step 3: Compile private staged Finder service adapter
staged_adapter="$staged_bin/UpitFinderService"
clang -fobjc-arc -mmacosx-version-min=14.0 -framework Cocoa \
    "$service_source" \
    "$selection_source" \
    -o "$staged_adapter"
chmod +x "$staged_cli" "$staged_desktop" "$staged_file_manager" "$staged_adapter"

# Step 4: Assemble package
package_args=(
    --source-root "$SOURCE_ROOT"
    --output-directory "$OUTPUT_DIRECTORY"
    --work-directory "$work_directory/package"
    --version "$VERSION"
    --build-version "$BUILD_VERSION"
    --cli-binary "$staged_cli"
    --desktop-binary "$staged_desktop"
    --helper-binary "$staged_file_manager"
    --adapter-binary "$staged_adapter"
)

if [[ -n "$SIGNING_IDENTITY" ]]; then
    package_args+=(--signing-identity "$SIGNING_IDENTITY")
fi
if [[ -n "$NOTARY_PROFILE" ]]; then
    package_args+=(--notary-profile "$NOTARY_PROFILE")
fi
if [[ -n "$PROTECTED_TAG" ]]; then
    package_args+=(--protected-tag "$PROTECTED_TAG")
fi

bash "$SOURCE_ROOT/packaging/macos/package.sh" "${package_args[@]}"
