# 04: Complete the Uploader lifecycle

**What to build:** Complete safe Uploader identity and deletion behavior. Users can explicitly rename or confirm deletion of an unreferenced Uploader while Upit prevents name collisions, dangling Global Configuration references, and an empty required Uploader document. The completed slice makes Uploader management fully usable without inventing cross-document transactions.

**Blocked by:** 03: Create and edit Uploaders

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise rename and delete commands through the Wails-free application module against real multi-Uploader Configuration Sets. Test only Wails-specific confirmation/focus behavior at the desktop edge. Preserve the CLI/config validation seam as the independent consumer of published state.

**Demo path:** Create two Uploaders, rename and delete the unreferenced one with confirmation, select an Uploader as the Global Configuration default, observe rename/delete blocked, change the default, then complete the pending lifecycle action and validate through the CLI.

- [x] Rename is a separate explicit action that accepts only a valid, unused exact name and canonically publishes one valid Uploader document.
- [x] A Global Configuration-referenced Uploader cannot be renamed; the user must first choose another default Uploader.
- [x] Delete requires explicit confirmation and never silently removes a definition.
- [x] A Global Configuration-referenced Uploader cannot be deleted.
- [x] The final Uploader cannot be deleted, preserving the non-empty required Uploader document contract.
- [x] After deletion, the editor selects an adjacent remaining Uploader or presents a defined empty selection without showing removed data.
- [x] Rename/delete detect stale documents and preserve both disk and draft on conflict; no lock, backup, auto-merge, or cross-document journal is introduced.
- [x] End-to-end tests prove every successful lifecycle action leaves a Configuration Set accepted by `upit config validate` and the current default upload flow.

**Earlier verification note:** Lifecycle tests covered successful unreferenced rename/delete, adjacent selection, `ValidateConfiguration`, invalid/duplicate names, stale rename rejection, default guards, and final-Uploader rejection. Native interaction evidence and the discovered dialog fix are recorded below.

## Comments

- 2026-10-03: fixed the native WebView lifecycle blocker: macOS `window.prompt()` returned null and `window.confirm()` returned false without usable dialogs. In-app Vue dialogs now execute real Go Rename/Delete publication, preserve exact names, retain errors, close after success, and manage initial/restored focus and keyboard containment. `scripts/native-dialog-smoke.mjs` passed against an actual rebuilt native Wails WebView, including whitespace rejection, successful rename, confirmed deletion, and focus assertions. Persisted documents then passed actual `bin/upit config validate` and a selected-default local HTTP upload. Full race-enabled Go suite and vet passed.
