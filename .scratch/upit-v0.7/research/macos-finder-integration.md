# macOS Finder Direct-Upload Integration Research

## Executive Summary

This document evaluates integration mechanisms for adding a macOS Finder contextual action to upload files directly using `upit` without opening the `upit-desktop` main user interface. It focuses on the sequential roadmap (`v0.5` Desktop Configuration and Manual Upload, `v0.6` Windows File Explorer, `v0.7` macOS Finder), strictly adhering to the architectural contract: direct upload executes with the configured default Uploader and default Shortener, reports noninteractive success/failure feedback (or notification), and leaves the Manual Upload interface solely for users who explicitly launch `upit-desktop`.

We analyze the following mechanisms against Apple's platform architecture, sandboxing, code signing, notarization, and distribution models:
1. **macOS Services (`NSServices` / Services menu & Finder Quick Actions)**
2. **Action Extensions (`com.apple.ui-services` / `com.apple.services`)**
3. **Share Extensions (`com.apple.share-services`)**
4. **Finder Sync Extensions (`FIFinderSync`)**
5. **Shortcuts / Automator / Quick Action Workflows (`.workflow` / `.shortcut`)**

**Recommended Default:** **Finder Service (`NSServices`) targeting a small native helper or the `upit` CLI via an application bundle wrapper (`upit.app` / `Upit Launcher.app`), or an un-sandboxed `NSServices` provider inside `upit-desktop.app` with `LSUIElement=1` background execution.**

---

## Architecture & Integration Mechanisms

### 1. macOS Services (`NSServices`)

#### Mechanism & Menu Placement
`NSServices` is macOS's native system-wide integration architecture (originating in NeXTSTEP and maintained through macOS 15+ Sequoia). Any valid `.app` bundle can declare services in its `Info.plist` under the `NSServices` key.

In macOS Finder:
- Services appear in the **Finder context menu** under **Services** (or promoted directly to the root contextual menu if space permits, as well as under **Finder > Services** in the menu bar).
- In modern macOS (macOS 10.14 Mojave through macOS 15 Sequoia), services configured for file types are also surfaced under **Quick Actions** in the Finder Preview pane and contextual menu.

#### Declaration in `Info.plist`
```xml
<key>NSServices</key>
<array>
    <dict>
        <key>NSMenuItem</key>
        <dict>
            <key>default</key>
            <string>Upload with Upit</string>
        </dict>
        <key>NSMessage</key>
        <string>uploadFileService</string>
        <key>NSPortName</key>
        <string>Upit</string>
        <key>NSSendFileTypes</key>
        <array>
            <string>public.item</string>
        </array>
        <key>NSRequiredContext</key>
        <dict>
            <key>NSApplicationIdentifier</key>
            <string>com.apple.finder</string>
        </dict>
    </dict>
</array>
```

#### File Handoff Mechanics
When invoked:
1. macOS launches the application (if not already running) or sends an Apple Event / Mach port message to the running instance registered under `NSPortName`.
2. The Cocoa runloop delivers the message to the application delegate or service listener:
   `[NSApp registerServicesMenuSendTypes:returnTypes:]` or an instance method `uploadFileService:userData:error:`.
3. The method signature:
   `- (void)uploadFileService:(NSPasteboard *)pboard userData:(NSString *)userData error:(NSString **)error;`
4. The service extracts file paths/URLs from `NSPasteboard`:
   ```objc
   NSArray<NSURL *> *fileURLs = [pboard readObjectsForClasses:@[[NSURL class]] options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
   ```
5. Exact URL Handoff: The pasteboard delivers absolute `file://` URLs conforming to `public.item`.

#### Bundle Requirements & Execution
- **Must be a bundled `.app`**: Standalone Mach-O CLI binaries (`bin/upit`) cannot register `NSServices`. Only `.app` bundles inspected by Launch Services can register services.
- **App Launch Behavior**: By default, invoking a service launches the `.app` bundle. If the target is `upit-desktop.app` without modification, launching the app would display the main GUI window, violating the requirement that direct upload must not open `upit-desktop`.
- **Solution for Background Direct Upload**:
  - *Option A (Dedicated Helper App)*: A dedicated lightweight Cocoa launcher bundle (e.g., `UpitService.app` or `UpitDirectUpload.app` containing `LSUIElement=1` / `LSBackgroundOnly=1`) that delegates directly to `upit upload <file>` via `execv` or executes Go application logic headlessly.
  - *Option B (Service Handler in `upit-desktop.app`)*: If `upit-desktop` is built with Cocoa lifecycle hooks, it can detect invocation via `NSMessage` before opening any `WebviewWindow`, execute the upload silently with notification feedback, and terminate immediately without presenting a window.

