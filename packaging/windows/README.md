# Upit for Windows

Upit targets **Windows 11 x64**. The consumer installs one signed `Upit` installer; Desktop, the Explorer adapter, CLI, and one-shot upload worker are private payload components, not separate applications.

- `installer.nsi` installs a complete versioned payload in the current user's application directory and registers its signed sparse MSIX without a consumer PowerShell step.
- The installer retains `repair\Upit.msix`. Its signed `payload-sha256.json` binds the external files to the package; Desktop verifies trusted signatures, matching signing certificates, hashes, version, and effective external location before offering Repair.
- A new payload is staged without overwriting the active payload. Registration must succeed before shortcuts/uninstall metadata switch; failed registration rolls back to the previous trusted installation or unregisters the failed initial installation.
- Uninstall embeds the same signed worker, so it can unregister the package even when the payload path registry value or installed worker is damaged. It unregisters before deleting product files; failed cleanup retains the payload and reports recovery guidance.
- The Explorer command validates exactly-one selection and delegates network/configuration work to the Wails-free worker.
- `native-smoke.ps1` executes the consumer installer on a clean protected Windows 11 runner, verifies clipboard delivery with the preference disabled, and records operator-observed Explorer, Repair, update, and recovery evidence. It is not proof until actually executed.

## Local unsigned verification

Run `make build` and `make build-desktop` to produce the CLI, Desktop, and private worker. Creating the installer requires the Windows SDK (`makeappx.exe`), Visual Studio/CMake for the adapter, and NSIS (`makensis.exe`):

```powershell
.\packaging\windows\validate.ps1 -Version 0.6.0.0 -Publisher 'CN=Upit Development'
.\packaging\windows\package.ps1 -Version 0.6.0.0 -Publisher 'CN=Upit Development'
```

The `.msix` is a retained internal registration artifact; `*-setup.exe` is the consumer artifact. Unsigned verification builds validate package shape only and cannot install production registration or advertise trusted Repair. Signing requires `-CertificatePath`, `-ProtectedTag vX.Y.Z`, and `UPIT_PROTECTED_RELEASE=true`.

## Consumer installation and removal

Run the signed `*-setup.exe`; no elevation or manual PowerShell registration is required. Open **Upit** from Start. In Desktop, **File Manager Integration** reports only verifiable package/payload state; **Repair Integration** is explicit and runs in the current-user context. Verify **Upload with Upit** yourself in File Explorer's primary menu. Remove **Upit** from Windows Installed apps; uninstall removes its package identity and Explorer registration before removing files.

## Protected release

The GitHub workflow uses three gates:

1. Hosted unsigned verification on pull requests and ordinary branch pushes.
2. A protected SemVer-tag job that signs the complete payload, registration artifact, and consumer installer, then verifies actual consumer install/uninstall.
3. A required self-hosted runner labelled `Windows`, `X64`, and `upit-native-smoke`. The operator verifies the primary menu, selection rejection, privacy, cancellation, clipboard recovery without re-upload, Desktop status/Repair, atomic update, Manual Upload, CLI continuity, and uninstall. Evidence is uploaded before publishing the one installer.

The signing environment must provide `UPIT_WINDOWS_CERTIFICATE_BASE64`, `UPIT_WINDOWS_CERTIFICATE_PASSWORD`, and `UPIT_MSIX_PUBLISHER`. The native smoke environment must provide an interactive Windows 11 x64 runner; a boolean secret alone is not accepted as smoke evidence.

Unsupported release routes: Windows 10 classic shell verbs, Windows ARM64, unsigned production packages, and portable registration. macOS uses its own one-product DMG and Finder Services route.
