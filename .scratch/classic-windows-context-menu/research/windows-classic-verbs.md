# Windows Classic Static Context-Menu Verbs Research

## Overview & Scope

This research investigates replacing Upit's Windows 11 packaged `IExplorerCommand` primary-menu integration with an unpackaged, per-user classic static verb displayed under File Explorer's "Show more options", while preserving Upit's **File Manager Upload** and **File Manager Integration** contracts as defined in `CONTEXT.md`.

All technical claims are derived from official Microsoft documentation (Microsoft Learn / MSDN), first-party Windows Developer publications, and Win32 SDK specifications. Documented facts and architectural inferences are explicitly separated.

---

## 1. Primary Sources & Citations

1. **Creating Shortcut Menu Handlers**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/context-menu-handlers
2. **Choosing a Static or Dynamic Shortcut Menu Method**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/shortcut-choose-method
3. **Best Practices for Shortcut Menu Handlers and Multiple Verbs**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/verbs-best-practices
4. **How to Employ the Verb Selection Model**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/how-to-employ-the-verb-selection-model
5. **Association Arrays**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/fa-associationarray
6. **Application Registration**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/app-registration
7. **Best Practices for File Associations**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/fa-best-practices
8. **Registering Shell Extension Handlers**
   Microsoft Learn (Win32 apps / Desktop Environment / Windows Shell):
   https://learn.microsoft.com/en-us/windows/win32/shell/reg-shell-exts
9. **HKEY_CLASSES_ROOT Key & Merged View**
   Microsoft Learn (Win32 apps / System Services / SysInfo):
   https://learn.microsoft.com/en-us/windows/win32/sysinfo/hkey-classes-root-key
   https://learn.microsoft.com/en-us/windows/win32/sysinfo/merged-view-of-hkey-classes-root
10. **Extending the Context Menu and Share Dialog in Windows 11**
    Windows Developer Blog (Xander Fiss, Lead Program Manager):
    https://blogs.windows.com/windowsdeveloper/2021/07/19/extending-the-context-menu-and-share-dialog-in-windows-11/
11. **Application User Model IDs (AppUserModelIDs)**
    Microsoft Learn (Win32 apps / Shell):
    https://learn.microsoft.com/en-us/windows/win32/shell/appids
12. **Sending a Toast Notification from the Desktop / Enable Desktop Toast with AppUserModelID**
    Microsoft Learn (Win32 apps / Shell):
    https://learn.microsoft.com/en-us/windows/win32/shell/quickstart-sending-desktop-toast
    https://learn.microsoft.com/en-us/windows/win32/shell/enable-desktop-toast-with-appusermodelid
13. **Respond to Toast Activations / INotificationActivationCallback**
    Microsoft Learn (Win32 apps / Shell / Notification Activation Callback):
    https://learn.microsoft.com/en-us/previous-versions/windows/desktop/win32_tile_badge_notif/respond-to-toast-activations
    https://learn.microsoft.com/en-us/windows/win32/properties/props-system-appusermodel-toastactivatorclsid
14. **How Explorer Detects Long File Names**
    The Old New Thing (Raymond Chen, Microsoft):
    https://devblogs.microsoft.com/oldnewthing/20041020-00/?p=37523
15. **Simplifying Context Menu Extensions with IExecuteCommand**
    The Old New Thing (Raymond Chen, Microsoft):
    https://devblogs.microsoft.com/oldnewthing/20100312-01/?p=14623

---

## 2. Windows 11 Context Menu Architecture & "Show More Options"

### Documented Facts
- **Primary vs. Classic Context Menu**:
  In Windows 11, the primary context menu was redesigned to address menu length, clustering, and performance issues. Apps extending the primary context menu MUST use `IExplorerCommand` coupled with package identity (MSIX or sparse package with external location).
  *(Source: Windows Developer Blog, "Extending the Context Menu and Share Dialog in Windows 11")*
