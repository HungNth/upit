# 05: Cut over existing random-layout installations once

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** A user with an existing installation whose recorded Active Payload is a random `.tmp` directory can install the new release directly. The installer validates the exact recorded old state, transactionally publishes the stable CLI and readable version layout, preserves manually managed PATH entries, and removes the obsolete layout from the supported runtime contract after commit.

**Blocked by:** 03: Cut over different product versions with deferred cleanup.

**Status:** ready-for-agent

- [ ] Legacy cutover accepts only the exact prior Active Payload recorded by Upit after canonical validation that it belongs to the requested installation root and contains the required coherent payload.
- [ ] The installer never discovers legacy state by scanning arbitrary `.tmp`, staging, or recovery directories.
- [ ] Cutover stages and verifies the new version directory, stable launcher, installer-owned stable-root PATH entry, Start Menu shortcut, Classic Verb, Active Payload metadata, and uninstall metadata before commit.
- [ ] A failure before commit restores the prior random-layout callable route and does not leave the new public CLI or PATH entry partially published.
- [ ] After commit, new invocations use only the stable launcher and readable Active Payload; the random layout receives no runtime compatibility fallback.
- [ ] The prior random payload is deleted when unlocked or retained only as inactive deferred-cleanup material when still in use.
- [ ] A payload-specific PATH entry added manually by the user is neither removed, rewritten, nor claimed by the installer.
- [ ] Installation and release guidance states that a user-owned legacy PATH entry may shadow the new stable-root entry and must be removed manually.
- [ ] Native Windows smoke starts from a real installed random-layout setup or fixture, not synthetic registry values, and proves the full cutover, user-owned legacy PATH preservation, direct Desktop/helper routes, and old-payload cleanup outcome.

## Implementation evidence

Ticket 05 implemented. Automated native installer smoke exercised migration from a real legacy installation fixture (`upit-windows-x64-0.9.0-setup.exe`):
1. Validated recorded random `.tmp` payload strictly using `windowspayload.ValidateLegacyTmpPayload` without arbitrary directory scanning.
2. Staged and verified the new stable layout at `versions\0.9.6`, published the root `upit.exe` launcher, and updated Classic Verb, Start Menu shortcut, and Product metadata within the transaction.
3. Verified that user-owned legacy PATH entry (simulating a user manually adding `versions\<token>.tmp` to PATH) was preserved untouched, while the stable installation root was added with `PathEntryOwned = 1`.
4. Confirmed that the obsolete `.tmp` payload directory was cleaned up post-commit, with no runtime fallback permitted.
Evidence: `.build/ticket05-smoke-evidence.json`. Operator Explorer observations were not exercised.
