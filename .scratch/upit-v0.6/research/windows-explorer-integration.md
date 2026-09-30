# Primary-Source Research: Windows File Explorer Direct-Upload Integration

## Executive Summary & Design Decision Default

For `upit` v0.6 Windows integration, the core requirement is: **upload a selected file directly using the default uploader and default shortener without opening the `upit-desktop` GUI**. Manual upload via UI remains exclusively for users who intentionally launch the desktop application.

### Key Finding on Windows 11 Primary Menu vs "Show more options"
- **Windows 11 Primary (Modern) Menu**: Windows 11 completely restructured Explorer's context menu. It **strictly ignores** traditional registry shell verbs (`HKEY_CLASSES_ROOT\*\shell\...`) in the top-level menu. To appear in the modern top-level menu, an app **must** implement the COM interface `IExplorerCommand` packaged in a native DLL and declare the extension in an MSIX package manifest (`windows.fileExplorerContextMenus`). For unpackaged apps (such as standard NSIS or portable binaries), Windows requires a **Sparse Package** (packaging with external location) signed with a trusted certificate.
- **"Show more options" (Windows 10 Classic Shell Menu)**: Standard Win32 static registry verbs registered under `HKCU\Software\Classes\*\shell\<verb>` appear directly on Windows 10, but on Windows 11 they are relegated behind the secondary "Show more options" menu (or accessed via `Shift+F10`).
- **Target Executable**: Invoking a dedicated Wails-free helper directly from File Explorer avoids Wails GUI overhead. A legacy CLI alternative would use the current operand syntax `upit.exe upload "<path>"`, not an invented `--file` flag.

### Evidence-Based Default Recommendation
- **Phase 1 / Minimum Viable Route (MVR)**: Register a per-user static shell verb under `HKCU\Software\Classes\*\shell\upit.upload` with command `"C:\Path\To\upit.exe" upload "%1"`.
  - **Pros**: Zero C++/COM compilation dependencies, zero packaging tools, zero certificate trust requirements, and simple per-user installer registration.
  - **Windows 10**: First-class primary context menu.
  - **Windows 11**: Appears under "Show more options" (classic menu).
- **Phase 2 / Modern First-Class Route (Windows 11 Primary Menu)**: Build a lightweight native C++ `IExplorerCommand` companion DLL (e.g. `upit-shell-extension.dll`) and package an MSIX sparse package manifest signed with a code-signing certificate, registered via `Add-AppxPackage -ExternalLocation`.
  - **Pros**: Direct presence on Windows 11 top-level modern menu alongside Cut/Copy/Paste.
  - **Cons**: Requires C++ toolchain in CI, strict architecture matching (`x64`, `arm64`), trusted signing certificate (untrusted roots fail with `CERT_E_UNTRUSTEDROOT` 0x800B0109), and PowerShell / `PackageManager` registration during installation.

---

## 1. Windows Context Menu Extensibility Models

