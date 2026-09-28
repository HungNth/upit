# 01: Cut over unchanged uploads to global config v2

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Move the existing upload CLI to strict global configuration version 2 without enabling shortening by default. A user who updates the version field and leaves the optional default Shortener unset must receive the same upload request, output, clipboard behavior, warnings, and exit codes as v0.1.

**Blocked by:** None (can start immediately)

Status: resolved

- [x] Global configuration requires version 2 and accepts the optional `defaultShortener` field while retaining the default Uploader and clipboard default.
- [x] Global configuration version 1 is rejected before any upload request, with an actionable version diagnostic and exit code 1.
- [x] Uploader configuration remains strict version 1 and its request/response contract is unchanged.
- [x] With no default Shortener, `upit upload` does not require or read Shortener configuration and preserves the exact v0.1 plain and JSON success shapes.
- [x] Existing timeout, cancellation, redirect, response-bound, redaction, and clipboard behavior remains green.
- [x] Shipped examples and permanent test fixtures use global configuration version 2.
- [x] The primary CLI-seam test proves one real multipart upload through a local endpoint with no Shortener file present.
- [x] A demo invocation shows a version-2 configuration uploading successfully with Original URL equal to Final URL.