- **Classic Verb Relegation**:
  Traditional verb registrations (`shell\<verb>\command` under `HKCR` or `HKCU\Software\Classes`) and in-process `IContextMenu` handlers are not shown on the Windows 11 primary context menu.
  Instead, selecting "Show more options" (or pressing `Shift+F10` or the keyboard Menu key) loads the Windows 10 context menu as-is. In this legacy menu, all classic verbs appear unchanged.
  *(Source: Windows Developer Blog, "Extending the Context Menu and Share Dialog in Windows 11")*
- **In-Process vs. Out-of-Process Activation**:
  Static verbs using `CreateProcess` (command line string) are executed out-of-process without loading any third-party code into the `explorer.exe` process address space.
  *(Source: MS Learn, "Choosing a Static or Dynamic Shortcut Menu Method")*

### Consequences of Superseding ADR 0015
ADR 0015 explicitly chose packaged `IExplorerCommand` because classic per-user registry verbs were relegated to "Show more options". Superseding ADR 0015 has the following direct impacts:
1. **User Workflow Change**: On Windows 11, invoking File Manager Upload requires an extra click on "Show more options" (or `Shift+F10`) unless the user has altered OS configuration to restore classic menus globally.
2. **Elimination of Packaging Overhead**: Removes the requirement for MSIX package identity, `AppxManifest.xml`, developer mode or code-signing certificates required for package deployment (`Add-AppxPackage`), and sparse package registration logic.
3. **Elimination of In-Process Shell Extension**: Removes `upit-explorer-command.dll` (the C++ COM DLL). Upit runs entirely as an on-demand external executable without binary injection into Explorer.
4. **Toast Notification Dependency Broken**: The current Windows clean-success Toast delivery script (`cmd/upit-file-manager/feedback_windows.go`) locates its AppUserModelID via `Get-AppxPackage -Name 'HungNth.Upit'`. Without package identity, this query returns `$null` and Toast notifications fail/fall back. (See Section 10 for complete unpackaged toast analysis).

---

## 3. Supported Per-User Registry Locations for One Regular File

### Documented Facts
- **Per-User Registry Store**:
  `HKEY_CLASSES_ROOT` (`HKCR`) is a composite merged view of `HKEY_LOCAL_MACHINE\Software\Classes` (system-wide) and `HKEY_CURRENT_USER\Software\Classes` (`HKCU\Software\Classes`, per-user).
  When an unprivileged process writes to `HKCR`, Windows redirects or fails the write depending on whether the key exists in HKLM. Microsoft documentation explicitly states:
  > *"To change the settings for the interactive user, store the changes under `HKEY_CURRENT_USER\Software\Classes` rather than `HKEY_CLASSES_ROOT`."*
  *(Source: MS Learn, "HKEY_CLASSES_ROOT Key" & "Merged View of HKEY_CLASSES_ROOT")*
- **File System Association Arrays**:
  Windows Explorer builds an ordered association array to determine which verbs apply to a selected item.
  For a file system item, the association hierarchy from specific to generic includes:
  1. ProgID associated with the extension (e.g. `HKCR\txtfile`)
  2. Per-extension SystemFileAssociations (e.g. `HKCR\SystemFileAssociations\.txt`)
  3. Perceived type (e.g. `HKCR\SystemFileAssociations\document`)
  4. Universal file class: `HKEY_CLASSES_ROOT\*` — defined as **"All files (non-folders)"**
  5. Universal file system class: `HKEY_CLASSES_ROOT\AllFilesystemObjects` — defined as **"Files and file system folders"**
  *(Source: MS Learn, "Association Arrays", "Registering Shell Extension Handlers")*
- **Targeting Only Regular Files (Excluding Directories)**:
  - Registering under `*` applies to all regular files regardless of extension, and does **not** apply to folders or directories.
  - Registering under `AllFilesystemObjects` applies to both files and folders.
  - Registering under `Directory` or `Folder` applies to file folders or shell containers.
  Therefore, to target files and strictly exclude directories without writing custom filters, `*` is the documented, canonical key.

