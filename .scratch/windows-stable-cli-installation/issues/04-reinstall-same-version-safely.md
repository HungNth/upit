# 04: Replace the same version and launcher format safely

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** Reinstalling the current product version replaces the single final version directory without file-by-file overwrite. The installer asks the user to close processes that actually execute from the Active Payload, preserves the old directory as recovery material during replacement, and restores it on any pre-commit failure. The same lifecycle safely replaces the stable launcher only when its internal format requires an update.

**Blocked by:** 02: Install the public CLI through a stable Windows launcher.

**Status:** ready-for-agent

- [ ] Same-version replacement stages and verifies a complete payload before the current final version directory changes.
- [ ] Blocking process detection uses executable image paths and covers the real CLI, Upit Desktop, private File Manager helper, and the root launcher when its format must change.
- [ ] Interactive installation offers Retry and Cancel and never force-terminates an Upit process.
- [ ] Silent installation aborts nonzero without changing active state when required files are in use.
- [ ] Process detection is advisory; every rename, publication, verification, rollback, and cleanup result is checked so an unreported antivirus or filesystem lock cannot produce a partial installation.
- [ ] The current version directory is moved to recovery storage before the staged replacement takes the same final `versions\X.Y.Z` path.
- [ ] Successful same-version replacement retains the same Active Payload path and proves replacement through coherent contents and transaction evidence rather than a new random directory.
- [ ] Any pre-commit failure restores the prior version directory, launcher, Active Payload metadata, direct routes, shortcut, uninstall metadata, and PATH state changed by the transaction.
- [ ] A rollback that cannot be proven reports failure and retains all available recovery material without claiming a clean active state.
- [ ] The stable launcher has an internal technical format identifier, is preserved during ordinary product updates, and is transactionally replaced only when the required format changes.
- [ ] Native Windows smoke proves silent process-block abort, successful replacement after process closure, representative mid-transaction rollback, unresolved rollback retention, and launcher-format replacement from a real prior fixture.

## Implementation evidence

Ticket 04 implemented. Automated native installer smoke exercised same-version reinstall and launcher-format replacement:
1. Active payload process detection: when a CLI process is running from the active payload, silent same-version install aborts nonzero without altering active routes, version directory, or metadata (`silentSameVersionAbortsWhenInUse: true`).
2. Process closure recovery swap: after the active process exits, same-version reinstall swaps the active version directory into temporary recovery storage, moves staged replacement to the final path, verifies routes, removes the recovery directory, and retains the exact same `versions\0.9.4` payload path per ADR 0023 (`sameVersionUpdate: true`).
3. Launcher-format replacement: tested against a real format-0 setup fixture (`upit-windows-x64-0.9.5-setup.exe` compiled with `-LauncherFormat 0`); subsequent update by format-1 setup verified that `upit.exe --launcher-format` was updated from `0` to `1` (`launcherFormatReplacementVerified: true`).
Evidence: `.build/ticket04-smoke-evidence.json`. Operator Explorer observations were not exercised.