---

### 2. Action Extensions (`com.apple.ui-services` / `com.apple.services`)

#### Mechanism & Menu Placement
Action Extensions (introduced in macOS 10.10, updated through AppKit / ExtensionFoundation) allow apps to vend contextual actions.
- Identified by `NSExtensionPointIdentifier`:
  - `com.apple.services`: Non-UI Action Extension.
  - `com.apple.ui-services`: UI Action Extension (subclass of `NSViewController`).
- Placed in Finder's **Quick Actions** menu (Finder contextual menu > Quick Actions, and Finder Preview pane).

#### Constraints & Sandbox Isolation
- **Mandatory App Sandbox**: All modern App Extensions (`.appex`) **must** be sandboxed (`com.apple.security.app-sandbox = true`). Non-sandboxed `.appex` bundles are rejected by macOS Gatekeeper and the App Store.
- **Configuration Set Access**: Upit stores configuration at `~/.config/upit/`. Under App Sandbox:
  - The process home directory `~` is redirected to `~/Library/Containers/<bundle-id>/Data/`.
  - The extension *cannot* access `~/.config/upit/config.json` or `uploaders.json` unless granted access via Powerbox user open dialog or App Group containers (`group.<team-id>.upit`).
  - Accessing the global `~/.config/upit` path breaks the unified Configuration Set location defined in `docs/adr/0001-use-fixed-user-config-directory.md`.
- **Packaging Burden**: Requires an `.appex` bundle placed inside `Contents/PlugIns/` of the main `.app`. An `.appex` must be compiled as a Mach-O dynamic library or native executable linked against Apple's ExtensionKit/AppKit frameworks (Objective-C or Swift). Wails v3 does not build or package `.appex` plugins.

---

### 3. Share Extensions (`com.apple.share-services`)

#### Mechanism & Menu Placement
Share Extensions allow sharing selected content to a service via macOS system Share sheets and Finder's **Share** contextual submenu.

#### Constraints
- Appears under **Share > [App Name]**, not as a direct root contextual item or custom action.
- By design, presents standard macOS Sharing UI (`SLComposeServiceViewController` or custom UI) where user confirms sending. Bypassing the UI to perform silent, direct uploads violates Apple Human Interface Guidelines and Share sheet conventions.
- Mandatory App Sandbox applies, with the same container isolation issues as Action Extensions.

---

### 4. Finder Sync Extensions (`FIFinderSync`)

#### Mechanism & Menu Placement
The `FinderSync` framework (`FIFinderSyncController`, `FIFinderSyncProtocol`) allows apps to:
- Monitor specific directory hierarchies (`[FIFinderSyncController defaultController].directoryURLs`).
- Add custom items directly into the Finder contextual menu:
  ```swift
  override func menu(for menuKind: FIMenuKind) -> NSMenu {
      let menu = NSMenu(title: "")
      menu.addItem(withTitle: "Upload with Upit", action: #selector(uploadFile), keyEquivalent: "")
      return menu
  }
  ```
- Retrieve selected files with `FIFinderSyncController.default().selectedItemURLs()`.

#### Critical Flaws & Misalignment
- **Designed for Cloud Sync Roots**: `FIFinderSync` is intended for folder synchronization services (Dropbox, OneDrive, Nextcloud). It requires declaring monitored directories. Setting `directoryURLs = [URL(fileURLWithPath: "/")]` to monitor the entire filesystem causes severe performance penalties, elevated CPU/memory usage, and is actively discouraged or throttled in modern macOS releases.
- **Reliability & Settings Decay**: Starting with macOS 13/14/15, Finder Sync extensions have had widespread instability, frequently requiring manual user enablement in System Settings > Privacy & Security > Extensions > Added Extensions, and failing to register reliably without host reboots.
- **Mandatory App Sandbox**: Finder Sync extensions must run sandboxed, creating the same configuration boundary issue as Action Extensions.

---

### 5. Shortcuts / Automator / Quick Action Workflows

#### Mechanism & Menu Placement
Users or installers can place a `.workflow` or `.shortcut` in `~/Library/Services/` or the Shortcuts database:
- Automator Quick Actions (`.workflow`) placed in `~/Library/Services/` appear automatically in Finder's Context Menu > Quick Actions and Services.
- A workflow can invoke a shell script: `/usr/local/bin/upit upload "$1"`.

