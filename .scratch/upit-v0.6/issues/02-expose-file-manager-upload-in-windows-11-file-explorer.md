# 02: Expose File Manager Upload in Windows 11 File Explorer

**What to build:** Deliver `Upload with Upit` in the primary Windows 11 x64 File Explorer context menu. The native `IExplorerCommand` adapter accepts only one selected file, hands it to the completed one-shot helper, and supplies privacy-minimized native progress, Cancel, Copy, Retry, and Open Upit Desktop actions without performing upload work inside Explorer or showing Upit Desktop.

**Blocked by:** 01: Deliver one-shot File Manager Upload

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Reuse the completed File Manager Upload application seam for behavior. Limit Windows-native tests to Explorer selection translation, helper launch, notification/action dispatch, and no-window behavior; use a real Windows 11 x64 developer package and local endpoint for the vertical smoke.

**Demo path:** Install a developer package on Windows 11 x64, right-click one file, choose `Upload with Upit` from the primary menu, observe private progress, Cancel one slow upload, complete one success with and without automatic clipboard, trigger a failure then explicit Retry, and confirm Upit Desktop never opens.

- [ ] The primary Windows 11 File Explorer menu presents `Upload with Upit` only for exactly one file selection; folders and multi-selections do not start a helper.
- [ ] The COM adapter contains no configuration loading, upload execution, or response parsing; it passes the untrusted selection to the helper and returns without blocking Explorer on network work.
- [ ] The native adapter launches no Wails window, carries no URL/path in rendered notification text, and dispatches Cancel, Copy Final URL, Retry, and Open Upit Desktop only through the completed helper contract.
- [ ] Notification denial/failure reaches the selected native alert fallback without changing the upload outcome.
- [ ] A Windows 11 x64 local smoke proves exact-one selection, success, warning, failure, cancellation, recovery actions, no Desktop window, helper exit, and unchanged CLI behavior.