### 1.1 The Windows 11 Modern Context Menu Architecture
Source: [Microsoft Learn: Add a File Explorer context menu command to a packaged desktop app](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/integrate-packaged-app-with-file-explorer), [Windows Developer Blog: Extending the Context Menu and Share Dialog in Windows 11](https://blogs.windows.com/windowsdeveloper/2021/07/19/extending-the-context-menu-and-share-dialog-in-windows-11/).

Beginning with Windows 11 (Build 22000), Microsoft redesigned File Explorer context menus to reduce menu bloat, eliminate shell hangs caused by in-process legacy shell extensions, and provide predictable command grouping:
- **Top Level Placement**: Only inbox commands (Cut, Copy, Paste, Delete, Rename), Shell verbs, cloud sync verbs, and apps registered via the modern `windows.fileExplorerContextMenus` extension appear in the primary menu.
- **Mandatory Requirements for Primary Menu**:
  1. **Interface Contract**: The extension **must** implement `IExplorerCommand` (inheriting from `IUnknown`). The legacy `IContextMenu` interface is explicitly excluded from the Windows 11 top-level menu.
  2. **Package Identity**: Explorer will **only** enumerate commands registered through an Appx / MSIX manifest extension:
     ```xml
     <desktop4:Extension Category="windows.fileExplorerContextMenus">
       <desktop4:FileExplorerContextMenus>
         <desktop5:ItemType Type="*">
           <desktop5:Verb Id="UploadWithUpit" Clsid="<CLSID-GUID>" />
         </desktop5:ItemType>
       </desktop4:FileExplorerContextMenus>
     </desktop4:Extension>
     ```
  3. **In-Process COM DLL**: The CLSID must be backed by a native DLL registered via `com:SurrogateServer` in the package manifest. Explorer loads this DLL directly into its process address space to call `GetTitle()`, `GetIcon()`, `GetState()`, and `Invoke()`.
- **"Show more options" Fallback**:
  Any verb registered via classic registry keys (`HKEY_CLASSES_ROOT\*\shell\...`) or legacy in-process COM shell handlers (`IContextMenu` registered under `shellex\ContextMenuHandlers`) is relegated to the "Show more options" sub-tier (or shown when pressing `Shift+F10` / Menu key).

### 1.2 The Classic Win32 Static Shell Verb Model
Source: [Microsoft Learn: Creating Shortcut Menu Handlers](https://learn.microsoft.com/en-us/windows/win32/shell/context-menu-handlers), [Microsoft Learn: Verbs and File Associations](https://learn.microsoft.com/en-us/windows/win32/shell/fa-verbs).

Static verbs are registered directly in the Windows Registry without requiring custom DLLs or COM programming:
- **Registry Hierarchy**:
  Under file associations or the catch-all wildcard `*` (any file):
  - Per-user (no admin privileges required): `HKEY_CURRENT_USER\Software\Classes\*\shell\<VerbName>`
  - Machine-wide (requires elevation / UAC): `HKEY_LOCAL_MACHINE\Software\Classes\*\shell\<VerbName>`
  - Combined system view: `HKEY_CLASSES_ROOT\*\shell\<VerbName>`
- **Subkeys & Values**:
  - `(Default)`: Display name shown on the menu (e.g., `Upload with Upit`), or `MUIVerb` for MUI string indirection.
  - `Icon`: Absolute path to executable or icon file, e.g., `"C:\Users\User\AppData\Local\Programs\upit\upit.exe",0`.
  - `command`: Subkey whose `(Default)` string contains the command line executed by the Shell:
    ```
    "C:\Users\User\AppData\Local\Programs\upit\upit.exe" upload "%1"
    ```

---

## 2. Parameter Substitution, Quoting, and Multi-Selection

### 2.1 File Path Quoting and Escaping
Source: [Microsoft Learn: Creating Shortcut Menu Handlers](https://learn.microsoft.com/en-us/windows/win32/shell/context-menu-handlers).

When File Explorer executes a command string registered under a static verb:
- `%1` expands to the full, absolute, unquoted path of the selected item (e.g., `C:\Users\John Doe\Documents\My Report.pdf`).
- **Mandatory Quoting**: The registry value **must** wrap `%1` in literal quotes (`"%1"`). If quotes are omitted, paths containing spaces will be split across multiple argv tokens by the Windows command line parsing routines (`CommandLineToArgvW`), leading to argument fragmentation or security vulnerabilities.
- **Format**:
  ```text
  "C:\Path\To\upit.exe" upload "%1"
  ```
- Long paths (> 260 characters / `MAX_PATH`): On Windows 10/11 systems with long path awareness enabled in the registry (`LongPathsEnabled` under `HKLM\SYSTEM\CurrentControlSet\Control\FileSystem`) and declared in the application fusion manifest (`<longPathAware>true</longPathAware>`), Explorer supplies extended absolute paths to the invoked process.

### 2.2 Multi-Selection Behavior: `MultiSelectModel`
Source: [Microsoft Learn: Employing the Verb Selection Model](https://learn.microsoft.com/en-us/windows/win32/shell/context-menu-handlers#employing-the-verb-selection-model).

When a user selects multiple files in File Explorer and right-clicks:
- The registry entry can include a named value `MultiSelectModel` (**REG_SZ**) under `...\shell\<VerbName>`:
  - `Single`: The verb **only appears** if exactly 1 item is selected. If 2 or more files are selected, File Explorer suppresses the verb entirely.
  - `Document`: File Explorer invokes the target executable **once per selected file**. For legacy static verbs, selecting 5 files will launch **5 separate processes** simultaneously. Explorer enforces an internal ceiling of 15 items by default for `Document` model verbs to prevent system resource exhaustion.
  - `Player`: Designed for media players and batch tools. Allows up to 100 items for legacy static verbs. However, for command-line static verbs (`command` subkey), Explorer still spawns one process per item unless activated via COM `IDropTarget`.
- **COM / `IExplorerCommand` Behavior**:
  In contrast to static verbs, `IExplorerCommand::Invoke(IShellItemArray* psiItemArray, IBindCtx* pbc)` receives an `IShellItemArray` representing all selected items in a **single call**. The implementation can inspect `psiItemArray->GetCount(&count)` and either iterate through items or reject invocation if count exceeds 1.

### 2.3 Single-File vs Multi-Select Design Choice for `upit`
- **Initial Target (v0.6)**: The assignment specifies: *"Cover one selected regular file first; identify multi-select behavior as a design choice, not an assumption."*
- **Design Options**:
  1. **Strict Single-Select (`MultiSelectModel = "Single"`)**:
     - Menu verb only visible when exactly 1 regular file is selected.
     - Multi-select right-click cleanly hides the verb.
     - **Recommendation for v0.6**: Enforce `MultiSelectModel = "Single"` in static registry registration.
  2. **Multi-Select Spawning (Default / `Document`)**:
     - Selecting 10 files spawns 10 concurrent `upit` processes.
     - *Risks*: 10 parallel network uploads saturating bandwidth, rate-limiting on target host (e.g. Imgur, S3), 10 concurrent clipboard overwrite race conditions (the last one to finish wins), and 10 separate toast notifications spamming the desktop.
  3. **Batch Handling via CLI / Single Process**:
     - Requires either an `IExplorerCommand` native DLL that hands all selected paths to a future batch-capable helper, or an inter-process single-instance lock / named pipe daemon that batches uploads sequentially.

---

## 3. Package Identity and Modern Sparse Packages

### 3.1 What is a Sparse Package?
Source: [Microsoft Learn: Grant package identity by packaging with external location manually](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/grant-identity-to-nonpackaged-apps).

A **Sparse Package** (officially termed *"packaging with external location"*) is an MSIX package containing only an Appx manifest (`AppxManifest.xml`) and visual assets (icons), but **no application binaries**.
- In the manifest:
  ```xml
  <Properties>
    <uap10:AllowExternalContent>true</uap10:AllowExternalContent>
  </Properties>
  ```
- The external content points to the application's actual install directory (e.g., `%LOCALAPPDATA%\Programs\upit`).
- **Purpose**: Gives an unpackaged Win32 application full **Package Identity** (Package Family Name, AppUserModelID, registered COM servers, modern context menus, modern toast notifications) without forcing the app into the virtualized MSIX container / file system redirection.

### 3.2 Registration and Unregistration
Source: [Microsoft Learn: Register the identity package in your installer](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/grant-identity-to-nonpackaged-apps#register-the-identity-package-in-your-installer).

#### Per-User Installation (No Admin Privileges Needed)
- **Registration**:
  ```powershell
  Add-AppxPackage -Path "<PathTo>\upit-identity.msix" -ExternalLocation "<InstallDirectory>"
  ```
- **Unregistration**:
  ```powershell
  Get-AppxPackage "Upit.App" | Remove-AppxPackage
  ```
- Alternatively, via WinRT `Windows.Management.Deployment.PackageManager`:
  ```csharp
  var pm = new PackageManager();
  var options = new AddPackageOptions { ExternalLocationUri = new Uri(installDir) };
  var result = await pm.AddPackageByUriAsync(new Uri(msixPath), options);
  ```

#### Machine-Wide Installation (Requires Admin Elevation)
- Registration requires staging and provisioning:
  ```powershell
  Add-AppxPackage -Stage "<PathTo>\upit-identity.msix" -ExternalLocation "<InstallDirectory>"
  Add-AppxProvisionedPackage -Online -PackagePath "<PathTo>\upit-identity.msix"
  ```

### 3.3 Signing & Trust Requirements
Source: [Microsoft Learn: Sign an app package using SignTool](https://learn.microsoft.com/en-us/windows/msix/package/sign-app-package-using-signtool).
- **Hard Requirement**: Unlike traditional registry edits or unsigned `.exe` files, **Windows will refuse to install any Appx/MSIX package (sparse or full) that is not signed with a cryptographically valid and trusted certificate**.
- **Error on Untrusted Root**: If signed with an ad-hoc self-signed certificate not in the machine's `Trusted People` or `Trusted Root Certification Authorities` store, `Add-AppxPackage` fails with:
  ```
  CERT_E_UNTRUSTEDROOT (0x800B0109): A certificate chain processed, but terminated in a root certificate which is not trusted by the trust provider.
  ```
- **Development vs. Production**:
  - *Local Dev / Testing*: Can generate a self-signed certificate via PowerShell `New-SelfSignedCertificate`, sign with `SignTool.exe`, and import the public key (`.cer`) into `Cert:\CurrentUser\TrustedPeople` (per-user) or `Cert:\LocalMachine\TrustedPeople` (machine-wide).
  - *End-User Distribution*: Must be signed with a commercial Authenticode Code Signing certificate (from a trusted public CA like DigiCert, Sectigo) or Microsoft Azure Trusted Signing.

---

## 4. Wails v3 Packaging Integration

Source: [Wails v3 Official Windows Packaging Guide](https://v3.wails.io/guides/build/windows), [Wails v3 Signing Guide](https://v3.wails.io/guides/build/signing).

### 4.1 Built-in Packaging Formats
Wails v3 provides native CLI and Taskfile support for two Windows packaging formats:
1. **NSIS Installer (Default)**:
   - Command: `wails3 package GOOS=windows`
   - Generates: `build/windows/nsis/<AppName>-installer.exe`
   - Configured via: `build/windows/nsis/project.nsi`
   - Can easily execute custom registry write commands (`WriteRegStr HKCU ...`) during install and `DeleteRegKey HKCU ...` during uninstall.
2. **MSIX Package**:
   - Command: `wails3 package GOOS=windows FORMAT=msix`
   - Generates: `bin/<AppName>-<arch>.msix`
   - Uses `makeappx.exe` and `signtool.exe` from Windows SDK.
   - Signs package using variables configured in `build/windows/Taskfile.yml` (`SIGN_CERTIFICATE`, `SIGN_THUMBPRINT`, `TIMESTAMP_SERVER`).

### 4.2 Shell Extension Capabilities in Wails
- Wails v3 is designed for desktop GUI applications powered by Go and WebView2.
- Wails v3 **does not provide built-in primitives for generating native Windows Explorer shell extension DLLs (`IExplorerCommand`)**.
- A native COM in-process shell extension DLL must be built using C/C++ (via MSVC or MinGW) and shipped as a separate binary asset alongside `upit.exe` and `upit-desktop.exe`.

---

## 5. Noninteractive User Feedback Mechanisms

When an upload is triggered directly from Windows File Explorer without launching the desktop GUI, the user needs fast, unobtrusive status and result feedback.

### 5.1 Feedback Channel Comparison

| Mechanism | Speed / Footprint | Packaging Requirement | Windows 10 & 11 Support | Best Used For |
| :--- | :--- | :--- | :--- | :--- |
| **System Clipboard (URL Copy)** | Immediate (< 1 ms) | None (standard Win32 API) | Universal (all versions) | Primary success artifact (paste-ready URL) |
| **Windows Toast Notification (WinRT)** | Modern, rich (supports action buttons, hero image) | Requires AppUserModelID on shortcut or Package Identity | Windows 10 & 11 | Rich success ("URL copied to clipboard") / failure alerts |
| **Legacy Balloon Tip (`Shell_NotifyIcon`)** | Minimal, classic Win32 | None (tray icon required) | Universal | Lightweight unpackaged notifications |
| **Standard Audio Cue (`MessageBeep`)** | Instant audio feedback | None (`user32.dll`) | Universal | Subtle completion chime (`MB_OK` or `MB_ICONASTERISK`) |

### 5.2 Recommended Feedback Contract
1. **On Upload Trigger**: Cursor briefly shows working/background spinner (`SetCursor`).
2. **On Upload Success**:
   - URL copied directly to Windows Clipboard.
   - Fire a Windows Toast Notification:
     - Title: `Upload Complete`
     - Body: `https://upit.example.com/xyz (copied to clipboard)`
     - Clicking notification can open URL in default browser or open `upit-desktop` history.
3. **On Upload Failure**:
   - Fire an error notification:
     - Title: `Upload Failed`
     - Body: Specific error diagnostic (e.g., `Endpoint 500: Server error` or `Network timeout`).

---

## 6. Route Comparison: Minimum Viable vs Modern First-Class

| Dimension | Route A: Minimum Viable Supported (MVR)<br>*(Per-User Static Shell Verb)* | Route B: Modern First-Class Route<br>*(IExplorerCommand + Signed Sparse Package)* |
| :--- | :--- | :--- |
| **Windows 11 UI Position** | Sub-menu (**"Show more options"** / `Shift+F10`) | **Primary top-level context menu** |
| **Windows 10 UI Position** | **Primary context menu** | **Primary context menu** |
| **Required Binaries** | `upit.exe` (Go CLI) only | `upit.exe` + `upit-shell-ext.dll` (C++ COM DLL) |
| **Build & Toolchain** | Pure Go (`go build`) | Go toolchain + C++ compiler (MSVC or MinGW) |
| **Packaging & Format** | Standalone `.exe` or standard NSIS installer | Appx Manifest + MakeAppx + MSIX / NSIS bundle |
| **Code Signing** | Optional (standard Authenticode recommended to avoid SmartScreen) | **Mandatory** (MSIX registration fails without trusted CA certificate) |
| **Install Elevation** | **Per-user (no admin rights)** (`HKCU\Software\Classes`) | **Per-user supported** (`Add-AppxPackage -ExternalLocation`), but requires pre-trusted certificate |
| **Uninstall Cleanliness** | Deletes single registry tree in HKCU | `Remove-AppxPackage` cleanly removes identity and COM hooks |
| **Multi-Select Handling** | `MultiSelectModel = "Single"` hides verb on multi-selection | `IExplorerCommand` inspects `IShellItemArray` programmatically |
| **Launch Overhead** | 0 ms Explorer delay; fast Go CLI process invocation | Fast COM DLL dispatch; launches Go CLI process on invoke |

---

## 7. Open Design Questions for v0.6

1. **Selection Scope**:
   - *Question*: Should the Explorer verb appear on folders as well as regular files?
   - *Evidence*: `HKEY_CLASSES_ROOT\*\shell` applies strictly to files. Folder background (`Directory\Background`) or folders (`Directory`) require separate keys. For v0.6 single-file direct upload, scoping strictly to files (`*`) is cleanest and prevents ambiguity regarding folder zipping/archiving.
2. **Failure Transparency**:
   - *Question*: If an upload fails silently in the background, how does the user troubleshoot?
   - *Evidence*: Without GUI feedback, users can be confused if nothing pastes from clipboard. An interactive toast notification with a "View Log" button or a direct fallback error popup (`MessageBoxW`) ensures errors are never dropped silently.
3. **CLI Self-Registration Command**:
   - *Question*: Should `upit.exe` provide self-registration flags like `upit windows-shell register` and `upit windows-shell unregister`?
   - *Evidence*: Portable/zip distributions benefit tremendously from self-registration commands that write to `HKCU\Software\Classes\*\shell\upit.upload` without needing a full NSIS installer.
