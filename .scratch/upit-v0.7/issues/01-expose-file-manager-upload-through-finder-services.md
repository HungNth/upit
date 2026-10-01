# 01: Expose File Manager Upload through Finder Services

**What to build:** Deliver `Upload with Upit` through Finder Services or Quick Actions on macOS 14+ Apple Silicon. A background-only `NSServices` helper receives one Finder file URL, delegates to the completed File Manager Upload behavior, supplies native progress and recovery actions, and exits without a Wails window, Dock icon, tray process, or resident worker.

**Blocked by:** v0.6 / 03: Release signed Windows File Manager Upload

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Reuse the completed File Manager Upload application seam for all upload behavior. Keep macOS-native tests limited to pasteboard URL translation, exact-one selection, background helper lifecycle, native notification/action dispatch, and Service registration; use a real macOS 14+ Apple Silicon local smoke for the vertical path.

**Demo path:** Install a local macOS package, select one file in Finder, choose `Upload with Upit` from Services or Quick Actions, observe private progress, cancel a slow upload, complete success/warning/failure paths with Copy/Retry/Open Desktop recovery, and verify no Upit Desktop window or Dock icon appears.

- [x] The background-only NSServices helper accepts one Finder-selected file URL and rejects zero, multiple, directory, and non-regular selections before network activity.
- [x] The helper does not launch Wails, parse CLI output, duplicate Configuration Set/upload behavior, display a Dock icon, or survive after the operation or action window.
- [x] The Finder adapter renders no path, file name, URL, endpoint, request value, response data, or credential; it reuses the shared progress, Cancel, Copy, Retry, Open Upit Desktop, and native alert contracts.
- [x] The macOS package remains outside App Sandbox and keeps the fixed Configuration Set location intact; Finder Sync, Share, and Action Extensions are not introduced.
- [ ] A macOS 14+ Apple Silicon local smoke proves Services or Quick Actions discovery, selection validation, direct upload lifecycle, privacy, recovery, no-window behavior, helper exit, and unchanged CLI behavior.

## Comments

- Native Service adapter: `native/macos/finder-service/main.m` and `selection.m`; it passes `--file-url` to the Wails-free helper and discards stdout/stderr.
- Shared Go boundary: `internal/finder` validates local file URLs; `cmd/upit-file-manager` retains all Configuration Set, upload, action-token, cancellation, and feedback policy.
- Darwin feedback uses an Objective-C `UserNotifications`/`AppKit` bridge in `feedback_darwin.m`; notification denial falls back to a modeless main-thread AppKit progress panel with Cancel, and terminal actions use main-thread native alerts.
- The real macOS 14+ Apple Silicon Finder smoke remains pending because this development host is Windows; `packaging/macos/native-smoke.sh` is the protected runner path.
