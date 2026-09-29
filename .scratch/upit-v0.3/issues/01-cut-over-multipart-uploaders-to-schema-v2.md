# 01: Cut over existing multipart Uploaders to schema v2

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Move the existing Uploader document to strict version 2 while preserving the complete multipart plus JSONPath upload workflow. A user who changes only the document version must receive the same HTTP request, result, output, Shortener behavior, clipboard behavior, warnings, stages, and exit codes as v0.2.

**Blocked by:** None (can start immediately)

Status: resolved

- [x] `custom-uploader.json` requires version 2; version 1 and unsupported versions fail before any network request with an actionable diagnostic and exit code 1.
- [x] Global configuration remains version 2 and Shortener configuration remains version 1; neither document gains Uploader protocol fields.
- [x] Existing multipart requests preserve configured method, endpoint, headers, query values, file-field name, literal fields, filename, streamed bytes, and generated multipart Content-Type boundary.
- [x] Existing JSON URL and error extraction preserves full RFC 9535 support, exact-one-string semantics, absolute HTTP(S) URL validation, response limits, and failure stages.
- [x] The complete named Uploader document remains strict-decoded and semantically validated before the selected Uploader sends a request.
- [x] Unknown body or extractor types, unknown fields, incompatible existing multipart/JSON fields, and invalid request/extractor settings fail before network access.
- [x] Plain and JSON output, timeout, parent cancellation, redirect rejection, one-attempt behavior, redaction, Shortener sequencing/fallback, and clipboard sequencing remain unchanged.
- [x] Shipped Uploader examples and permanent fixtures use version 2 without otherwise rewriting working multipart definitions.
- [x] CLI-seam regression tests prove a real multipart upload and JSONPath extraction after the version-only cutover, including rejection of version 1 before the endpoint is called.
- [x] A demo invocation uploads through a version-2 multipart Uploader and emits the same Final URL contract as v0.2.

## Comments

- Implemented and verified with the full Go suite, race detector, static analysis, cross-builds, and the final CLI review.
- Changes remain uncommitted pending separate commit authorization.