### Concrete Per-User Path
For Upit's static verb:
- Key: `HKEY_CURRENT_USER\Software\Classes\*\shell\<VerbName>`
- Command Key: `HKEY_CURRENT_USER\Software\Classes\*\shell\<VerbName>\command`

---

## 4. The Smallest Correct Registry Contract for Upit

To register Upit's File Manager Upload for exactly one regular file, the smallest correct registry contract consists of **two keys and four values** under `HKEY_CURRENT_USER\Software\Classes`:

```reg
[HKEY_CURRENT_USER\Software\Classes\*\shell\Upit.Upload]
@="Upload with Upit"
"Icon"="\"C:\\Users\\<User>\\AppData\\Local\\Programs\\Upit\\upit-file-manager.exe\",0"
"MultiSelectModel"="Single"

[HKEY_CURRENT_USER\Software\Classes\*\shell\Upit.Upload\command]
@="\"C:\\Users\\<User>\\AppData\\Local\\Programs\\Upit\\upit-file-manager.exe\" \"%1\""
```

### Purpose of Each Value
1. **Verb Key Name (`Upit.Upload`)**:
   Prefixing the verb with an independent software vendor (ISV) prefix is an official Microsoft best practice (`ISVName.verb`) to prevent key collisions with other installed software.
   *(Source: MS Learn, "Best Practices for Shortcut Menu Handlers and Multiple Verbs")*
2. **Display Name (`@` / Default Value)**:
   A `REG_SZ` value setting the human-readable text shown on the context menu (`"Upload with Upit"`). Alternatively, `MUIVerb` can be used.
   *(Source: MS Learn, "Creating Shortcut Menu Handlers")*
3. **`Icon` (`REG_SZ` or `REG_EXPAND_SZ`)**:
   Specifies the icon resource displayed next to the verb in File Explorer's classic context menu. Format is `"<path-to-binary>",<resource-index>` or a path to an `.ico` file.
   *(Source: MS Learn, "Choosing a Static or Dynamic Shortcut Menu Method")*
4. **`MultiSelectModel` = `"Single"` (`REG_SZ`)**:
   Restricts the verb strictly to single-file selections. If 2 or more files are selected, Explorer suppresses the verb entirely.
   *(Source: MS Learn, "How to Employ the Verb Selection Model")*
5. **Command Line (`@` / Default Value on `command`)**:
   A `REG_SZ` (or `REG_EXPAND_SZ`) string containing the fully qualified executable path and `"%1"` parameter.
   *(Source: MS Learn, "Best Practices for File Associations", "Creating Shortcut Menu Handlers")*

---

## 5. Command Invocation & Quoting Semantics

### Documented Facts
- **Replacement Parameter `%1`**:
  `%1` represents the absolute file system path of the target item passed by Explorer to the process.
  *(Source: MS Learn, "Creating Shortcut Menu Handlers")*
- **Quoting Requirements**:
  Microsoft documentation explicitly mandates wrapping both expanding strings and the `%1` argument in double quotation marks:
  > *"Expanding strings can contain spaces when they expand. Because spaces are often interpreted as argument delimiters, they cause problems under certain circumstances... The arguments are interpreted correctly, however, regardless of whether they contain spaces, if the expanding strings are wrapped in quotation marks as follows: `"%SYSTEMROOT%\MyProgram" "%1" "%2"`"*
  *(Source: MS Learn, "Best Practices for File Associations")*
- **Long File Name (LFN) Preservation**:
  As documented by Raymond Chen, Explorer inspects the executable path in the command string:
  - If the path to the executable contains spaces and is unquoted (e.g. `C:\Program Files\Upit\upit.exe`), Explorer parses up to the first space (`C:\Program`), fails to find the executable, assumes the program does not support Long File Names, and passes an 8.3 short path.
  - Quoting the executable path (`"C:\Program Files\Upit\upit.exe" "%1"`) guarantees that Explorer locates the binary, recognizes LFN support, and passes the complete long path with spaces preserved.
  *(Source: The Old New Thing, "How does Explorer detect whether your program supports long file names?")*
