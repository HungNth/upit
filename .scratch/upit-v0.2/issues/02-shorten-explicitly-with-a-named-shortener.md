# 02: Shorten explicitly with a named Shortener

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Let a user pass `--shortener <name>` to run one named Shortener after a successful upload. Upit must validate the selected credential-bearing configuration before upload, send the Original URL through the approved configurable JSON HTTP contract, preserve that Original URL, and emit the extracted Final URL through the unchanged success output.

**Blocked by:** 01: Cut over unchanged uploads to global config v2

Status: resolved

- [x] `--shortener <name>` works before or after the file operand and is documented in command help.
- [x] Selecting a Shortener loads strict `custom-shortener.json` version 1, whose top-level `shorteners` map contains named definitions.
- [x] The complete Shortener document is strict-decoded and every definition is semantically validated before the upload endpoint is called.
- [x] Unix rejects group/other permission bits with actionable `chmod 600` guidance; Windows does not apply POSIX mode checks.
- [x] A Shortener requires a valid HTTP method, absolute HTTP(S) URL, valid static headers, non-empty static query keys, a JSON object data template, and exactly one string value equal to `{input}`.
- [x] A configured Content-Type is rejected; Upit sends JSON with `Content-Type: application/json` and preserves other configured headers, query parameters, nested objects, arrays, and scalar values.
- [x] Only the one full-string `{input}` value becomes the Original URL; substrings, object keys, and non-string values are not substituted.
- [x] The Shortener request is sent only after upload success, follows no redirects, accepts only 2xx, and reads at most 1 MiB of response data.
- [x] The required RFC 9535 URL extractor selects exactly one JSON string that is validated as an absolute HTTP(S) URL; the optional error extractor uses the same cardinality/type rules.
- [x] Success preserves Original URL, updates Final URL, keeps the existing plain and JSON output shapes, and exits 0.
- [x] Missing files, unsupported versions, unknown fields, absent names, invalid definitions, malformed templates, and permission failures exit 1 before upload.
- [x] Black-box CLI tests use separate local upload and Shortener endpoints to assert request details and Original/Final URL output end to end; focused tests cover recursive substitution invariants.
- [x] A demo invocation with `--shortener` prints the shortened Final URL and JSON mode shows distinct Original and Final URLs.
