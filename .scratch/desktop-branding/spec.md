# Spec: Desktop Branding and Cross-Platform Application Icons

Status: ready-for-agent

## Problem Statement

Upit Desktop currently lacks unified visual identity and native operating-system icons. On Windows, `upit-desktop.exe` is built without PE icon resources (defaulting to the generic executable icon), File Explorer context menu registration points to `upit-desktop.exe,0` which resolves to a blank/generic icon, the Start Menu shortcut lacks an explicit branded icon, and the NSIS installer/uninstaller binaries use generic default installer icons. On macOS, the `.app` bundle lacks a `CFBundleIconFile` specification and `AppIcon.icns` resource, appearing as a generic application in Finder and the Dock. Furthermore, the Desktop user interface has no visual branding in its header, and the webview shell lacks a favicon.

## Solution

Establish a unified branding asset pipeline and integrate native icon resources across all target platforms:
1. Maintain the canonical squircle master vector asset in `assets/branding/logo.svg` alongside reproducible, committed derivative binaries (`app.ico`, `AppIcon.icns`, `logo.png`).
2. Embed the Windows icon directly into `upit-desktop.exe` via a committed COFF resource (`.syso`), ensuring native icon appearance in File Explorer, the taskbar, and the Start Menu without external build-time dependencies.
3. Brand the Windows installer and uninstaller executables with the canonical `.ico` via NSIS modern UI directives.
4. Integrate the native macOS icon resource (`AppIcon.icns`) into the desktop application bundle layout and update the bundle metadata plist.
5. Enhance the Upit Desktop frontend with in-app branding in the top navigation header and configure the webview favicon.
6. Provide clear, reproducible instructions for regenerating derivative assets from the master SVG.

## User Stories

1. As a Windows user running Upit Desktop, I want to see the branded Upit icon on the taskbar and title bar, so that I can easily distinguish Upit from other open applications.
2. As a Windows user browsing files in File Explorer, I want to see the Upit icon next to the "Upload with Upit" context menu action, so that I have clear visual confirmation of the application identity.
3. As a Windows user opening the Start Menu, I want the Upit shortcut to show the official Upit icon, so that the application looks native and polished.
4. As a Windows user downloading and running the Upit installer or uninstaller, I want the installer window and setup binary to display the Upit logo, so that I can trust the software being installed.
5. As a macOS user, I want the Upit application bundle in Finder and the Dock to display the native high-resolution Upit icon, so that it integrates seamlessly into the macOS desktop experience.
6. As an interactive desktop user, I want to see the Upit logo in the application header next to the product name, so that the user interface feels branded and cohesive.
7. As a desktop user viewing the application shell, I want the window to load an official favicon, so that internal webview fallbacks do not show a generic browser page icon.
8. As a developer modifying the Upit brand asset, I want documented, deterministic commands to regenerate all derivative platform icons from `logo.svg`, so that platform assets remain reproducible without hidden dependencies.

## Implementation Decisions

### Centralized Branding Assets
- Move and consolidate canonical brand assets into `assets/branding/`.
- Retain `logo.svg` as the single authoritative source of truth (squircle shape with dark background `#09090b` and emerald upload motif `#34d399` / `#10b981`).
- Commit pre-generated derivative assets:
  - `app.ico`: Multi-resolution Windows icon containing layers `16x16`, `24x24`, `32x32`, `48x48`, `64x64`, `128x128`, and `256x256`.
  - `AppIcon.icns`: Apple Icon Image containing standard and retina resolutions up to `1024x1024`.
  - `logo.png`: High-resolution 1024x1024 PNG with alpha support.
- Include an `assets/branding/README.md` recording exact commands to reproduce each derivative from `logo.svg`.

### Windows Native Integration & Packaging
- Generate and commit a COFF resource file `cmd/upit-desktop/icon_windows_amd64.syso` embedding `app.ico` at resource ID 1 (index 0).
- When `go build` compiles `cmd/upit-desktop`, Go's PE linker automatically embeds the `.syso` into the resulting binary, fulfilling the Classic Verb contract (`upit-desktop.exe,0`) and Taskbar/Start Menu icon requirements.
- Configure `packaging/windows/installer.nsi` with `!define MUI_ICON "..\..\assets\branding\app.ico"` and `!define MUI_UNICON "..\..\assets\branding\app.ico"`.

### macOS Native Integration & Packaging
- Configure `packaging/macos/desktop-Info.plist.in` to declare `<key>CFBundleIconFile</key><string>AppIcon</string>`.
- Update `packaging/macos/package.sh` to copy `assets/branding/AppIcon.icns` into `Contents/Resources/AppIcon.icns` of the desktop application bundle during packaging.

### Desktop Frontend UI Branding
- Copy or import the SVG logo into the frontend application assets.
- Update `cmd/upit-desktop/frontend/src/App.vue` header to render the 24x24 px logo adjacent to the "Upit" title.
- Update `cmd/upit-desktop/frontend/index.html` to configure the `<link rel="icon">` referencing the logo SVG.

## Testing Decisions

A good test validates observable artifacts and contracts rather than internal generator state.
- **PE Resource Validation**: Verify via automated test or packaging validation script that `upit-desktop.exe` contains a valid RT_GROUP_ICON resource entry (index 0).
- **Installer Validation (`validate.ps1`)**: Verify that the generated Windows setup executable retains the custom icon resource and that NSIS completes without packaging errors.
- **macOS Bundle Validation (`validate.sh`)**: Assert the presence and non-zero size of `Contents/Resources/AppIcon.icns` and the presence of `<key>CFBundleIconFile</key><string>AppIcon</string>` in `Contents/Info.plist`.
- **Frontend Static & Build Seam (`npm run build`)**: Assert that `npm run build` succeeds, that built `dist/index.html` references the branded favicon, and that the Desktop header template references the logo asset.

## Out of Scope

- Dynamic runtime theme switching for the logo icon (e.g. automatic light/dark variant swapping in the OS dock/taskbar).
- System tray / menu bar resident icons (Upit architecture explicitly avoids resident background daemons).
- Replacing CLI binary icons (console executables on Windows/macOS remain headless).

## Further Notes

- The committed `.syso` pattern eliminates any requirement for `windres` or external C compilers in the standard Go build environment or CI pipeline.
- If the master `logo.svg` changes in the future, developers can re-run the documented asset generation steps in `assets/branding/README.md`.
