# 03: Manage macOS File Manager Integration from Desktop

**What to build:** Give macOS users truthful File Manager Integration status and explicit recovery inside Upit Desktop. Desktop inspects the installed bundles and Launch Services registration without mutating startup state, offers Repair when safe, guides the user to Finder and Keyboard Settings, and prepares deterministic Service cleanup before removal.

**Blocked by:** 02: Show File Manager Integration as unsupported on Linux.

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the File Manager Integration module through a macOS adapter for bundle inspection, Launch Services discovery, explicit registration, settings navigation, Finder navigation, and removal preparation.

**Demo path:** On macOS, remove only the Service registration while keeping the installed bundle intact, open Desktop, observe `Needs Repair`, choose Repair, then follow Finder verification guidance. Finally choose `Prepare to Remove Upit` and observe registration cleanup before removal instructions.

- [ ] Passive inspection verifies expected nested bundles and discoverable Launch Services registration without changing either.
- [ ] Intact bundles with discoverable registration produce `Registered` without claiming Services enablement or live Finder visibility.
- [ ] Intact bundles with absent or stale registration produce `Needs Repair` and expose explicit `Repair Integration`.
- [ ] Missing or damaged required bundles produce `Reinstall Upit`; Repair cannot recreate payload.
- [ ] Repair re-registers the installed outer application and nested Service, refreshes dynamic services, and re-inspects the resulting state.
- [ ] Desktop never reads or writes private Services preference data and never shows an enable/disable toggle.
- [ ] `Open Keyboard Settings` uses best-effort navigation and always includes the manual path to Services → Files and Folders.
- [ ] Finder verification guidance opens Finder but does not invoke the private upload worker as a fake integration test.
- [ ] `Prepare to Remove Upit` requires confirmation, unregisters the nested Service and outer application, refreshes dynamic services, exits Desktop, and directs the user to remove the application bundle.
- [ ] The File Manager Integration area remains usable during Configuration Set Setup or Repair and does not block Manual Upload.
- [ ] Deterministic tests cover state classification, explicit-only Repair, settings fallback, reinspection, and unregister-before-removal ordering; narrow native checks cover real Launch Services interactions where available.
