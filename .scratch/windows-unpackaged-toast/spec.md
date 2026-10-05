# Upit — Windows Unpackaged Start Menu AUMID Toast Notification

Status: ready-for-agent

## Problem Statement

When Upit migrated from the packaged `IExplorerCommand` route to the unpackaged per-user Classic Verb in ADR 0021, Windows Toast delivery was removed because the prior implementation queried MSIX package identity via `Get-AppxPackage` to derive the AUMID.

Users who prefer visual confirmation upon upload completion receive no notification on clean success; while the Final URL is copied to the clipboard, there is no immediate desktop cue that the upload has completed. Windows 11 supports unpackaged Win32 Toast notifications provided a Start Menu shortcut exists with the `System.AppUserModel.ID` property set.

## Solution

Restore native silent Windows 11 Toast notifications for File Manager Upload clean success without requiring MSIX, AppX manifests, or code signing:

1. **AUMID Definition**: Upit defines a canonical static AUMID: `HungNth.Upit`.
2. **Start Menu Shortcut with AUMID**:
   - NSIS installer sets property `System.AppUserModel.ID` (`{9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}, 5`) to `"HungNth.Upit"` on `$SMPROGRAMS\Upit.lnk`.
   - Desktop Repair ensures the shortcut exists and has the canonical AUMID property.
   - Uninstall deletes `$SMPROGRAMS\Upit.lnk`.
3. **Helper Toast Delivery**:
   - Helper (`cmd/upit-file-manager`) sends a silent Toast using `[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('HungNth.Upit')`.
   - Title: `Upload complete`
   - Body: `Final URL copied to clipboard.`
   - Audio: `<audio silent="true"/>`
   - No interactive actions, no activation launch argument, no resident notifier.
   - Unique event tag per toast prevents replacement across sequential clean uploads.
4. **Fallback & Error Isolation**:
   - If Toast delivery fails or is disabled by user/OS Focus Assist, the helper exits cleanly (code 0) without modal fallback or error, preserving the copied URL in the clipboard.
   - Actionable non-clean outcomes (warnings, cancellation, failure, retry) continue to use native interactive surfaces (Task Dialog / Message Box).

## Out of Scope

- Click activation callbacks or resident background notification processes.
- Toast history / notification management inside Upit Desktop.
- Machine-wide Start Menu shortcuts.
