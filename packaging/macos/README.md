# Upit for macOS

Upit targets **macOS 14 or newer on Apple Silicon**. One application contains Desktop and private File Manager Integration components. Production distribution requires a Developer ID-signed, Hardened Runtime, notarized, and stapled disk image.

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

1. Mount the DMG and drag `Upit.app` onto the Applications shortcut. No shell script is required.
2. Open Upit. File Manager Integration performs a non-mutating inspection; use **Repair Integration** if registration is absent or stale.
3. Verify **Upload with Upit** from Finder Services or Quick Actions. If necessary, enable it in **System Settings → Keyboard → Keyboard Shortcuts → Services → Files and Folders**. Desktop cannot observe that private preference or promise live menu visibility.
4. Before removing the application, choose **Prepare to Remove Upit** in Desktop and confirm. After the Service and outer app are unregistered and Desktop closes, move `Upit.app` to Trash. Your Configuration Set remains intact.

`install.sh` and `uninstall.sh` are internal native-smoke helpers, not the consumer installation/removal flow.

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

To run:

```bash
bash packaging/macos/native-smoke.sh \
  --app dist/macos/Upit.app \
  --install-directory /Applications \
  --cli bin/upit-darwin-arm64 \
  --evidence native-smoke-evidence.json \
  [--require-signature]
```

### What can and cannot be proven

- **Can be proven locally**: Real Finder Services/Quick Actions discovery, single-selection validation, progress feedback closing, silent native notification delivery via `UNUserNotificationCenter` with exact title (`Upload complete`) and body (`Final URL copied to clipboard.`), operating-system dismissal, absence of Upit action on selection, two sequential clean successes generating two distinct events without aggregation/replacement, absence of clean-success `NSAlert` modal fallback, retention of existing interactive native feedback (banner action buttons or alert fallback) for warnings/failures/cancellations/recovery actions, helper exit, absence of Desktop window/Dock icon/tray/resident worker during upload, and CLI/Manual upload continuity against the same Configuration Set.
- **Cannot be proven by local or unsigned smoke**: Unsigned or development evidence does **not** prove publisher trust, Developer ID signatures, Apple notarization, stapled tickets, Gatekeeper acceptance on clean machines, or production readiness.
- **Authoritative production proof**: Only the protected signed/notarized release workflow (`UPIT_PROTECTED_RELEASE=true` on a tagged release with `--require-signature`) running on the dedicated `upit-native-smoke` runner provides authoritative production release proof. The focused clean-success smoke evidence augments rather than replaces the umbrella signed/notarized release gates.
