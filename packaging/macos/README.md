# macOS File Manager Upload

The v0.7 macOS integration targets **macOS 14 or newer on Apple Silicon**. It is distributed outside the Mac App Store as a Developer ID-signed, Hardened Runtime, notarized, and stapled disk image.

## Bundle layout

`Upit.app` contains:

- `Contents/MacOS/upit-desktop`: the existing Wails Desktop application;
- `Contents/Helpers/UpitFinderService.app`: the background-only `NSServices` provider;
- `Contents/Helpers/UpitFinderService.app/Contents/Helpers/UpitFileManager.app`: the Wails-free one-shot helper.

The Finder provider accepts exactly one local regular file URL and passes the untrusted URL to the shared Go helper as `--file-url`. It does not load configuration, upload bytes, parse CLI output, render private values, or remain resident. The Go helper converts and validates the URL, then invokes the existing File Manager Upload service.

The package is intentionally **not App Sandbox**. Finder, Desktop, and CLI must read the same fixed `~/.config/upit/` Configuration Set. Finder Sync, Share Extensions, Action Extensions, Automator workflows, and Mac App Store distribution are not used.

## Local unsigned package

On a macOS 14+ Apple Silicon host with Go, Node.js, npm, Xcode Command Line Tools, and `clang`:

```bash
make setup-desktop
make package-macos MACOS_VERSION=0.7.0
```

The output is an unsigned verification DMG under `dist/macos/`. The package script validates the app layout and Finder Service metadata before creating the DMG.

## Installation and removal

Mount the DMG, then register the app and Service explicitly:

```bash
packaging/macos/install.sh --app /Volumes/Upit\ 0.7.0/Upit.app
packaging/macos/uninstall.sh
```

Launch Services registration is automatic through `lsregister`. macOS user preferences can still require enabling **System Settings → Keyboard → Keyboard Shortcuts → Services → Files and Folders**. The installer prints that path.

## Protected release

Only the protected SemVer-tag workflow may provide signing and notarization credentials:

```bash
UPIT_PROTECTED_RELEASE=true make package-macos \
  MACOS_VERSION=0.7.0 \
  MACOS_PROTECTED_TAG=v0.7.0 \
  MACOS_SIGNING_IDENTITY='Developer ID Application: Example (TEAMID)' \
  MACOS_NOTARY_PROFILE=upit-release
```

The workflow signs the desktop binary, Finder Service executable and bundle, nested helper executable and bundle, and outer app with Hardened Runtime. It verifies signatures, submits the distributable DMG to Apple notarization, staples and validates the DMG ticket, writes SHA-256 checksums and metadata, runs the protected native smoke, and publishes only after smoke evidence is present.

Ordinary pull requests and branch pushes build unsigned artifacts and never receive release secrets.

## Native smoke

`native-smoke.sh` requires a real macOS 14+ Apple Silicon host. It installs the bundle into a supplied directory, prepares a temporary Configuration Set and local HTTP endpoint, opens a Finder fixture, prompts for the native Finder lifecycle assertions, runs the unchanged CLI regression when supplied, writes JSON evidence, and unregisters/removes the Service during cleanup.

The evidence must cover Service discovery, selection rejection, success, warning, failure, cancellation, privacy, recovery actions, no Desktop/Dock behavior, helper exit, CLI continuity, and uninstall cleanup. A compile or package check is not a substitute for this smoke.
