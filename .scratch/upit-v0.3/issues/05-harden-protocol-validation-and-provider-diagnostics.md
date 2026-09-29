# 05: Harden protocol validation and provider diagnostics

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Complete the cross-protocol safety contract so every version-2 Uploader is validated consistently, every Response Extractor follows the same failure rules, and endpoint-provided Uploader or Shortener diagnostics cannot inject additional terminal or log lines.

**Blocked by:** 02: Stream binary uploads and extract header URLs; 03: Upload UTF-8 text through form bodies and raw-body extraction; 04: Upload UTF-8 text through JSON bodies and regex extraction

Status: resolved

- [x] The complete Uploader document is semantically validated before upload; an invalid unselected definition blocks the invocation without calling any endpoint.
- [x] Every Request Body Mode rejects all incompatible fields and enforces its required fields, Content-Type ownership, placeholder count, method, URL, header, query, and media-type rules.
- [x] `Host`, `Content-Length`, `Transfer-Encoding`, `Connection`, `Trailer`, `Upgrade`, and `Proxy-Connection` are rejected as transport-managed request headers before network access.
- [x] Every Response Extractor rejects fields belonging to another extractor type; JSONPath, header names, regex patterns, and regex groups are validated before network access.
- [x] JSON, header, regex, and body work consistently for optional provider error extraction.
- [x] Non-2xx responses use a successfully extracted provider error when available and otherwise retain the generic HTTP diagnostic.
- [x] A 2xx URL extraction failure uses a successfully extracted provider error when available and otherwise retains the original parse diagnostic; error extraction never creates success.
- [x] No automatic fallback or chaining occurs between Response Extractor types.
- [x] Every success path still requires 2xx, a body no larger than 1 MiB, and one absolute HTTP(S) URL, including header extraction; redirects remain rejected and no upload is retried.
- [x] Endpoint-provided Uploader and Shortener messages are trimmed and normalized to one printable line by replacing CR, LF, and other control characters with spaces before redaction and output.
- [x] Existing configured header, query, static form field, JSON string, and endpoint-query values remain redacted; Upit does not buffer full file content for redaction and never emits request bodies in its own diagnostics.
- [x] Failure stages remain `config`, `validation`, `request`, `network`, `response`, and `parse`; plain/JSON output shapes and exits 0, 1, 2, and 130 remain unchanged.
- [x] Shortener runtime fallback, deadline fallback, parent-cancellation failure, warning order, clipboard ordering, and lazy configuration behavior remain unchanged apart from provider-message normalization.
- [x] CLI-seam tests cover every externally distinct validation, non-2xx, body-limit, parse, cardinality, UTF-8, URL, redaction, control-character, timeout, and cancellation outcome without duplicate helper-wiring tests.
- [x] The full suite proves one request attempt, no false success, no cross-extractor fallback, and unchanged v0.2 automation contracts.

## Comments

- Implemented and verified with normal/race suites, static analysis, cross-builds, smoke execution, and final standards/spec review.
- Changes remain uncommitted pending separate commit authorization.
