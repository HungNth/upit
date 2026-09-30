# 05: Manage Shorteners

**What to build:** Deliver the full optional Shortener editor and lifecycle. Users can create, edit, explicitly rename, and confirm deletion of Shorteners while retaining existing Shortener request-template, JSON-only Response Extractor, fallback, validation, secret, and reference semantics. The final unreferenced Shortener removes its optional document instead of persisting an invalid empty one.

**Blocked by:** 02: Edit Global Configuration safely

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Use Wails-free application-module sessions and real optional Shortener documents for creation, publication, deletion, reference, and validation behavior. Use local HTTP only when proving an edited Shortener's existing runtime behavior; the desktop smoke demonstrates the visible editor and confirmation path.

**Demo path:** Create a Shortener with masked credentials and JSON request data, select it as default, observe rename/delete blocked, clear the default, rename it, delete it with confirmation, and observe `custom-shortener.json` removed after the final deletion while CLI upload still works without shortening.

- [x] The Shortener area handles absent optional documents and presents deterministic named-definition selection with default indication when present.
- [x] New Shortener drafts, structured request/data/JSON-extractor editing, nested data handling, masked secrets, inline diagnostics, and explicit valid-only Save match the established editor contract.
- [x] Shortener behavior remains limited to existing request-template rules and JSON-only Response Extractors; no new Shortener protocol, extractor, or provider adapter is added.
- [x] Rename is explicit, validates exact unique names, and is blocked while Global Configuration selects the Shortener.
- [x] Delete requires confirmation and is blocked while Global Configuration selects the Shortener.
- [x] Deleting the final unreferenced Shortener removes the optional Shortener document rather than publishing an empty invalid document.
- [x] Saves and deletion detect stale documents, use safe native publication, preserve strict diagnostics/redaction, and introduce no backup, lock, automatic merge, or cross-document transaction journal.
- [x] Local-server regression coverage proves an edited Shortener still produces the existing Final URL or nonfatal fallback warning behavior through the production upload workflow.

**Verification note:** Focused tests cover optional creation, masked values, reference-blocked rename/delete, clearing the default, final-document removal, and safe publication. CLI smoke covers edited Shortener Final URL and nonfatal fallback. Native WebView UI interaction was not directly automated.