#### Tradeoffs
- **Not a self-contained application binary**: Shipping a `.workflow` requires installer scripts or user imports.
- Automator is in maintenance mode in favor of Shortcuts on macOS Monterey+. Shortcuts scripting requires Apple Events or `shortcuts run`, which has noticeable execution latency (1–2 seconds) and prompts users for permission on first run.

---

## Comparison of Viable Routes

| Dimension | Route 1: Finder Service (`NSServices`) in App Wrapper (Recommended) | Route 2: Standalone Automator Quick Action | Route 3: Native Action Extension (`.appex`) | Route 4: Finder Sync Extension (`FIFinderSync`) |
| :--- | :--- | :--- | :--- | :--- |
| **Finder UI Location** | Context Menu > Services & Quick Actions (Mojave+) | Context Menu > Quick Actions | Context Menu > Quick Actions & Preview Pane | Context Menu (direct custom menu item) |
| **File Handoff** | `NSPasteboard` (`readObjectsForClasses:@[[NSURL class]]`) | Shell argument (`$@` via `bash`/`zsh` runner) | `NSExtensionContext.inputItems` -> `NSItemProvider` | `FIFinderSyncController.selectedItemURLs()` |
| **Process Model** | Helper `.app` (`LSUIElement=1`) or headless `upit` invocation | Spawns `/bin/sh` -> `/usr/local/bin/upit` | Ephemeral plugin host process (`pkd` / extension host) | Persistent background extension process hosted by Finder |
| **Opens Desktop App?** | **No** (Helper runs background-only without UI) | **No** (Runs CLI directly) | **No** (Can run non-UI) | **No** (Finder host) |
| **Sandbox Constraints** | **Can remain unsaved/un-sandboxed** (Developer ID signed). Full access to `~/.config/upit/`. | Runs in user shell context. Full access to `~/.config/upit/`. | **Mandatory Sandbox**. Cannot read `~/.config/upit/` without entitlements/App Groups. | **Mandatory Sandbox**. Cannot read `~/.config/upit/`. |
| **Wails v3 Packaging Fit** | Standard macOS `.app` bundle; helper can be nested in `Contents/Helpers/`. | Script files; external to `.app` packaging. | Incompatible with pure Go/Wails; requires Xcode `.appex` build pipeline. | Incompatible with pure Go/Wails; requires Xcode `.appex` build pipeline. |
| **System Registration** | Automatic on bundle detection by Launch Services / `lsregister` / `NSUpdateDynamicServices`. | Copied to `~/Library/Services/`. | Registered via `pluginkit` / App Store / Application folder install. | Registered via `pluginkit` and System Settings. |

---

## Technical Deep-Dive: The Recommended Route (NSServices via Helper App)

### 1. Selected-File URL Handoff
When the user right-clicks a file in Finder and selects the Service:
1. Launch Services checks `NSSendFileTypes` in the app's `Info.plist`:
   ```xml
   <key>NSSendFileTypes</key>
   <array>
       <string>public.item</string>
   </array>
   ```
2. Launch Services invokes the application's service handler.
3. In Cocoa / Objective-C:
   ```objc
   - (void)uploadFileService:(NSPasteboard *)pboard
                    userData:(NSString *)userData
                       error:(NSString **)error {
       NSArray *classes = @[[NSURL class]];
       NSDictionary *options = @{NSPasteboardURLReadingFileURLsOnlyKey: @YES};
       NSArray<NSURL *> *urls = [pboard readObjectsForClasses:classes options:options];

       if (urls.count == 0) {
           *error = @"No valid files selected.";
           return;
       }

       // Single-file contract check:
       if (urls.count > 1) {
           // Multi-select behavior is a design choice (see Design Questions).
           // If strict single-file:
           [self showNotificationTitle:@"Upload Failed" message:@"Multi-file upload is not supported."];
           return;
       }

       NSURL *targetURL = urls.firstObject;
       [self executeDirectUpload:targetURL.path];
   }
   ```
4. The helper delegates execution:
   It calls the Wails-free Go module (`app.Service.ExecuteUpload`) directly in-process or executes `bin/upit <file-path>` with stdout/stderr redirection.

### 2. Single-File vs. Multi-File Selection
- **Single regular file**: The service receives 1 URL. Verifies via `stat` that it is a regular file (not directory, socket, or device), matching the CLI and Desktop core contract.
- **Multi-select behavior**:
  In `NSServices`, if multiple files are selected, Finder passes all selected URLs in the pasteboard array.
  *Design choice (not an assumption)*:
  - *Option 1 (Strict rejection)*: Reject if `urls.count > 1` and notify the user that Upit only accepts exactly one regular file.
  - *Option 2 (Sequential upload)*: Loop through URLs sequentially (violates the current v0.5 one-file-at-a-time invariant).
  - *Recommended Default*: Treat multi-select as unsupported in v0.7; reject immediately with an explicit notification when `urls.count > 1`, identical to drag-and-drop validation in v0.5 desktop.

