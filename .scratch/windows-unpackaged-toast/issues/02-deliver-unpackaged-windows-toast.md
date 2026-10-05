# 02: Deliver unpackaged Windows 11 Toast on clean success

**What to build:** Restore silent native Windows Toast notification in `cmd/upit-file-manager/feedback_windows.go` using static AUMID `HungNth.Upit` without requiring MSIX package identity or `Get-AppxPackage`.

**Blocked by:** 01: Set Start Menu shortcut AUMID in installer and Desktop Repair.

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise `completeFeedback` and native notification delivery in helper tests with mocked/real WinRT notification subsystem and static AUMID.

**Demo path:** Execute clean File Manager Upload, verify progress dialog closes, Final URL is in clipboard, and native Toast appears with title `Upload complete` and body `Final URL copied to clipboard.`

- [x] Helper defines static AUMID `HungNth.Upit` and calls WinRT `ToastNotificationManager::CreateToastNotifier("HungNth.Upit")`.
- [x] No call to `Get-AppxPackage` or package identity check is made.
- [x] Toast is silent (`<audio silent="true"/>`), title is `Upload complete`, body is `Final URL copied to clipboard.`.
- [x] Toast has unique random event tag to prevent replacement.
- [x] Toast delivery failure or disabled notification setting exits code 0 silently without modal fallback.
- [x] Actionable outcomes (warnings, cancellation, failure, retry) remain interactive.

## Comments