- **`%1` vs. `%L` vs. `%*`**:
  - `%1`: Standard parameter for the primary selected file.
  - `%L`: Historical flag (Windows 95/NT 4.0) used to force long file names; superseded by `%1` with proper quoting.
  - `%*`: Passes all command-line arguments; used in batch files, not applicable to single-file static shell verbs.

---

## 6. Multi-Selection Behavior & Enforcement

### Documented Facts
- **Selection Models**:
  Windows Shell provides three models via the `MultiSelectModel` registry value:
  - `Single`: The verb supports only a single selection.
  - `Player`: The verb supports any number of items passed simultaneously.
  - `Document`: The verb creates a top-level window for each item (up to 15 items in legacy CreateProcess verbs).
  *(Source: MS Learn, "How to Employ the Verb Selection Model")*
- **Suppression Rule**:
  > *"When the number of items selected does not match the verb selection model or is greater than the default limits outlined in the following table, the verb fails to appear."*
  *(Source: MS Learn, "How to Employ the Verb Selection Model")*
- **Default Fallback**:
  If `MultiSelectModel` is omitted for a static command verb, Explorer defaults to `Document`. If a user selects 10 files and invokes the verb, Explorer will spawn **10 independent processes** simultaneously.
- **Contract Fulfillment**:
  Because Upit's File Manager Upload supports exactly one regular file, setting `MultiSelectModel = "Single"` guarantees that the context menu entry will **not appear** if the user selects multiple files, enforcing the product contract before any process is launched.

---

## 7. Dynamic Applicability Filters (Advanced Query Syntax)

### Documented Facts
- **AQS via `AppliesTo`**:
  Windows 7 and later support dynamic filtering for static verbs using Advanced Query Syntax (AQS) via the `AppliesTo` value (`REG_SZ`) under the verb key:
  ```reg
  "AppliesTo"="System.ItemType:<>directory"
  ```
  *(Source: MS Learn, "Getting Dynamic Behavior for Static Verbs by Using Advanced Query Syntax")*
- **Necessity for Upit**:
  - Because Upit registers under `HKCU\Software\Classes\*`, the registration inherently targets only files and excludes directories at the association array level.
  - `MultiSelectModel = "Single"` ensures that selections of $>1$ items are excluded.
  - Therefore, an explicit `AppliesTo` filter is **not required** for basic operation, keeping the registry contract minimal. It remains available if the product owner wants to exclude specific file attributes (e.g., hidden or system files) without COM code.

---

## 8. Lifecycle & Uninstall Cleanup

### Documented Facts
- **Per-User Deletion**:
  Because all keys reside under `HKEY_CURRENT_USER\Software\Classes\*\shell\Upit.Upload`, uninstall or de-registration does not require administrator privileges.
  Cleaning up requires deleting the `Upit.Upload` subkey tree:
  ```powershell
  Remove-Item -Path "HKCU:\Software\Classes\*\shell\Upit.Upload" -Recurse -ErrorAction SilentlyContinue
  ```
- **Shell Notification**:
  When file associations or verbs are added or removed, the calling process should notify Explorer using `SHChangeNotify`:
  ```cpp
  SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_IDLIST, NULL, NULL);
  ```
  This forces Explorer to invalidate cached association arrays immediately without requiring a logoff or system restart.
  *(Source: MS Learn, "Registering Shell Extension Handlers")*

---

## 9. Security, Signing & Execution Policies

### Documented Facts & Comparisons

| Policy / Control | Packaged `IExplorerCommand` (Current v0.6) | Unpackaged Classic Static Verb |
| :--- | :--- | :--- |
| **Package Identity** | Mandatory (MSIX or sparse package manifest). | None required. |
| **Code Signing Requirement** | Mandatory. Windows blocks package deployment without a trusted certificate chain. | Not required by Windows Shell to register or launch via registry verb. |
| **AppLocker** | Evaluated under **Packaged app rules** (`Appx`). | Evaluated under **Executable rules** (`.exe`). Unsigned binaries or binaries in user-writable paths (`%LOCALAPPDATA%`) may be blocked if strict path/publisher rules exist. |
| **WDAC (Windows Defender Application Control)** | Requires signed package and certificate rule. | Requires binary signature if WDAC enforcement policy requires publisher rules. Unsigned binaries blocked under strict WDAC. |
| **Defender SmartScreen** | Inspected during package installation; reputation tracked against signer. | Inspected if binary carries Mark of the Web (`Zone.Identifier`). If installed via uninstaller/installer without MotW, SmartScreen does not block standard local process launches. |

