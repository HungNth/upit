# 04: Fall back safely when runtime shortening fails

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Preserve a completed upload when the selected Shortener fails at runtime. Upit must return the Original URL as the Final URL, keep successful output and exit status, and expose one sanitized warning without retrying an uncertain request.

**Blocked by:** 02: Shorten explicitly with a named Shortener

Status: resolved

- [x] Shortener network, non-2xx, oversized-response, invalid-JSON, JSONPath cardinality/type, invalid-Final-URL, and context-deadline failures fall back to Original URL.
- [x] Fallback keeps exit code 0, sets Final URL equal to Original URL, and preserves the exact existing plain or JSON success payload on stdout.
- [x] Both plain and JSON modes add exactly one stderr line beginning `Warning: shorten URL: ` for the Shortener failure.
- [x] A usable configured error extractor may replace the generic HTTP/extraction message with provider text.
- [x] Provider text is scrubbed of configured header values, query values, string data values, and endpoint query values before output.
- [x] A failed Shortener request is not retried, and the upload request is never repeated.
- [x] A deadline reached during shortening uses the remaining whole-invocation timeout budget and follows fallback behavior rather than interruption behavior.
- [x] Redirect responses are not followed and enter the same sanitized fallback path.
- [x] CLI-seam tests prove each externally distinct failure class, warning placement, unchanged JSON shape, redaction, one Shortener attempt, and one upload attempt.
- [x] A demo with a failing Shortener prints the Original URL, writes the warning to stderr, and exits 0.
