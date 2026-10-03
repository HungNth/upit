# 05: Manage Windows File Manager Integration from Desktop

**What to build:** Give Windows users truthful File Manager Integration status and explicit recovery inside Upit Desktop. Desktop inspects current-user package registration, required payload, repair material, and trust prerequisites; it repairs only when safe and otherwise directs the user to reinstall without claiming live Explorer visibility.

**Blocked by:** 02: Show File Manager Integration as unsupported on Linux.

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the File Manager Integration module through a Windows adapter for package queries, payload inspection, signature/trust verification, current-user registration, Explorer navigation, and post-action reinspection.

**Demo path:** On Windows, remove package registration while keeping valid repair inputs, open Desktop, observe `Needs Repair`, choose Repair, and verify the state becomes `Registered`. Repeat with missing or untrusted repair material and observe `Reinstall Upit` without registration changes.

- [ ] Passive inspection checks current-user package registration and required adapter/worker payload without modifying registration.
- [ ] Intact payload with healthy package registration produces `Registered` without claiming live Explorer menu visibility.
- [ ] Intact payload with absent or stale registration and trustworthy repair inputs produces `Needs Repair` and exposes `Repair Integration`.
- [ ] Missing payload, missing registration material, unsigned material, untrusted signatures, or mismatched external payload produces `Reinstall Upit` with actionable guidance.
- [ ] Repair runs only after explicit user action, operates in the current-user context, and re-inspects state before reporting success.
- [ ] Failed registration never produces `Registered` and never suppresses the underlying cause with a success message.
- [ ] Unsigned verification builds expose detection and negative-path behavior but do not present production Repair as available.
- [ ] Explorer verification guidance opens File Explorer and explains the right-click path without invoking the private worker directly.
- [ ] The File Manager Integration area remains usable during Configuration Set Setup or Repair and does not block Manual Upload.
- [ ] Deterministic tests cover package/payload/trust combinations, explicit-only Repair, fail-closed behavior, action availability, and post-repair state; narrow Windows-native checks cover real package queries where available.
