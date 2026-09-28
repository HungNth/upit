# Use RFC 9535 JSONPath for response extraction

Upit uses `github.com/theory/jsonpath` pinned to `v0.12.1` for RFC 9535 response extraction. The full RFC selector language is accepted, but each configured URL or error extractor must select exactly one JSON string: zero matches, multiple matches, `null`, and non-string values are errors rather than implicit coercions; response bodies are limited to 1 MiB before extraction.
