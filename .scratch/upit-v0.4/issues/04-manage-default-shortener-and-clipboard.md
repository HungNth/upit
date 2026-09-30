# 04: Manage default Shortener and clipboard behavior

**What to build:** Complete writable Global Configuration management with explicit commands to set or clear the default Shortener and enable or disable default clipboard copying. Each command must use the safe writer, validate only its relevant component, preserve unrelated fields, and support targeted repair without weakening full Configuration Set validation.

**Blocked by:** 03: Safely select the default Uploader

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Exercise every mutation through the production CLI runner and real temporary configuration documents. Reuse the established writer tests rather than retesting its private mechanics; retain upload-level regressions for Shortener fallback and clipboard behavior.

**Demo path:** Set a valid default Shortener, clear it while its definition document is malformed, enable and disable clipboard copying, inspect `config show` after each step, and confirm subsequent uploads preserve existing Shortener fallback and clipboard warning semantics.

- [ ] `set-default-shortener <name>` validates the complete Shortener document, requires the requested case-sensitive Shortener, applies the change, and validates the resulting Global Configuration without loading the Uploader document.
- [ ] Missing Shortener documents, unknown targets, invalid sibling Shorteners, or unrelated invalid global fields prevent a write and produce one actionable diagnostic.
- [ ] `clear-default-shortener` loads only the Global Configuration, can repair a bad current Shortener selection without reading a malformed Shortener document, and preserves Uploader and clipboard settings.
- [ ] An existing malformed optional Shortener document remains invalid under `config validate` after clearing, while uploads no longer load it by default.
- [ ] `enable-clipboard` and `disable-clipboard` load only the Global Configuration, preserve both default names, and do not probe, install, or invoke any platform clipboard helper.
- [ ] Successful mutation stdout is exact and newline-terminated: `Default Shortener set to "<name>".`, `Default Shortener cleared.`, `Clipboard copying enabled.`, or `Clipboard copying disabled.` Names use JSON string escaping inside the quotes.
- [ ] No-op Shortener selection validates the complete relevant Shortener document, does not rewrite, and prints exactly `Default Shortener is already "<name>".` plus newline.
- [ ] No-op clear and clipboard commands validate the resulting Global Configuration, do not rewrite, and print exactly `Default Shortener is already clear.`, `Clipboard copying is already enabled.`, or `Clipboard copying is already disabled.` plus newline.
- [ ] Every mutation uses canonical serialization and the safe staged publication behavior established by the blocking ticket.
- [ ] Existing upload-time Shortener selection, explicit override, disable flag, fallback to Original URL, warnings, cancellation, clipboard ordering, nonfatal clipboard failure, stdout, stderr, and exit codes remain unchanged.
- [ ] Command-specific help and user documentation cover all writable Global Configuration commands without introducing a generic key/value API.
