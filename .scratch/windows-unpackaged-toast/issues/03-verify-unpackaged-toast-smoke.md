# 03: Verify end-to-end unpackaged Toast in packaging and smoke gates

**What to build:** Update `packaging/windows/native-smoke.ps1` and verification workflows to assert that the Start Menu shortcut has `System.AppUserModel.ID` set and that clean upload emits a native Toast with AUMID `HungNth.Upit`.

**Blocked by:** 02: Deliver unpackaged Windows 11 Toast on clean success.

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Execute `native-smoke.ps1 -AutomatedOnly` to verify shortcut property and WinRT notifier status; execute interactive smoke to observe actual Toast banner.

**Demo path:** Build package, install, verify shortcut AUMID via PowerShell, run smoke upload, observe Toast.

- [x] Automated smoke verifies `$SMPROGRAMS\Upit.lnk` exists in Start Menu Programs.
- [x] Automated smoke verifies `ToastNotificationManager::CreateToastNotifier("HungNth.Upit")` returns `Enabled`.
- [x] Native smoke interactive checklist prompts operator to observe Toast banner on clean success.
- [x] Release documentation is updated to reflect Toast availability via unpackaged Start Menu AUMID.

## Comments
