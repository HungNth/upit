# 03: Upload UTF-8 text through form bodies and raw-body extraction

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Let a version-2 Uploader validate a selected file as UTF-8, stream its text into one URL-encoded form field, and obtain the uploaded file URL or provider error from the trimmed plain response body.

**Blocked by:** 02: Stream binary uploads and extract header URLs

Status: resolved

- [x] `body: "form"` requires a non-empty string field map containing exactly one value whose complete content is `{input}`.
- [x] The selected file must be valid UTF-8; invalid input fails at stage `validation` before the endpoint receives a request.
- [x] Upit validates UTF-8 with bounded memory, then streams a second-pass `application/x-www-form-urlencoded` body without buffering the complete file.
- [x] File text replaces only the one full-string `{input}` value; other values remain literal and substitution does not occur in keys, substrings, headers, queries, URLs, or any other field.
- [x] Form encoding preserves the complete UTF-8 text semantically, percent-encodes names and values according to the URL-encoded form contract, and handles empty valid files.
- [x] Upit owns `application/x-www-form-urlencoded`; a configured Content-Type, file-field, or JSON-data field is rejected before network access.
- [x] Local validation and transformed streaming honor the same whole-invocation context: parent cancellation exits 130, deadline expiry exits 1, and encoding/stream failures use stage `request`.
- [x] `type: "body"` requires a valid UTF-8 response body, trims surrounding Unicode whitespace, and returns one non-empty string for URL or provider error extraction.
- [x] Body URL results still require absolute HTTP(S); invalid UTF-8, empty trimmed bodies, invalid URLs, non-2xx responses, redirects, and oversized bodies remain fatal without extractor fallback.
- [x] Binary/header and multipart/JSONPath behavior remains green.
- [x] CLI-seam tests use a real local endpoint to assert decoded fields, exact text replacement, generated Content-Type, non-UTF-8 preflight rejection, cancellation, timeout, raw-body URL/error extraction, diagnostics, output, and exit codes.
- [x] Focused tests cover UTF-8 validation across chunk boundaries and streamed percent encoding only where those invariants are impractical to observe through the CLI seam.
- [x] A demo invocation uploads a UTF-8 text file through a form body and prints a URL returned as plain response text.

## Comments

- Implemented with two-pass UTF-8 validation, streamed form encoding, raw-body extraction, cancellation/deadline handling, and focused invariant tests.
- Changes remain uncommitted pending separate commit authorization.
