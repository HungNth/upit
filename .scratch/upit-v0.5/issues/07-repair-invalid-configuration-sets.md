# 07: Repair invalid Configuration Sets

**What to build:** Turn partial, malformed, unsupported, permission-invalid, or semantically invalid Configuration Sets into a safe Repair workflow. Users receive authoritative diagnostics, can explicitly unlock and repair the original raw document, and return to structured editing only after valid publication. No existing document is reset, silently normalized, or overwritten while invalid.

**Blocked by:** 01: Launch the read-only Upit Desktop; 02: Edit Global Configuration safely

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Classify and repair Configuration Set states through the Wails-free application module using real malformed and invalid temporary documents. Use focused desktop tests/smoke for unlock, lock, raw rendering, and the visible disabled-upload state. Do not parse or validate raw JSON in Vue as a second authority.

**Demo path:** Launch Desktop against a malformed credential-bearing Uploader document, observe Repair and disabled Manual Upload, explicitly unlock the raw document, correct it, Save only after strict validation succeeds, observe relock and normal structured editor state, then run `upit config validate`.

- [x] Valid, completely absent, and every partial/invalid Configuration Set state select normal, Setup, or Repair deterministically without changing existing document bytes during classification.
- [x] Repair identifies missing required documents, malformed JSON with line/column, unsupported versions, permission failures, strict semantic failures, and cross-document reference failures with existing redaction.
- [x] Manual Upload remains unavailable throughout Setup and Repair.
- [x] Raw Repair is locked by default and requires explicit user confirmation warning that the document may reveal credentials.
- [x] Unlock exposes the complete original document only for the current session; leaving Repair or closing the desktop application relocks it and persists neither raw contents nor reveal state.
- [x] Raw Repair delegates strict syntax, document, and cross-document validation to Go and cannot Save invalid content.
- [x] Successful repair uses the safe canonical publication path and returns to normal structured editing only when the complete Configuration Set is valid.
- [x] Tests prove Repair never silently resets, truncates, writes draft sidecars, logs credentials, or enables an upload from invalid configuration.

**Earlier verification note:** Startup tests covered partial/missing, malformed line/column, unsupported-version, Unix permission, cross-document, stale, publication/reclassification, and CLI refusal. Frontend cleanup covered successful repair, kind changes, and unmount; the later actual native session evidence is recorded below.

## Comments

- 2026-10-03: actual macOS Wails WebView opened a malformed temporary Uploader document in locked Repair with Manual Upload disabled. Explicit Unlock exposed exactly the original bytes; changing document kind removed raw contents and relocked. Closing while unlocked exited 0, left original bytes unchanged, and created no sidecar/reveal files. A fresh native process reopened locked with no raw textarea. Validated Save canonically published the corrected document, left Repair, and removed raw state. Full race-enabled Go suite and vet passed.
