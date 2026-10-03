# 03: Create and edit Uploaders

**What to build:** Give users a complete structured editing path for valid Uploader documents: create an unsaved draft, select an Uploader, edit every existing request, Request Body Mode, and Response Extractor rule, validate inline and on Save, and safely publish the changed document. This slice delivers the reusable editor behavior needed by Setup without adding rename or deletion.

**Blocked by:** 02: Edit Global Configuration safely

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Drive Uploader sessions through the Wails-free application module with real temporary Uploader documents and the existing strict protocol validator. UI behavior checks cover structured field visibility, masked controls, dynamic rows, and field-scoped JSON interaction; do not duplicate protocol validation in frontend tests.

**Demo path:** Create an unsaved Uploader draft, configure multipart and a response extractor, save it, edit it to a different supported Request Body Mode, use nested JSON request data where applicable, and validate the resulting Configuration Set through both Desktop and `upit config validate`.

- [x] The Uploader area presents deterministic master-detail selection and indicates the Global Configuration default.
- [x] New Uploader opens as an unsaved structured draft and cannot overwrite or publish until it has a valid unique name and complete valid definition.
- [x] The structured editor exposes every existing Uploader request, Request Body Mode, and Response Extractor contract without adding a new protocol or extractor.
- [x] Body-mode and extractor-specific controls adapt to the selected type and do not present incompatible fields as valid.
- [x] Header, query, and form-field maps use dynamic key-value controls with immediate duplicate and basic field diagnostics.
- [x] Arbitrary nested request `data` uses a field-scoped JSON editor with syntax feedback and Go-authoritative placeholder/semantic validation.
- [x] Sensitive configured values are masked by default; reveal, edit, and copy require explicit action and reveal state is not persisted.
- [x] Explicit valid-only Save canonically publishes the complete Uploader document, preserves unrelated Uploaders, detects stale snapshots, and does not create backup, lock, or draft files.
- [x] End-to-end tests and an application smoke prove all Request Body Modes and Response Extractor configurations still execute through the existing CLI upload path after desktop editing.

**Earlier verification note:** The permanent post-edit CLI smoke covered multipart/body, binary/header, form/body, json/json, and binary/regex after `SaveUploaderEditor`; focused tests covered masked values, stale saves, invalid drafts, canonical persistence, and incompatible-field cleanup. Later native evidence and the discovered transition fix are recorded below.

## Comments

- 2026-10-03: actual rebuilt macOS Wails WebView edited/saved one real Uploader through multipart/body, binary/header, form/regex, and json/json. For each native Save, actual `bin/upit upload` read the persisted document and uploaded to a real local HTTP server; assertions checked multipart file content, exact binary bytes, decoded form text, JSON text, all four extractors, exact stdout/stderr, and exit 0.
- Found and fixed the UI transition bug: changing extractor type retained incompatible hidden fields, so valid JSON-to-body editing failed Go validation. Setup and Uploader URL/error type changes now clear incompatible fields without weakening Go validation. The permanent native regression ran red against the old embedded bundle and green after rebuilding/restarting, covering URL and error JSON-to-body Save. Full race-enabled Go suite/vet and final frontend/native builds passed.
