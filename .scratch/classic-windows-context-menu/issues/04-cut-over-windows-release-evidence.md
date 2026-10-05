# 04: Cut over Windows release evidence and documentation

**What to build:** Replace primary-menu, package-identity, signed-registration release claims with evidence for the unsigned Windows 11 x64 Classic Verb product. Prove the installed File Explorer surface, direct File Manager Upload lifecycle, Desktop Repair, transactional update, old-route migration, and fail-closed uninstall, then update current documentation without rewriting historical release records.

**Blocked by:** 03: Package the unsigned Classic Verb product transactionally.

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Use the actual unsigned consumer installer on an interactive Windows 11 x64 runner. Registry inspection supports the proof but does not replace operator-observed **Show more options** placement and invocation. Migration proof starts from an actual registered old package identity rather than a mocked package query.

**Demo path:** Install the unsigned setup on Windows 11 x64, open File Explorer, invoke `Upload with Upit` through **Show more options**, exercise success and non-clean outcomes, damage and Repair registration, update from an old packaged installation, verify one final route, and uninstall with complete cleanup.

- [x] Current product and architecture documentation identifies Windows 11 x64 **Show more options** as the supported File Manager Integration surface and no longer claims primary-menu placement.
- [x] Current packaging documentation states that ordinary unsigned installers are fully functional and that protected Authenticode signing is optional rather than an integration prerequisite.
- [x] Historical v0.6/v0.7 specifications and research remain intact as evidence; the new ADR and specification explicitly supersede their packaged-route decisions.
- [x] Active repository documentation contains no stale requirement for the Explorer COM adapter, sparse MSIX, package identity, MakeAppx, signed Repair material, or Windows clean-success Toast.
- [x] Ordinary Windows CI builds and validates the unsigned consumer artifact using the public package command without release secrets.
- [x] Optional protected-tag CI still verifies Authenticode signatures and protected secret boundaries when signing inputs are configured, without making signing a functional gate.
- [x] Native Windows 11 x64 smoke installs the actual unsigned setup into a current-user location and verifies the canonical Classic Verb registration and installed payload.
- [ ] Operator-observed smoke proves `Upload with Upit` appears under **Show more options** for exactly one file, is absent for multiple files and folders, and invokes the registered command rather than a direct helper shortcut.
- [ ] A real local upload proves privacy-safe progress, cancellation, Final URL clipboard delivery, silent clean success, non-clean warning/failure/recovery feedback, complete helper exit, and no Desktop window.
- [ ] Desktop smoke distinguishes `Registered`, `Needs Repair`, and `Reinstall Upit`; deliberately damaged registration is restored only by explicit Repair and works again through File Explorer.
- [x] Update smoke observes bounded transactional overlap only during cutover, one Classic Verb after commit, the new helper target, no duplicate command, and previous payload removal only after verification.
- [x] Compensation smoke covers Classic Verb failure, legacy cleanup failure with the old package still present, removal-command error with the old package observed absent, unknown package state, and retained recovery payload.
- [ ] Migration smoke starts from an actual registered old Upit package/primary-menu route and proves successful update removes package identity and COM registration before reporting completion with one Classic Verb.
- [x] Uninstall smoke proves registry, obsolete package identity, product metadata, shortcuts, and payload are removed; injected cleanup failure proves payload is retained and uninstall is not reported successful.
- [x] CLI and Manual Upload regressions pass against the same Configuration Set, and macOS build/tests retain their Finder and native-notification contracts.
- [x] Native smoke evidence is structured, redacts private file/configuration/upload values, records artifact version and signing state, and distinguishes functional unsigned proof from optional Authenticode trust proof.

## Comments
