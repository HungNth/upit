# 08: Perform one Manual Upload

**What to build:** Make Manual Upload work end to end for one regular file. A user selects one file with the native picker or drops one file, chooses the same one-upload overrides available through the CLI, receives the production outcome in the desktop, and can explicitly retry a failure without Upit ever retrying automatically.

**Blocked by:** 01: Launch the read-only Upit Desktop

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Exercise selection, freshness, configuration reload/validation, one upload, Shortener processing, clipboard behavior, outcomes, and Retry through the Wails-free application module with temporary homes and real local HTTP servers. Test Wails file-dialog/drop translation only at its adapter seam. A running desktop smoke proves native picker/drop and visible results.

**Demo path:** Select one temporary file, upload it through a local endpoint with default settings, override Uploader/Shortener/clipboard/timeout for another upload, Copy the Final URL, force a structured endpoint failure, and use explicit Retry after correcting the cause.

- [ ] Manual Upload accepts one selected regular file from the native picker or exactly one dropped file; directories, non-regular paths, and any multi-file drop are rejected with an inline explanation and no endpoint request.
- [ ] The view preselects Global Configuration defaults and permits per-upload Uploader, Shortener, no-shorten, clipboard, no-clipboard, and optional whole-operation timeout overrides with existing CLI meanings.
- [ ] Upload cannot begin during Setup, Repair, dirty Configuration Set edits, or an existing active upload.
- [ ] Before start, the application rechecks freshness, reloads, and strictly validates the on-disk Configuration Set; the operation then uses an immutable configuration snapshot.
- [ ] Manual Upload reuses every existing Uploader Request Body Mode, Response Extractor, Shortener fallback, clipboard-warning, timeout, cancellation, and redaction contract rather than creating a desktop protocol.
- [ ] Success prominently presents Final URL, shows Original URL only when different, preserves warning order, and provides a Copy action for Final URL.
- [ ] Failure presents sanitized stage, message, and HTTP status when available while preserving the selected input and overrides.
- [ ] Retry is explicit, performs a fresh configuration preflight, and causes exactly one new attempt; no automatic retry, queue, history, auto-open, or batch behavior is added.
- [ ] Local-server end-to-end tests and a desktop smoke prove one successful upload, one Shortener fallback, one clipboard warning, one structured failure, and one explicit retry.
