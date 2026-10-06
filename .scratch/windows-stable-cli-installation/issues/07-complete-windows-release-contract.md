# 07: Make the stable CLI lifecycle the Windows release contract

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** The Windows release gate and product documentation treat the stable public CLI, readable Active Payload, transactional update modes, PATH ownership, legacy cutover, and uninstall lifecycle as the authoritative shipped contract. Maintainers and users receive one consistent description, and releases cannot pass on source inspection or outdated random-payload assertions.

**Blocked by:** 06: Uninstall every installer-owned route and payload safely.

**Status:** ready-for-agent

- [ ] The authoritative native Windows 11 x64 smoke runs the real installer and uninstaller and covers fresh install, stable command resolution in a new process, launcher forwarding, Active Payload validation, PATH ownership, different-version cutover, same-version replacement, launcher-format replacement, rollback, deferred cleanup, real legacy-layout cutover, and uninstall.
- [ ] Native-smoke evidence distinguishes clean commit, committed update with deferred cleanup, silent process-block abort, pre-commit failure with successful rollback, and unresolved rollback with retained recovery material.
- [ ] Same-version acceptance retains the final `versions\X.Y.Z` Active Payload path and no longer requires every update to change `PayloadPath`.
- [ ] Different-version acceptance permits a locked inactive payload after verified commit and no longer requires immediate prior-directory deletion in that distinct deferred-cleanup outcome.
- [ ] The smoke proves installer-added stable PATH removal, pre-existing stable PATH preservation, and user-owned legacy payload PATH preservation with manual-removal guidance.
- [ ] Legacy acceptance begins from a real installed random-layout artifact or fixture rather than fabricated product metadata.
- [ ] Existing File Manager Integration registry tests remain supporting regression evidence; they do not replace the native installer acceptance seam.
- [ ] Package and release checks prove one public `X.Y.Z` version, Windows x64 payload coherence, required installer contents, checksum metadata, unsigned functional installation, and optional protected signing of the stable launcher and payload binaries.
- [ ] Project and Windows installation documentation describe the packaged Windows CLI as public through the stable launcher and retain the private one-shot File Manager helper boundary.
- [ ] Build/package and current Windows integration specifications replace private-CLI, no-PATH, random active `.tmp`, and random same-version payload assumptions with the accepted ADR 0023 contract.
- [ ] Documentation clearly tells users with manually added random payload PATH entries to remove those entries themselves because they may shadow the stable command.
- [ ] Notification and Toast behavior, upload semantics, Configuration Set behavior, and File Explorer menu placement remain unchanged and are not silently resolved by this ticket.

## Implementation evidence

Ticket 07 implemented. Comprehensive automated native installer smoke executed on Windows 11 x64 covering the complete release contract against real installers (0.9.0 baseline, 0.9.2, 0.9.4, 0.9.5 format-0 fixture, and 1.0.0 release candidate):
1. Fresh installation: verified readable `versions\1.0.0` layout, version-aligned AMD64 binaries with embedded PE version resources, stable root launcher (`upit.exe`), Start Menu shortcut with AUMID, Classic Verb registration, and User PATH addition (`PathEntryOwned = 1`).
2. Stable CLI invocation: verified command-name invocation in new process without absolute paths, UTF-8/relative Unicode arguments, nonzero status & stderr preservation, and rejection of invalid/absent/out-of-root Active Payload metadata without version scanning.
3. Console cancellation: verified real in-flight console cancellation without leaving orphan child processes.
4. Different-version update & deferred cleanup: side-by-side deployment with in-flight process surviving on original payload, locked payload recorded under `InactivePayloads` with exit code 10, and subsequent cleanup by later version.
5. Same-version replacement & launcher format: running process causes silent abort nonzero without state mutation; after closure, swaps active version via recovery storage retaining exact `versions\1.0.0` path; launcher format 0 upgraded to format 1 upon update.
6. Real legacy cutover: migrated from real 0.9.0 random `.tmp` setup, preserved user-owned legacy PATH entry while adding stable root, deleted obsolete `.tmp` payload post-commit with no runtime fallback.
7. Fail-closed & clean uninstallation: ACL permission denial proves fail-closed retention; clean uninstall removes launcher, shortcut, versions directory, and installer-owned PATH entry while preserving initial environment.
Evidence: `.build/ticket07-smoke-evidence.json`. All 22 automated lifecycle checks passed. Operator Explorer observations were not exercised.
