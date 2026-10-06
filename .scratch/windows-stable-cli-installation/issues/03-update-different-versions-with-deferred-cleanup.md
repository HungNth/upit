# 03: Cut over different product versions with deferred cleanup

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** Updating from one product version to another stages and verifies the new Active Payload beside the old one, allows already-running processes to finish from the old payload, and sends every new invocation to the verified new payload. A locked old payload becomes explicit inactive cleanup material rather than causing a committed route to roll back.

**Blocked by:** 02: Install the public CLI through a stable Windows launcher.

**Status:** ready-for-agent

- [ ] A different-version update stages a complete version-aligned payload before changing any public route.
- [ ] Upit Desktop, the real CLI, or the private helper may remain running from the former payload without being force-terminated or blocking the side-by-side cutover.
- [ ] Active Payload metadata, the stable launcher, Start Menu shortcut, and Classic Verb serve the new version only after deterministic route verification succeeds.
- [ ] Any failure before commit restores the prior Active Payload, direct routes, shortcut, uninstall metadata, launcher state, and PATH state changed by the transaction.
- [ ] After commit, an unlocked former payload is removed.
- [ ] After commit, a locked former payload is retained as inactive deferred-cleanup material; the new route remains active and the installer reports the distinct deferred-cleanup outcome rather than rolling back.
- [ ] The stable launcher, Desktop Repair, and File Manager Integration never select an inactive retained payload as an alternate route or automatic runtime fallback.
- [ ] A later installation retries deferred cleanup after the old process exits without changing the current Active Payload.
- [ ] Native Windows smoke uses two real installer artifacts with different product versions, keeps a process running from the first payload, proves that process survives, and proves new terminal and File Manager invocations use the second payload.
- [ ] Native-smoke evidence distinguishes clean committed update, committed update with deferred cleanup, and pre-commit failure with successful compensation.

## Implementation evidence

Ticket 03 implemented. Automated native installer smoke exercised real setup updates across versions 0.9.2, 0.9.3, and 0.9.4: staged new payload side-by-side at `versions\0.9.3`, kept the previous 0.9.2 launcher and long-running CLI in-flight process alive without termination, committed routes to the new version, recorded the locked inactive payload in `InactivePayloads` under `HKCU\Software\Upit`, and reported exit code 10. A subsequent installation of 0.9.4 retried deferred cleanup after old process closure, purged the unlocked inactive 0.9.2 payload from disk and registry, and retained the selected new Active Payload. Pre-commit compensation captures the observed legacy package presence and retains all recovery material when package state changes or is unknown. Evidence: `.build/ticket03-smoke-evidence.json`. Operator Explorer observations were not exercised.