*(Sources: MS Learn, "Windows Defender Application Control", "AppLocker technical reference")*

---

## 10. Windows 11 Toast Notifications for Unpackaged Win32 Apps

### Context & Constraint
Upit's existing clean-success notification on Windows (`cmd/upit-file-manager/feedback_windows.go`) executes a PowerShell script that derives the Application User Model ID (AUMID) by querying:
```powershell
$pkg = Get-AppxPackage -Name 'HungNth.Upit'
$aumid = $pkg.PackageFamilyName + '!Upit'
```
If Upit supersedes ADR 0015 and removes package identity, `Get-AppxPackage` returns `$null`, exiting with code `2` (unavailable).

### Official-Source Capabilities for Unpackaged Win32 Toasts
- **Can an unpackaged Win32 app deliver Windows 11 Toast notifications without MSIX or code signing?**
  **YES.** Documented directly in Microsoft Learn Win32 desktop shell guidance.
  *(Source: MS Learn, "Sending a toast notification from the desktop", "How to enable desktop toast notifications through an AppUserModelID")*

### Architectural Requirements for Unpackaged Desktop Toasts
1. **Explicit Application User Model ID (AUMID)**:
   The app must define an explicit AUMID string matching the standard pattern:
   `CompanyName.ProductName[.SubProduct]` (e.g. `HungNth.Upit`), max 128 characters without spaces.
   *(Source: MS Learn, "Application User Model IDs (AppUserModelIDs)")*
2. **Start Menu Shortcut with `PKEY_AppUserModel_ID`**:
   The Windows Notification subsystem strictly requires that a valid shortcut (`.lnk` file) bearing the application's AUMID exist in the user's Start Menu programs directory:
   `%APPDATA%\Microsoft\Windows\Start Menu\Programs\<ShortcutName>.lnk`
   - The shortcut's `IPropertyStore` MUST have `System.AppUserModel.ID` (`PKEY_AppUserModel_ID` = `9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3, 5`) set to the identical AUMID string.
   - Without this shortcut, calls to `ToastNotificationManager::CreateToastNotifier(aumid)` or `Show(toast)` will fail, or Windows Action Center will silently drop the notification.
   *(Source: MS Learn, "How to enable desktop toast notifications through an AppUserModelID")*
