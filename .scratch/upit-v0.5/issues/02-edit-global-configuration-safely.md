# 02: Edit Global Configuration safely

**What to build:** Make the Global Configuration desktop area useful end to end. Users can inspect and explicitly save the existing default Uploader, optional default Shortener, and clipboard choices through one authoritative validation and safe-publication path. The slice also establishes dirty-state handling and external-change protection that later Configuration Set editors reuse.

**Blocked by:** 01: Launch the read-only Upit Desktop

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise Global Configuration sessions through the Wails-free application module using temporary homes and real documents. Use focused platform filesystem tests for native replacement invariants. The desktop smoke proves the rendered controls, dirty prompt, Save, Refresh, and stale-save feedback rather than re-testing validation in Vue.

**Demo path:** Open a valid Configuration Set, change all three Global Configuration choices, save canonical bytes, run the CLI to observe the new defaults, externally change the file, and observe stale Save rejected without overwriting either disk or the in-memory draft.

- [x] The Global Configuration area exposes only default Uploader, optional default Shortener, and clipboard-copying state, with exact named-definition references and no new settings.
- [x] Edits are dirty in memory until explicit Save; navigation, Refresh, and close offer Save, Discard, and Cancel rather than silently persisting or discarding changes.
- [x] Inline diagnostics and Save use the Go-authoritative strict and cross-document validation path; invalid Global Configuration cannot publish.
- [x] Successful Save produces canonical deterministic JSON and uses safe native staged replacement without backup, lock, or remove-before-replace behavior.
- [x] Mutations refuse linked targets, preserve documented permission behavior, and retain the existing Windows ambiguous-publication recovery contract.
- [x] Snapshot comparison detects external changes on focus, explicit Refresh, and immediately before Save; stale Save is rejected without automatic merge or draft loss.
- [x] Clean Refresh adopts current on-disk state; dirty external changes become an explicit conflict rather than an automatic reload.
- [x] The UI smoke changes each field, resolves a dirty prompt, saves, observes persistent state, and demonstrates stale-save rejection.

**Earlier verification note (superseded by the native evidence below):** Wails-free tests covered canonical Save, stale-save rejection, invalid-draft no-write behavior, and close/focus adapter seams. Native Windows startup with a valid Configuration Set passed; native interaction was not automated in that earlier run.

## Comments

- 2026-10-03: actual macOS Wails WebView smoke via the pinned dependency's MCP adapter changed both default names and clipboard state with input/change events, saved through the dirty-navigation dialog, asserted canonical on-disk values, and rejected a stale Save without changing externally modified bytes. Full race-enabled Go suite, vet, frontend typecheck/production build, and native desktop build passed. The temporary HOME isolated all writes from real user configuration.
