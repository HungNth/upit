# Upit Windows File Manager Upload package

The v0.6 integration targets **Windows 11 x64** only.

- `AppxManifest.xml.in` registers the package-identity `IExplorerCommand` in the primary File Explorer menu.
- `upit-explorer-command.dll` is an in-process COM adapter. It validates selection shape and launches `upit-file-manager.exe`; it performs no configuration or network work.
- `upit-file-manager.exe` owns the Wails-free File Manager Upload operation.
- `install.ps1` registers the sparse package against an external payload directory.
- `uninstall.ps1` removes package identity and Explorer registration.
- `package.ps1` builds the native DLL, validates the payload/logo/manifest, creates an unsigned or signed MSIX, and writes a SHA-256 checksum.
- `validate.ps1` is the unsigned verification gate used for ordinary CI changes.
- `native-smoke.ps1` is a protected, interactive Windows 11 x64 smoke. It runs executable helper/CLI checks, validates package registration/uninstall, starts a local endpoint, and records operator-observed Explorer lifecycle evidence. It is not a substitute for the self-hosted protected smoke runner.

## Local unsigned verification

`make build-desktop` builds `bin/upit-desktop.exe` and `bin/upit-file-manager.exe`. A Windows SDK installation providing `makeappx.exe` is required to create an MSIX:

```powershell
.\packaging\windows\validate.ps1 -Version 0.6.0.0 -Publisher 'CN=Upit Development'
.\packaging\windows\package.ps1 -Version 0.6.0.0 -Publisher 'CN=Upit Development'
```

The package is unsigned unless `-CertificatePath` is supplied. The script refuses signing unless `-ProtectedTag vX.Y.Z` is supplied and `UPIT_PROTECTED_RELEASE=true` is set.

## Protected release

The GitHub workflow uses three gates:

1. Hosted unsigned verification on pull requests and ordinary branch pushes.
2. A protected SemVer-tag job that signs, verifies with `signtool`, installs/uninstalls the package, and validates the external payload.
3. A required self-hosted runner labelled `Windows`, `X64`, and `upit-native-smoke`. The operator must run the primary-menu, selection, cancellation, recovery, no-window, helper-exit, and CLI continuity checks. The evidence JSON is uploaded before the publish job can run.

The signing environment must provide `UPIT_WINDOWS_CERTIFICATE_BASE64`, `UPIT_WINDOWS_CERTIFICATE_PASSWORD`, and `UPIT_MSIX_PUBLISHER`. The native smoke environment must provide an interactive Windows 11 x64 runner; a boolean secret alone is not accepted as smoke evidence.

Unsupported release routes: Windows 10 classic shell verbs, Windows ARM64, unsigned production packages, portable registration, and macOS Finder integration (planned for v0.7).