3. **Per-User Viability**:
   The Start Menu shortcut location is `%APPDATA%\Microsoft\Windows\Start Menu\Programs\`, which resides in the user's roaming profile. This does **not** require administrator privileges.
4. **COM Activation vs. No-Activation Silent Toast**:
   - **Interactive / Background Activation**: If a toast has clickable buttons or responds to user selection when the process is closed, Windows requires a registered out-of-proc COM server implementing `INotificationActivationCallback`, with its CLSID registered in `HKCU\Software\Classes\CLSID\{GUID}\LocalServer32` and referenced on the shortcut via `System.AppUserModel.ToastActivatorCLSID`.
   - **Silent No-Activation Toast (Upit Contract)**:
     - Upit's clean-success Toast is purely informational: `<audio silent="true"/>` with title `Upload complete` and body `Final URL copied to clipboard.`
     - It contains no buttons (`<actions>`), no `launch` argument, and no interactive activation callback.
     - Therefore, **no COM server registration (`LocalServer32`) and no `ToastActivatorCLSID` are required** to deliver the notification.
     - *Caveat*: If a user explicitly clicks the body of a toast that has no COM activator, Windows falls back to launching the shortcut target (`upit.exe`). If the shortcut points to an interactive UI or a no-op handler, that behavior must be considered.

### Options & Constraints for Upit Clean-Success Notifications
- **Option 1: Retain WinRT Toast via Per-User Start Menu Shortcut**
  - Install a shortcut `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Upit.lnk` with `PKEY_AppUserModel_ID` = `"HungNth.Upit"`.
  - Update `feedback_windows.go` PowerShell script to call:
    `[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("HungNth.Upit")`
    directly without calling `Get-AppxPackage`.
  - *Pros*: Preserves the silent Windows 11 native Toast behavior.
  - *Cons*: Adds a Start Menu shortcut lifecycle dependency to File Manager Integration.
- **Option 2: Silent Exit Without Toast for File Manager Upload**
  - In unpackaged mode without a Start Menu shortcut, File Manager Upload copies the Final URL to the clipboard and exits silently on clean success. Non-clean outcomes retain Task Dialog / Message Box.
  - *Pros*: Zero external footprint beyond the single context menu registry key.
  - *Cons*: Diverges from ADR 0020's goal of native visual completion feedback.
- **Option 3: Fall Back to Task Dialog / Balloon Tip**
  - Note: ADR 0020 explicitly rejected Task Dialogs for clean success because they block until dismissed. Standard system tray balloon tips require a resident tray icon.

---

## 11. Separation of Documented Facts vs. Inferences

### Documented Facts
1. Windows 11 context menu relegates all classic static verbs to "Show more options" (`Shift+F10`). Only packaged apps with `IExplorerCommand` appear on the primary menu.
2. `HKCU\Software\Classes\*\shell\<Verb>\command` is the documented per-user registry location for commands that apply to all regular files.
3. `MultiSelectModel = "Single"` instructs File Explorer to suppress the verb when 2 or more files are selected.
4. Unquoted executable paths containing spaces cause Explorer to pass 8.3 short paths. Wrapping both executable path and `"%1"` in double quotes preserves long file names.
5. Unpackaged Win32 applications can raise native Windows Toasts if a Start Menu shortcut exists with `PKEY_AppUserModel_ID` matching the notifier AUMID.
6. A silent, non-interactive toast does not require COM `INotificationActivationCallback` or `ToastActivatorCLSID` registration.
7. Unpackaged registry verbs and per-user Start Menu shortcuts do not require administrator elevation or code signing certificates.

### Inferences & Architectural Deductions
1. **Product UX Trade-off**: Moving to a classic verb will require Windows 11 users to click "Show more options" or press `Shift+F10`. This is an unavoidable Windows 11 OS design boundary.
2. **Helper Direct Launch**: Launching `upit-file-manager.exe "%1"` directly from the `command` key is more efficient and reliable than chaining through a wrapper script or generic CLI dispatcher.
3. **Start Menu Shortcut vs. Toast Retention**: If product requirements dictate keeping native Windows 11 Toasts without MSIX, creating a per-user shortcut at install time is the least invasive mechanism documented by Microsoft.

---

## 12. Uncertainties & Decisions for Product Owner

1. **UX Location Acceptance**:
   Does the product owner accept that on Windows 11, Upit's context menu command will reside under "Show more options" (2 clicks or `Shift+F10`), abandoning top-level primary menu placement to eliminate MSIX/signing dependencies?
2. **Clean-Success Feedback Policy under Unpackaged Verb**:
   If package identity is removed:
   - Should Upit install a per-user Start Menu shortcut with `System.AppUserModel.ID` so that native Toast notifications continue to work?
   - OR should clean success complete silently (clipboard copied, process exits with code 0, no notification) when running unpackaged without a Start Menu shortcut?
3. **Executable Target on Command Line**:
   Should the registry verb invoke `upit-file-manager.exe "%1"` directly, or should `upit.exe` provide a private dispatch command (e.g. `upit.exe file-manager "%1"`) to present a single user-facing executable file?
4. **Handling Non-Regular Files**:
   While `*` excludes directories, what should the helper do if invoked on special files (e.g., symbolic links, named pipes, zero-byte system files)? The helper already enforces single regular file validation at startup.