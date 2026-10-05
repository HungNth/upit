# Use Start Menu AUMID shortcut for unpackaged Windows Toast notification

Status: accepted

Upit on Windows 11 x64 delivers native silent clean-success Toast notifications by assigning a static Application User Model ID (`HungNth.Upit`) to the installed Start Menu shortcut (`%APPDATA%\Microsoft\Windows\Start Menu\Programs\Upit.lnk`), with property `System.AppUserModel.ID` (`PKEY_AppUserModel_ID`). When File Manager Upload completes cleanly without warnings or recovery actions, the one-shot helper copies the Final URL to the clipboard, closes active progress, and emits one silent native Toast notification via `Windows.UI.Notifications.ToastNotificationManager::CreateToastNotifier("HungNth.Upit")`.

This modifies the Windows notification portion of ADR 0021 while preserving all of its Classic Verb, packaging, and lifecycle decisions:

1. **No MSIX or Package Identity required**: Toast delivery targets the unpackaged static AUMID bound to the per-user Start Menu shortcut, without sparse MSIX, AppX manifest, or trusted code-signing prerequisites.
2. **Purely informational silent Toast**: The notification payload contains `<audio silent="true"/>`, title `Upload complete`, and body `Final URL copied to clipboard.`, with no interactive action buttons or background activation requirements. No out-of-proc COM activation server (`INotificationActivationCallback`) is required.
3. **Fail-closed & non-blocking fallback**: If the shortcut is absent, notifications are disabled, or the WinRT notification API fails, the helper exits cleanly with code 0 without converting clean success to a warning, failure, or modal popup, because the Final URL is already safely preserved in the clipboard.
4. **Installer and uninstaller contract**:
   - NSIS installation injects `System.AppUserModel.ID = "HungNth.Upit"` into `$SMPROGRAMS\Upit.lnk`.
   - Uninstall removes `$SMPROGRAMS\Upit.lnk` alongside existing cleanup.
   - Desktop Repair verifies or restores the shortcut AUMID property.
