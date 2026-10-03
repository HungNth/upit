# 01: Strict shared configuration diagnostics

**What to build:** Make every existing upload invocation use one strict, deterministic configuration decoder and validation contract. Invalid configuration must fail before network access with one actionable, redacted diagnostic that identifies the absolute document path and exact configuration path. The completed slice is visible through the current `upload` command and establishes the shared behavior later config commands reuse.

**Blocked by:** None (can start immediately)

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise invalid and valid Configuration Sets through the production CLI runner with temporary homes, captured stdout/stderr, and local HTTP servers only when proving that validation stops before network access. Use focused lower-level tests only where duplicate-key location or deterministic traversal cannot be demonstrated clearly through the CLI.

**Demo path:** Run the built CLI against a Configuration Set containing multiple independent errors, observe the same first absolute-file/configuration-path diagnostic across repeated runs, repair that error, and observe the next deterministic error without any endpoint request.

- [x] Upload configuration loading rejects duplicate JSON object keys at every nesting depth before map decoding can discard an earlier value.
- [x] Global Configuration, Uploader, and Shortener documents retain versions 2, 2, and 1; unsupported-version errors report received and expected versions and state that automatic migration is unavailable.
- [x] Uploader and Shortener definition names are case-sensitive, non-empty printable Unicode with no leading or trailing Unicode whitespace; invalid names are rejected without trimming or normalization.
- [x] The required Uploader document contains at least one Uploader; a present Shortener document contains at least one Shortener; each definition requires its request, response, and URL-extractor structure.
- [x] Shortener header names reject case-insensitive duplicates while retaining the existing JSON-only Shortener Response Extractor contract.
- [x] Existing v0.3 Request Body Mode, Response Extractor, Shortener, permission, redaction, transport, and output behavior remains unchanged except for the explicitly tightened malformed/ambiguous configuration cases.
- [x] Validation reports one error at a time in deterministic order: Global Configuration, Uploaders, Shorteners; definition names use exact lexicographic order; field checks use fixed order.
- [x] Semantic errors report the absolute file, an RFC 9535-style path with escaped bracket notation for dynamic keys, the violated rule, and an actionable correction.
- [x] Malformed JSON reports the absolute file, root path, line, column, and decoder cause without guessing a field path.
- [x] Diagnostics never emit credential values, request values, static body values, secret-bearing URL values, or control-character output.
- [x] Valid existing example configurations and representative upload flows still succeed with unchanged upload stdout, stderr, JSON shapes, exit codes, Shortener fallback, and clipboard ordering.

## Comments

- 2026-10-03: `go test -race ./...` and `go vet ./...` passed on macOS arm64. CLI/schema parity suites exercised duplicate keys, strict structure, deterministic diagnostics, redaction, versions, permissions, and retained upload behavior. Actual `bin/upit` local-server smoke passed plain/JSON uploads and rejected version-1 Global Configuration with exit 1 and empty stdout.
