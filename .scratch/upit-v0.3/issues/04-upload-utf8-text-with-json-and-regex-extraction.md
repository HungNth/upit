# 04: Upload UTF-8 text through JSON bodies and regex extraction

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Let a version-2 Uploader validate a selected file as UTF-8, stream its text into one configured JSON object value, and identify the uploaded file URL or provider error through a strict regular-expression capture.

**Blocked by:** 03: Upload UTF-8 text through form bodies and raw-body extraction

Status: resolved

- [x] `body: "json"` requires a JSON object containing exactly one string value whose complete content is `{input}`.
- [x] Nested objects and arrays plus static strings, numbers, booleans, and nulls are preserved; array or scalar roots are rejected.
- [x] The selected file must be valid UTF-8 before network access, then its exact text is streamed into the placeholder with correct JSON escaping and bounded memory.
- [x] Placeholder substitution is value-only and full-string-only; zero/multiple placeholders, key matches, substring matches, filename placeholders, base64 conventions, and non-string markers are rejected or left unsupported as specified.
- [x] Upit owns `application/json`; configured Content-Type, file-field, or form-field configuration is rejected before network access.
- [x] Local validation and transformed streaming honor timeout and parent cancellation with the established exit codes and stage mapping.
- [x] `type: "regex"` requires a Go RE2 pattern compiled during configuration validation and supports an optional group string defaulting to group `0`, with valid numeric or named groups checked before network access.
- [x] Regex extraction requires a valid UTF-8 bounded response body, uses the first match, and returns one non-empty whole match or configured capture; no match, missing group, empty capture, or invalid UTF-8 fails at stage `parse`.
- [x] Regex extraction works for required URL and optional provider error extraction; URL results still require absolute HTTP(S) and error extraction cannot create success.
- [x] No regex-to-body/JSON/header fallback occurs, and all existing 2xx, redirect, retry, response-bound, output, and diagnostic contracts remain intact.
- [x] Multipart, binary, and form Request Body Modes plus JSON, header, and body Response Extractors remain green.
- [x] CLI-seam tests assert nested JSON preservation, exact escaped text, generated Content-Type, pre-network validation, pattern/group behavior, first-match behavior, URL/error extraction, output, and exit codes against a real local endpoint.
- [x] Focused tests cover streamed JSON escaping around the placeholder and regex group selection only where the CLI seam would obscure the invariant.
- [x] A demo invocation uploads UTF-8 text through JSON and prints a URL selected by a configured regex capture.

## Comments

- Implemented with structure-aware JSON templating, strict RE2 group validation, all extractor boundary cases, provider-error extraction, and focused invariant tests.
- Changes remain uncommitted pending separate commit authorization.
