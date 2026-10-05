# 01: Replace Windows File Manager Integration with a per-user Classic Verb

**What to build:** Replace Windows package-identity registration with one current-user Classic Verb named `Upload with Upit`. The command must invoke the private one-shot helper for exactly one selected regular file, remain inspectable and explicitly repairable through Upit Desktop, remove an obsolete Upit package route when found, and never report a duplicate or stale route as `Registered`.

**Blocked by:** None (can start immediately).

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the existing File Manager Integration module as the external seam. Keep registry mechanics behind one internal Windows adapter seam that can use an isolated current-user test root, real registry operations, temporary payload directories, and real filesystem state without mutating the canonical user registration during deterministic tests.

**Demo path:** Build the Windows helper and Desktop, register the integration for a test payload, inspect `Registered`, open File Explorer and find `Upload with Upit` under **Show more options** for one file, damage the verb, observe `Needs Repair`, run explicit Repair, and remove registration.

- [x] Windows registers one per-user verb under the all-files association; it does not create machine-wide, directory, folder-background, or all-filesystem-object entries.
- [x] The verb display name is exactly `Upload with Upit`, the icon resolves from Upit Desktop, `MultiSelectModel` is `Single`, and the command contains the fully qualified private helper path plus one correctly quoted `"%1"` argument.
- [x] File Explorer can invoke the private helper directly for one selected file without opening Upit Desktop or routing through the CLI.
- [x] Multiple selections suppress the verb, folders do not receive it, and the helper remains authoritative for rejecting non-regular, changed, or otherwise invalid paths before network work.
- [x] Passive Desktop inspection performs no registry mutation.
- [x] `Registered` requires every expected verb value, the active payload path, an existing regular helper file, and absence of the obsolete Upit package registration.
- [x] Intact payload with missing/stale Classic Verb values or a remaining obsolete package route produces `Needs Repair`; missing required payload produces `Reinstall Upit`.
- [x] Explicit Repair writes the exact Classic Verb, removes the obsolete package route when present, notifies Explorer of association changes, re-inspects state, and succeeds only when state becomes `Registered`.
- [x] Failed registration or legacy cleanup never produces `Registered`, never suppresses the underlying cause with a success message, and leaves recovery guidance actionable.
- [x] Removal deletes only Upit's exact Classic Verb and obsolete package registration, verifies their absence, and reports failure rather than claiming cleanup when either remains.
- [x] File Manager Upload selection, Configuration Set loading, Uploader/Shortener behavior, always-copy behavior, privacy, cancellation, single-flight, recovery actions, and helper exit remain unchanged.
- [x] Deterministic tests cover exact registry values, quoting, active/stale helper paths, absent/wrong keys, legacy-route presence, explicit-only Repair, removal, post-action reinspection, and every user-visible state without asserting source text or mock call echoes.
