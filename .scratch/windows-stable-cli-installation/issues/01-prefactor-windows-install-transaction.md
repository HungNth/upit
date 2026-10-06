# 01: Prefactor Windows installation into one recoverable transaction seam

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** Preserve the current Windows installation experience while giving install, update, and uninstall one private transaction boundary that can observe the previous callable state, apply mutations, verify the result, and compensate on failure. This prefactor makes later launcher, PATH, payload-layout, and migration work additive instead of extending independent installer and File Manager Integration transactions.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Existing fresh install, same-version update, File Manager Integration, Desktop shortcut, and uninstall behavior remain externally unchanged.
- [ ] One authoritative private lifecycle seam represents the previously observed Active Payload, product metadata, Classic Verb, Start Menu shortcut, uninstall metadata, and owned filesystem state needed for compensation.
- [ ] File Manager Integration remains a private direct-helper route; no public launcher, PATH entry, or new user-facing application is introduced by this ticket.
- [ ] A pre-commit mutation failure restores the previously observed callable route and does not report installation success.
- [ ] An unresolved compensation failure is explicit and retains the payload required for recovery.
- [ ] Passive Desktop inspection and explicit Repair continue to use Active Payload metadata rather than the executable path of a stale Desktop process.
- [ ] The existing native Windows installer smoke remains the acceptance seam and passes without weakening its current lifecycle assertions.
- [ ] Existing Windows registry compensation tests remain green and continue to prove exact Classic Verb restoration.

## Implementation evidence

Ticket 01 implemented. Native automated installer smoke passed with real 0.9.1 setup: fresh install, local helper and CLI uploads, locked-uninstaller rollback with exact registry/file/PATH state equality, same-version random-layout update, fail-closed registry-deletion uninstall, and clean uninstall. Existing Windows integration tests and private-worker Go vet passed. Evidence: `.build/ticket01-smoke-evidence.json`. Existing user installation was isolated and restored; Explorer operator observations were not exercised. Later tickets retain the separate explicit unresolved-rollback acceptance case.
