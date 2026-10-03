# 04: Deliver one-product macOS installation and removal

**What to build:** Deliver macOS as one user-visible Upit application in a drag-to-install DMG. Desktop, the Finder Service provider, and the one-shot worker ship together under one product identity; consumer installation uses no shell script, updates do not duplicate registration, and removal uses the Desktop cleanup flow.

**Blocked by:** 03: Manage macOS File Manager Integration from Desktop.

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Verify package structure and metadata mechanically, then exercise the installed application through Desktop status/Repair and Finder Services on a real macOS host without duplicating upload behavior tests.

**Demo path:** Build the macOS artifact, drag `Upit.app` into Applications, open Desktop, inspect File Manager Integration, invoke `Upload with Upit` from Finder, update the installed app without a duplicate action, then prepare removal and confirm the Service disappears.

- [ ] The DMG contains one outer application named `Upit` and the normal Applications shortcut expected by drag installation.
- [ ] The Finder Service provider and one-shot worker remain private nested package components with no independent Dock or application entry.
- [ ] User-visible package, notification, and Service-provider names resolve to `Upit`; the Finder action remains `Upload with Upit`.
- [ ] Consumer documentation uses drag-to-Applications installation and does not require the installation shell script.
- [ ] First Desktop launch performs the non-mutating inspection defined by the macOS management slice and offers Repair rather than silently registering.
- [ ] Installing or updating the application does not create duplicate Service registrations or separate product identities.
- [ ] The explicit removal flow unregisters the nested Service and outer application before the user removes the bundle.
- [ ] Unsigned verification builds validate bundle layout, minimum platform, metadata, helper privacy, action naming, and install/update/removal scripts without claiming production trust.
- [ ] Existing CLI and Manual Upload behavior remains unchanged against the same Configuration Set.
- [ ] A local macOS smoke proves drag installation, Desktop inspection, explicit Repair, Finder invocation, update continuity, and deterministic cleanup; protected signing/notarization proof remains ticket 09.
