# Upit for Windows

Upit targets **Windows 11 x64**. The consumer installs one signed `Upit` installer; Desktop, the Explorer adapter, CLI, and one-shot upload worker are private payload components, not separate applications.

- `installer.nsi` installs a complete versioned payload in the current user's application directory and registers its signed sparse MSIX without a consumer PowerShell step.
- The installer retains `repair\Upit.msix`. Its signed `payload-sha256.json` binds the external files to the package; Desktop verifies trusted signatures, matching signing certificates, hashes, version, and effective external location before offering Repair.
- A new payload is staged without overwriting the active payload. Registration must succeed before shortcuts/uninstall metadata switch; failed registration rolls back to the previous trusted installation or unregisters the failed initial installation.
- Uninstall embeds the same signed worker, so it can unregister the package even when the payload path registry value or installed worker is damaged. It unregisters before deleting product files; failed cleanup retains the payload and reports recovery guidance.
- The Explorer command validates exactly-one selection and delegates network/configuration work to the Wails-free worker.
- `native-smoke.ps1` executes the consumer installer on a clean protected Windows 11 runner, verifies clipboard delivery with the preference disabled, and records operator-observed Explorer, Repair, update, and recovery evidence. It is not proof until actually executed.

## Local packaging and verification

Creating the installer requires Go, Node.js, npm, the Windows SDK (`makeappx.exe`), Visual Studio/CMake for the adapter, and NSIS (`makensis.exe`).

Run the unified package command:

```cmd
make package
```

By default `make package` creates an unsigned verification installer at `dist\windows\upit-windows-x64-0.0.0-setup.exe` with `0.0.0.0` technical package identity, checksum, and metadata sidecars:

```cmd
make package VERSION=0.0.0
```

The `.msix` is retained only inside the setup installer as private registration/repair material; `*-setup.exe` is the single consumer deliverable. Unsigned verification builds validate package shape only and cannot install production registration or advertise trusted Repair.

All private binaries, CMake output, and installer payloads use per-run `.build/package/` staging, removed on success and failure. `bin/` remains the standalone CLI boundary. `make clean` removes `bin/`, `.build/`, and repository-root `dist/`, including interrupted package runs.

For protected builds, set `WINDOWS_PUBLISHER`, `WINDOWS_CERTIFICATE_PATH`, `WINDOWS_CERTIFICATE_PASSWORD`, `WINDOWS_PROTECTED_TAG=vX.Y.Z`, and `UPIT_PROTECTED_RELEASE=true` in the environment, then run `make package VERSION=X.Y.Z`. `WINDOWS_TIMESTAMP_SERVER` optionally overrides the existing timestamp service. Platform scripts are implementation details behind this command.

## Consumer installation and removal

Run the signed `*-setup.exe`; no elevation or manual PowerShell registration is required. Open **Upit** from Start. In Desktop, **File Manager Integration** reports only verifiable package/payload state; **Repair Integration** is explicit and runs in the current-user context. Verify **Upload with Upit** yourself in File Explorer's primary menu. Remove **Upit** from Windows Installed apps; uninstall removes its package identity and Explorer registration before removing files.

## Protected release

The GitHub workflow uses three gates:

1. Hosted unsigned verification on pull requests and ordinary branch pushes using `bash packaging/command-smoke.sh` and `make package`.
2. A protected SemVer-tag job that signs the complete payload, registration artifact, and consumer installer using `make package VERSION=$version` with environment signing inputs, then verifies actual consumer install/uninstall.
3. A required self-hosted runner labelled `Windows`, `X64`, and `upit-native-smoke`. The operator verifies the primary menu, selection rejection, privacy, cancellation, clipboard recovery without re-upload, Desktop status/Repair, atomic update, Manual Upload, CLI continuity, and uninstall. Evidence is uploaded before publishing the one installer.

The signing environment must provide `UPIT_WINDOWS_CERTIFICATE_BASE64`, `UPIT_WINDOWS_CERTIFICATE_PASSWORD`, and `UPIT_MSIX_PUBLISHER`. The native smoke environment must provide an interactive Windows 11 x64 runner; a boolean secret alone is not accepted as smoke evidence.

## Native smoke

`native-smoke.ps1` runs on a Windows 11 x64 runner. It installs the consumer installer, validates automated headless helper and CLI behavior, restores any inherited headless environment flags, opens a File Explorer fixture, prompts the operator through interactive clean-success and failure paths, writes structured JSON evidence, and unregisters/uninstalls the package on exit.

To run:

```powershell
.\packaging\windows\native-smoke.ps1 `
  -InstallerPath dist\windows\upit-windows-x64-0.7.0-setup.exe `
  -InstallDirectory "$env:LOCALAPPDATA\Programs\Upit" `
  [-EvidencePath native-smoke-evidence.json]
```

### What can and cannot be proven

- **Can be proven on the local test runner**: Primary Windows 11 File Explorer context-menu integration, folder and multi-selection suppression, progress Task Dialog closing, silent Windows Toast delivery via package identity with exact title (`Upload complete`) and body (`Final URL copied to clipboard.`), operating-system dismissal into Notification Center, absence of Upit action on Toast click, two sequential clean successes generating two distinct events without aggregation/replacement, absence of clean-success Task Dialog/Message Box modal fallback, retention of interactive modal Task Dialogs/alerts for warnings/failures/cancellations/recovery actions, helper process exit immediately upon completion, absence of Desktop window/tray icon/resident worker during direct upload, Desktop status and Repair integration, atomic update preserving package identity, Manual Upload and CLI continuity against the same Configuration Set, and clean uninstall unregistering package identity and removing payload files.
- **Cannot be proven by supporting smoke alone**: Executing the native smoke script on an individual runner is supporting smoke evidence only; it does **not** independently prove publisher trust, SmartScreen reputation across consumer fleets, or complete production readiness.
- **Authoritative production proof**: Only the protected signed release workflow (`UPIT_PROTECTED_RELEASE=true` on a tagged release with valid Authenticode signature) running on the dedicated `upit-native-smoke` runner provides authoritative production release proof. The focused clean-success smoke evidence augments rather than replaces the umbrella signed-release gates.

Unsupported release routes: Windows 10 classic shell verbs, Windows ARM64, unsigned production packages, and portable registration. macOS uses its own one-product DMG and Finder Services route.
