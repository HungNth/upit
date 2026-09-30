# 01: Expose File Manager Upload through Finder Services

**What to build:** Deliver `Upload with Upit` through Finder Services or Quick Actions on macOS 14+ Apple Silicon. A background-only `NSServices` helper receives one Finder file URL, delegates to the completed File Manager Upload behavior, supplies native progress and recovery actions, and exits without a Wails window, Dock icon, tray process, or resident worker.

**Blocked by:** v0.6 / 03: Release signed Windows File Manager Upload

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Reuse the completed File Manager Upload application seam for all upload behavior. Keep macOS-native tests limited to pasteboard URL translation, exact-one selection, background helper lifecycle, native notification/action dispatch, and Service registration; use a real macOS 14+ Apple Silicon local smoke for the vertical path.

**Demo path:** Install a local macOS package, select one file in Finder, choose `Upload with Upit` from Services or Quick Actions, observe private progress, cancel a slow upload, complete success/warning/failure paths with Copy/Retry/Open Desktop recovery, and verify no Upit Desktop window or Dock icon appears.

- [ ] The background-only NSServices helper accepts one Finder-selected file URL and rejects zero, multiple, directory, and non-regular selections before network activity.
- [ ] The helper does not launch Wails, parse CLI output, duplicate Configuration Set/upload behavior, display a Dock icon, or survive after the operation or action window.
- [ ] The Finder adapter renders no path, file name, URL, endpoint, request value, response data, or credential; it reuses the shared progress, Cancel, Copy, Retry, Open Upit Desktop, and native alert contracts.
- [ ] The macOS package remains outside App Sandbox and keeps the fixed Configuration Set location intact; Finder Sync, Share, and Action Extensions are not introduced.
- [ ] A macOS 14+ Apple Silicon local smoke proves Services or Quick Actions discovery, selection validation, direct upload lifecycle, privacy, recovery, no-window behavior, helper exit, and unchanged CLI behavior.