### 3. Application Bundle & Launch Model
To satisfy: *"upload directly with the default Uploader and default Shortener without opening Upit Desktop"*:
- An `Info.plist` with `NSServices` must reside in an `.app` bundle recognized by Launch Services.
- **Bundle Options**:
  - **Option 1: Auxiliary Helper Bundle (`UpitDirectUpload.app`)**:
    Shipped inside `upit-desktop.app/Contents/Helpers/UpitDirectUpload.app` or as a companion bundle.
    Contains `LSUIElement = YES` (an agent application that does not show an icon in the Dock or present a window).
    When invoked, it runs headlessly, uploads the file, sends a native macOS notification, honors the Global Configuration clipboard default, and exits.
  - **Option 2: Single Bundle (`upit-desktop.app`) with Dispatch Branching**:
    `upit-desktop` registers the service. In Go / Cgo entry point:
    If launched via `NSMessage` service invocation, skip Wails `app.Window.NewWithOptions(...)`, run the headless Go upload pipeline, and exit when the upload completes.

### 4. Entitlements, Sandbox, and Configuration Access
- **Sandbox Requirement**:
  - Mac App Store applications **must** be sandboxed.
  - Direct notarized distribution outside the Mac App Store (via Developer ID) **does not require** App Sandbox.
  - Upit's Configuration Set contract (`~/.config/upit/`) relies on direct filesystem access to the user's home directory.
  - If sandboxed (`com.apple.security.app-sandbox = true`):
    - `~` resolves to `~/Library/Containers/<bundle-id>/Data/`.
    - Reading `~/.config/upit` requires the user to select the folder via `NSOpenPanel` or granting persistent security-scoped bookmarks, which violates noninteractive direct upload.
  - **Conclusion**: Upit direct upload integration on macOS must be distributed as an **un-sandboxed, Developer ID-signed application** (matching standard developer tools on macOS like Homebrew, Docker Desktop, VS Code, and Sublime Text).
  - Required Entitlements (when Hardened Runtime is enabled for Notarization):
    ```xml
    <?xml version="1.0" encoding="UTF-8"?>
    <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
    <plist version="1.0">
    <dict>
        <key>com.apple.security.cs.allow-jit</key>
        <true/>
        <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
        <true/>
        <key>com.apple.security.automation.apple-events</key>
        <false/>
    </dict>
    </plist>
    ```

### 5. Signing and Notarization Requirements
On macOS 10.15 Catalina through macOS 15 Sequoia:
1. **Code Signing**: Every Mach-O binary (`upit`, `upit-desktop`, helper binaries) and `.app` bundle must be signed with a valid "Developer ID Application" certificate:
   ```bash
   codesign --force --options runtime --sign "Developer ID Application: <Team Name> (<Team ID>)" --entitlements entitlements.plist <path-to-bundle>
   ```
2. **Hardened Runtime**: Mandatory (`--options runtime`).
3. **Notarization**: The `.app` or `.dmg` / `.pkg` must be submitted to Apple's notary service using `xcrun notarytool`:
   ```bash
   xcrun notarytool submit upit.dmg --keychain-profile "AC_PASSWORD" --wait
   ```
4. **Stapling**: The notarization ticket must be stapled to the DMG or app bundle:
   ```bash
   xcrun stapler staple upit.dmg
   ```
Without notarization, Gatekeeper blocks the application when downloaded from the internet with a warning that "Apple cannot check it for malicious software."

### 6. Install, Uninstall, and Service Registration
- **Registration**:
  - When an `.app` bundle is moved into `/Applications` or `~/Applications`, macOS Launch Services automatically discovers the bundle and registers its `NSServices`.
  - To force immediate registration without waiting for Launch Services polling or system reboot:
    ```bash
    /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -f /Applications/Upit.app
    ```
    or call the AppKit API:
    `NSUpdateDynamicServices()`
- **Enabling in Finder**:
  - Depending on macOS user settings, newly registered Services may be disabled by default in **System Settings > Keyboard > Keyboard Shortcuts > Services > Files and Folders** (or Extensions in Ventura/Sonoma/Sequoia).
  - While Launch Services registers the service, user preferences determine whether it is enabled if the user has customized their Services menu.
