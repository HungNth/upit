# 01: Set Start Menu shortcut AUMID in installer and Desktop Repair

**What to build:** Ensure that the Start Menu shortcut (`Upit.lnk`) is created with the `System.AppUserModel.ID` property set to `HungNth.Upit` during NSIS installation and restored during explicit Desktop Repair.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Inspect `$SMPROGRAMS\Upit.lnk` property store via PowerShell/C# COM interop to verify `PKEY_AppUserModel_ID` matches `HungNth.Upit`. Exercise Desktop Repair in Go to verify shortcut and AUMID are ensured.

**Demo path:** Install Upit via installer, inspect `$SMPROGRAMS\Upit.lnk` property store, verify `System.AppUserModel.ID` = `HungNth.Upit`. Delete the shortcut, trigger Desktop Repair, and verify the shortcut with AUMID is recreated.

- [ ] NSIS installer sets `System.AppUserModel.ID = "HungNth.Upit"` on `$SMPROGRAMS\Upit.lnk`.
- [ ] Desktop Repair checks and recreates/updates `$SMPROGRAMS\Upit.lnk` with the exact AUMID.
- [ ] Uninstall removes `$SMPROGRAMS\Upit.lnk`.
- [ ] Tests verify shortcut creation and property store inspection.

## Comments
