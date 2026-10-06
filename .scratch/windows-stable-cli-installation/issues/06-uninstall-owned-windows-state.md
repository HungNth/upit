# 06: Uninstall every installer-owned route and payload safely

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** Removing Upit unregisters File Manager Integration before deleting callable code, removes every launcher, payload, shortcut, metadata record, and PATH entry that the installer owns, and preserves environment entries created by the user. Cleanup failure never leaves a context-menu route pointing to missing code.

**Blocked by:** 03: Cut over different product versions with deferred cleanup; 04: Replace the same version and launcher format safely; 05: Cut over existing random-layout installations once.

**Status:** ready-for-agent

- [ ] Uninstall removes and verifies File Manager Integration before deleting the Active Payload or stable launcher.
- [ ] Registration cleanup failure aborts removal and retains callable payload and product metadata rather than leaving a dangling Classic Verb.
- [ ] Successful uninstall removes the stable launcher, Active Payload, inactive deferred-cleanup payloads, staging and recovery directories, Start Menu shortcut, uninstall metadata, and product-owned metadata.
- [ ] Uninstall removes one normalized-equivalent stable-root PATH entry only when product metadata proves Upit inserted it.
- [ ] A stable-root PATH entry that existed before installation is preserved.
- [ ] A user-owned legacy payload-specific PATH entry is preserved even when it points to an obsolete or removed payload.
- [ ] If the installer-owned PATH entry was removed or changed by the user, uninstall does not remove a substitute or unrelated entry.
- [ ] Remaining PATH entry text and ordering are preserved, and a successful PATH mutation broadcasts the Windows environment-change notification.
- [ ] Successful uninstall leaves no installer-owned command or route resolving to deleted files.
- [ ] Native Windows smoke proves both installer-owned and pre-existing stable-root PATH cases, deferred-payload cleanup, fail-closed registration removal, and final owned-state removal using the real uninstaller.

## Implementation evidence

Ticket 06 implemented. Automated native installer smoke exercised complete uninstall:
1. Fail-closed uninstallation: injected registry ACL permission denial into Classic Verb key before uninstallation; uninstaller failed closed with nonzero exit code, retaining the full callable Active Payload, stable root launcher, and product metadata key (`uninstallFailureRetainsPayload: true`).
2. Clean uninstallation: after ACL restore, uninstaller successfully removed File Manager Integration, stable launcher (`upit.exe`), Start Menu shortcut, active version directory, and installer-owned User PATH entry while broadcasting environment changes (`uninstallClean: true`, `uninstallRegistrationClean: true`).
3. Environment preservation: verified that User PATH was restored to the exact initial state before installation (`ownedPathRemovedOrUserPathPreserved: true`), with unrelated entries and text preserved.
Evidence: `.build/ticket06-smoke-evidence.json`. Operator Explorer observations were not exercised.