- **Uninstall**:
  - Deleting `Upit.app` from `/Applications` causes Launch Services to unregister the service automatically.
  - To flush immediately:
    ```bash
    /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -u /Applications/Upit.app
    ```

### 7. Noninteractive Success/Failure Feedback
Because direct upload does not open a window, feedback must follow native macOS conventions:
1. **Notifications (`UserNotifications.framework` / `UNUserNotificationCenter`)**:
   - The helper bundle requests notification permission (`UNAuthorizationOptionAlert | UNAuthorizationOptionSound`).
   - On upload start: Optional low-priority banner or progress notification.
   - On success: Deliver banner notification:
     - **Title**: `Upload Successful`
     - **Body**: `Final URL copied to clipboard: https://...`
   - On warning (e.g., Shortener fallback):
     - **Title**: `Upload Completed (Shortener Failed)`
     - **Body**: `Original URL copied to clipboard: https://...`
   - On failure:
     - **Title**: `Upload Failed`
     - **Body**: Sanitized error message (redacted, no credentials).
2. **Audio / Haptic Feedback**:
   - `[[NSSound soundNamed:@"Hero"] play]` or system alert sound on error.
3. **Clipboard Management**:
   - The headless service copies the resulting URL to `NSPasteboard` (`NSPasteboardTypeString`), honoring the global configuration's clipboard policy.

---

## Design Questions & Open Decisions for v0.7

1. **Host Executable Architecture: Helper vs. Unified Bundle**
   - *Alternative A*: Ship a single `Upit.app` bundle where `Contents/MacOS/upit-desktop` inspects its launch arguments. If launched by macOS Services (with `-psn_...` or Cocoa service invocation), it executes headless Go upload and exits without creating a Wails `WebviewWindow`.
   - *Alternative B*: Ship `upit-desktop.app` containing a dedicated background helper `Contents/Helpers/UpitService.app` (`LSUIElement=1`).
   - *Tradeoff*: Alternative A avoids maintaining two separate bundle targets, while Alternative B cleanly isolates the Wails runtime from the non-GUI service handler.

2. **Multi-Selection Policy**
   - Finder permits selecting multiple files when invoking a Service.
   - Should Upit:
     - (a) Reject the entire operation immediately if count > 1 (matching v0.5 drag-and-drop contract)?
     - (b) Sequentially upload each file and produce multiple notifications/clipboard entries?
     - *Evidence-based default*: Strictly reject multi-file selections with an error notification (`"Upit only supports uploading one file at a time"`), preserving the single-file invariant until batch uploads are explicitly designed.

3. **System Settings Extension Visibility & Discoverability**
   - In modern macOS (Ventura, Sonoma, Sequoia), third-party services sometimes remain unchecked in System Settings by default.
   - Should Upit Desktop provide a "Configure Finder Integration" button in its settings that opens `x-apple.systempreferences:com.apple.Keyboard-Settings.extension` or triggers `NSUpdateDynamicServices()`?

---

## Sources & Citations

1. **Apple Developer Documentation - Services Implementation Guide & NSServices**:
   - `https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/SysServices/Articles/properties.html`
   - `https://developer.apple.com/documentation/bundleresources/information-property-list/nsservices`
   - `https://developer.apple.com/documentation/appkit/nsapplication/registerservicesmenusendtypes(_:returntypes:)`
2. **Apple Developer Documentation - Action Extensions**:
   - `https://developer.apple.com/documentation/appkit/add-functionality-to-finder-with-action-extensions`
   - `https://developer.apple.com/documentation/bundleresources/information-property-list/nsextension/nsextensionpointidentifier`
3. **Apple Developer Documentation - Finder Sync Framework**:
   - `https://developer.apple.com/documentation/findersync`
   - `https://developer.apple.com/documentation/findersync/fifindersynccontroller`
4. **Apple Developer Documentation - App Sandbox & Security**:
   - `https://developer.apple.com/documentation/security/app-sandbox`
   - `https://developer.apple.com/documentation/security/accessing-files-from-the-macos-app-sandbox`
5. **Apple Developer Documentation - Notarization & Hardened Runtime**:
   - `https://developer.apple.com/documentation/technotes/tn3147-migrating-to-the-latest-notarization-tool`
   - `https://developer.apple.com/documentation/security/customizing-the-notarization-workflow`
6. **Apple Developer Documentation - User Notifications**:
   - `https://developer.apple.com/documentation/usernotifications/unusernotificationcenter`
7. **Wails v3 Packaging Documentation**:
   - `https://v3.wails.io/learn/packaging/` (Taskfile orchestration, `.app` bundle structure, developer ID signing)
