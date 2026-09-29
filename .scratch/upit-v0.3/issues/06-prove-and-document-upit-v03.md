# 06: Prove and document Upit v0.3

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Finish the user-facing examples, documentation, regression evidence, and actual-binary demonstration for the complete v0.3 Uploader protocol contract.

**Blocked by:** 05: Harden protocol validation and provider diagnostics

Status: resolved

- [x] User documentation explains the Uploader version-2 cutover, mechanical multipart migration, four Request Body Modes, four Response Extractors, strict mode-specific fields, Content-Type ownership, and unchanged CLI/output contract.
- [x] Credential-free examples show valid multipart, binary, form, and JSON Uploaders plus JSON, header, regex, and body URL/error extraction without unsupported placeholders or secrets.
- [x] Documentation states that form/JSON require UTF-8 and exactly one full-string `{input}`, use bounded-memory two-pass processing, and do not support base64, filename, key, substring, header, query, or URL substitution.
- [x] Documentation states that Uploader failures remain fatal exit 1, parent cancellation exits 130, Shortener fallback remains post-upload behavior, redirects/retries remain disabled, and responses remain bounded to 1 MiB.
- [x] Architecture and domain documentation use the canonical Uploader, Request Body Mode, Response Extractor, Shortener, Original URL, and Final URL terms and describe v0.3 decisions as implemented rather than proposed.
- [x] Permanent CLI-seam coverage exercises every Request Body Mode and every Response Extractor through representative end-to-end combinations while preserving multipart/JSONPath and v0.2 Shortener regressions.
- [x] Focused invariant tests remain limited to streaming UTF-8 validation, form percent encoding, JSON escaping, placeholder counting, and regex group selection.
- [x] The complete Go test suite, race detector, and static analysis pass.
- [x] Native, headless Linux amd64, and Windows amd64 CLI builds succeed without Wails or WebView dependencies.
- [x] An actual built binary uploads raw bytes to a local endpoint, sends the expected binary Content-Type, extracts the Final URL from one response header, exits successfully, and leaves no resident Upit process.
- [x] No throwaway servers, temporary configs, generated binaries, stale version-1 Uploader examples, or unrelated artifacts remain in the repository.

## Comments

- Implemented, documented, and verified. All changes remain uncommitted pending separate commit authorization.
