# Upit v0.3 — More Upload Protocols

Status: ready-for-agent

## Problem Statement

Upit v0.2 can upload one file only as multipart form-data and can identify the uploaded file URL only through RFC 9535 JSONPath. Custom upload endpoints also accept raw binary streams, URL-encoded text fields, or JSON text payloads, and they may return the resulting URL in a response header, plain-text body, or non-JSON body matched by a regular expression.

Without those protocols, users must place provider-specific proxies or scripts in front of otherwise compatible endpoints. The next release must extend the Uploader contract without changing CLI automation, weakening strict validation, buffering large files without bound, or expanding the Shortener contract.

## Solution

Make a clean cutover of Uploader configuration to version 2 and add four Request Body Modes—`multipart`, `binary`, `form`, and `json`—plus four Response Extractors—`json`, `header`, `regex`, and `body`.

Multipart behavior remains unchanged. Binary streams the selected file as the complete request body. Form and JSON are text-upload protocols: the selected file must be valid UTF-8 and replaces exactly one full-string `{input}` value in configured form fields or a JSON object. Upit validates the text before any network request and streams the transformed body with bounded memory.

Every Response Extractor produces exactly one string under strict type-specific rules. Successful URL extraction still requires one absolute HTTP(S) URL. Uploader request, response, or extraction failures remain fatal upload failures; only the separately configured Shortener retains its post-upload fallback behavior.

## User Stories

1. As a user of a raw-upload endpoint, I want to send a file as the complete request body, so that I do not need a multipart proxy.
2. As a user of a URL-encoded text endpoint, I want to send a UTF-8 file through a configured form field, so that text-sharing APIs can be used as Uploaders.
3. As a user of a JSON text endpoint, I want to place UTF-8 file contents into a configured JSON object, so that structured text APIs can be used as Uploaders.
4. As an existing user, I want multipart Uploader behavior to remain unchanged after updating the Uploader document version, so that the protocol expansion does not alter existing uploads.
5. As an existing user, I want current JSONPath URL and error extraction to remain unchanged, so that existing response contracts continue to work.
6. As a user, I want Uploader configuration version 1 rejected with an actionable diagnostic, so that the changed schema is never guessed or partially accepted.
7. As a user, I want Uploader version 2 to retain the current named Uploader model, so that protocol selection remains part of each destination definition.
8. As a Shortener user, I want Shortener configuration to remain version 1 with its current JSON request and JSONPath response contract, so that v0.3 does not silently expand post-processing behavior.
9. As an automation author, I want global configuration to remain version 2, so that adding Uploader protocols does not cause an unrelated global migration.
10. As a user uploading arbitrary bytes, I want binary mode to preserve file bytes exactly, so that images, archives, and other non-text data are not transformed.
11. As a user uploading a large binary file, I want it streamed from disk, so that memory use does not grow with file size.
12. As a binary endpoint user, I want `application/octet-stream` used when no Content-Type is configured, so that behavior is deterministic across operating systems.
13. As a binary endpoint user, I want to configure a valid Content-Type explicitly, so that endpoints requiring a specific media type can be described.
14. As a user, I want binary mode to send no multipart wrapper or Content-Disposition, so that the endpoint receives only file bytes.
15. As a form endpoint user, I want the selected file required to be valid UTF-8, so that URL-encoded text has one defined character interpretation.
16. As a form endpoint user, I want exactly one configured field value equal to `{input}` replaced with the complete file text, so that the file destination is explicit.
17. As a form endpoint user, I want other field values sent literally, so that provider parameters remain predictable.
18. As a form endpoint user, I want standards-compliant URL encoding with `application/x-www-form-urlencoded`, so that ordinary form decoders recover the configured values.
19. As a JSON endpoint user, I want exactly one string value equal to `{input}` replaced with the complete file text, so that the text destination is explicit.
20. As a JSON endpoint user, I want nested objects, arrays, numbers, booleans, nulls, and static strings preserved, so that provider-specific payloads remain expressible.
21. As a JSON endpoint user, I want the configured data root to be an object, so that the request shape remains readable and consistent with existing configuration.
22. As a user, I want `{input}` inside a larger string, object key, header, query value, or endpoint URL left unsupported, so that substitution cannot mutate unrelated configuration.
23. As a user, I want filename and base64 placeholders left unsupported, so that v0.3 does not invent a broader template language.
24. As a user, I want non-UTF-8 form or JSON input rejected before the endpoint is called, so that an invalid text upload cannot partially reach the provider.
25. As a user uploading a large text file, I want form and JSON transformation to use bounded memory, so that protocol choice does not create unbounded allocation.
26. As an automation author, I want one invocation timeout to cover text validation, encoding, upload, optional shortening, and clipboard work, so that `--timeout` keeps its whole-workflow meaning.
27. As a user, I want Ctrl-C during local validation or transformed streaming to stop promptly and exit 130, so that local preprocessing remains interruptible.
28. As a custom endpoint user, I want any syntactically valid HTTP method accepted, so that Upit does not impose a speculative method whitelist.
29. As a custom endpoint user, I want configured headers and query values sent literally, so that authentication and provider parameters remain predictable.
30. As a user, I want Upit to own multipart boundaries, URL-encoded form Content-Type, and JSON Content-Type, so that configured headers cannot disagree with the generated body.
31. As a user, I want managed transport headers rejected before upload, so that configuration is not silently ignored or overwritten by the HTTP stack.
32. As a user of a header-returning endpoint, I want to extract the URL from one configured response header, so that a response body is not required to carry the result.
33. As a user of a text-returning endpoint, I want to extract the URL with a configured regular expression and capture group, so that non-JSON provider responses are supported.
34. As a user of a plain-URL endpoint, I want the trimmed response body used directly, so that no expression is needed for a body containing only the URL.
35. As a JSON endpoint user, I want full RFC 9535 JSONPath support retained, so that nested and filtered responses continue to work.
36. As a user, I want a header extractor to require exactly one non-empty header value, so that duplicate or absent values are not guessed or joined.
37. As a regex user, I want Go RE2 syntax, the first match, and an optional whole, numeric, or named capture group, so that extraction is deterministic and resistant to catastrophic backtracking.
38. As a body or regex user, I want invalid UTF-8 responses rejected, so that extracted text has one defined encoding.
39. As a user, I want every URL extractor result validated as an absolute HTTP(S) URL, so that a successful upload always returns a usable link.
40. As a user, I want JSON, header, regex, and body extraction available for provider error messages, so that varied endpoints can produce useful diagnostics.
41. As a user, I want an error extractor to improve only the diagnostic and never convert a failed upload into success, so that failure semantics remain reliable.
42. As a user, I want no automatic fallback between Response Extractor types, so that configuration errors cannot become false success.
43. As a user, I want only HTTP 2xx responses eligible for URL extraction, so that redirects and endpoint failures are never mistaken for upload success.
44. As a security-conscious user, I want every response body limited to 1 MiB even when the URL comes from a header, so that extractor choice cannot bypass the response bound.
45. As a user, I want redirects disabled and uploads attempted once, so that file data and credentials are not replayed to another endpoint.
46. As a user, I want the complete Uploader document semantically validated before any upload, so that one configuration file is either wholly valid or rejected.
47. As a user, I want fields incompatible with a Request Body Mode or Response Extractor rejected, so that copied or misspelled configuration is not ignored.
48. As a regex user, I want invalid patterns and missing configured groups rejected before network access, so that response parsing cannot fail from a known configuration defect.
49. As a header user, I want invalid or empty configured header names rejected before network access, so that extraction behavior is explicit.
50. As a security-conscious user, I want configured credentials and static request values redacted from endpoint-provided diagnostics, so that existing secret-handling guarantees continue.
51. As a user, I want endpoint-provided messages normalized to one printable line before output, so that a provider cannot inject extra terminal or log lines.
52. As a user uploading sensitive file content, I want Upit never to log or construct its own diagnostics from the file body, so that local payload handling remains private.
53. As a user, I accept that configured provider error extraction may return text chosen by that provider, so that bounded-memory operation does not require retaining the full file solely for redaction.
54. As an automation author, I want the existing plain and JSON success and failure shapes unchanged, so that scripts do not need a new parser.
55. As an automation author, I want existing failure stage names and exit codes preserved, so that scripts can continue distinguishing validation, request, network, response, parse, usage, and interruption outcomes.
56. As a Shortener user, I want Uploader protocol failures to remain fatal before shortening starts, so that Shortener fallback is not confused with upload success.
57. As a clipboard user, I want clipboard behavior to remain after successful upload and optional shortening, so that protocol choice does not alter post-processing order.
58. As a headless user, I want all new protocols to remain CLI-only with no Wails, WebView, daemon, or resident-process dependency, so that server use remains unchanged.
59. As a maintainer, I want protocol variation hidden behind the existing deep upload module interface, so that callers do not need to understand builders or extractors.
60. As a maintainer, I want every body mode and Response Extractor proven through the production CLI seam, so that configuration, HTTP behavior, diagnostics, output, and exit codes are verified together.

## Implementation Decisions

- v0.3 delivers all roadmap Uploader protocol additions in one release: `multipart`, `binary`, `form`, and `json` Request Body Modes; `json`, `header`, `regex`, and `body` Response Extractors; and stricter protocol validation.
- The release remains CLI-only. It adds no command flags, result fields, desktop surface, configuration-management commands, or JSON Schema files.
- `custom-uploader.json` makes a clean cutover from version 1 to version 2. Version 1 is rejected; there is no compatibility reader, fallback, automatic migration, or dual-version period.
- Existing multipart plus JSONPath definitions migrate mechanically by changing the document version. Their request, response, timeout, cancellation, output, Shortener, and clipboard behavior must not otherwise change.
- Global configuration remains version 2. Shortener configuration remains version 1 and does not gain the new Request Body Modes or Response Extractors.
- The Uploader request retains common method, absolute HTTP(S) URL, header, query, and body-mode fields. Mode-specific fields are strict: incompatible or unused fields are rejected rather than ignored.
- Multipart requires a non-empty file-field name, accepts optional literal string fields, streams the file as a named part, and continues to own the generated multipart Content-Type boundary.
- Binary accepts no file-field, form-field, or JSON-data configuration. It streams the exact file bytes as the complete request body and sends no Content-Disposition.
- Binary uses `application/octet-stream` when no Content-Type header is configured. A configured binary Content-Type must be a valid non-empty media type and is sent as configured. Filename-extension MIME inference is not used.
- Form requires a string field map containing exactly one value whose complete content is `{input}`. The selected file must be valid UTF-8; its exact text replaces that value, all other values remain literal, and the result is streamed using standards-compliant `application/x-www-form-urlencoded` encoding.
- JSON requires a data object containing exactly one string value whose complete content is `{input}`. The selected file must be valid UTF-8; its exact text replaces that value, nested JSON values are preserved, and the result is streamed as `application/json`.
- Placeholder matching is value-only and full-string-only. There is no substitution in keys, substrings, headers, queries, endpoint URLs, multipart fields, or binary data. Base64 and filename placeholders are not supported.
- Form and JSON perform a bounded-memory UTF-8 validation pass before network access, then stream the transformed body in a second pass. Empty UTF-8 files are valid. No arbitrary request file-size cap is introduced.
- One context covers configuration, local file work, upload, optional Shortener work, and clipboard work. Parent cancellation remains interruption exit 130; deadline expiry remains runtime failure exit 1 at the stage where it occurs.
- Any syntactically valid HTTP method remains accepted, and the configured Request Body Mode is sent with that method.
- Headers and query values remain literal. Header names and values retain CR/LF validation. Query keys must remain non-empty.
- Upit owns Content-Type for multipart, form, and JSON and rejects a configured Content-Type for those modes. Binary may configure it as described above.
- Uploader version 2 rejects transport-managed `Host`, `Content-Length`, `Transfer-Encoding`, `Connection`, `Trailer`, `Upgrade`, and `Proxy-Connection` headers rather than allowing them to be ignored or conflict with the HTTP transport.
- The complete Uploader document is strict-decoded and every named Uploader is semantically validated before the selected upload starts.
- A Response Extractor has type-specific fields. JSON uses an RFC 9535 path. Header uses a header name. Regex uses a pattern plus an optional group string whose default is group `0` and whose value may identify a numeric or named group. Body has no additional field.
- JSON extraction preserves the existing full RFC 9535 implementation and exact-one-string rule.
- Header extraction is case-insensitive by HTTP semantics and requires exactly one non-empty header value. Zero or multiple values are parse failures; values are not joined.
- Regex extraction compiles with Go's RE2 engine during configuration validation, operates on the complete bounded UTF-8 response body, uses the first match, and returns the configured whole, numeric, or named capture group. No match, an absent group, or an empty selected value is a parse failure.
- Body extraction requires valid UTF-8, trims surrounding Unicode whitespace from the complete bounded response body, and requires a non-empty result.
- Every URL extractor result must be an absolute HTTP(S) URL. Error extractor results are messages and do not receive URL validation.
- URL and error extraction support all four extractor types. There is no automatic fallback or chaining between types.
- Non-2xx responses remain failures. A successful error extractor may replace the generic HTTP diagnostic; a failed error extractor leaves the generic diagnostic intact.
- On a 2xx response whose URL extraction fails, a successful error extractor may replace the parse diagnostic; otherwise the original URL extraction error is preserved. Error extraction never changes failure into success.
- Response bodies are always read and limited to 1 MiB before success, including when URL extraction uses a response header. Response Content-Type is not required or used to choose an extractor.
- Redirects remain disabled. Uploader requests are attempted once with no retry or backoff.
- Failure stages remain `config`, `validation`, `request`, `network`, `response`, and `parse`. Invalid local file/text input is validation; body construction or transformed streaming is request; transport is network; non-2xx/read/size failures are response; extraction and result URL validation are parse.
- Endpoint-provided Uploader and Shortener messages are trimmed and normalized to one printable line by replacing CR, LF, and other control characters with spaces before redaction and output.
- Existing configured secret redaction continues for non-empty header, query, static field, JSON string, and endpoint-query values. Upit does not retain full file content solely to scrub provider-selected messages and never emits request bodies from its own diagnostics.
- Request construction, HTTP execution, bounded response handling, extraction, URL validation, and structured failures remain behind one deep upload module interface. Private files or functions may separate body writers and Response Extractors, but no public request/response packages, strategy interfaces, factories, or provider adapters are introduced.
- Plain success remains exactly the Final URL plus newline. JSON success remains `success`, `originalUrl`, and `finalUrl`. Failure JSON and exit codes remain unchanged. New protocol capabilities are selected only through Uploader configuration.

## Testing Decisions

- The primary permanent seam remains the CLI command-running interface with real temporary configuration, a temporary input file, stdout/stderr buffers, and local HTTP servers. Tests assert externally visible HTTP requests, outputs, warnings, stages, and exit codes rather than helper calls.
- Existing multipart plus JSONPath black-box coverage remains the regression proof for the behavior-preserving Uploader v2 cutover.
- Every Request Body Mode is exercised end to end through the CLI. Representative pairings cover multipart with JSON, binary with header, form with body, and JSON with regex while additional cases cover error extraction and failure boundaries.
- Binary tests assert exact streamed bytes, absence of multipart framing and Content-Disposition, deterministic default Content-Type, valid configured Content-Type, rejection of invalid media types, cancellation, and no file-sized memory dependency.
- Form tests assert UTF-8 replacement, literal static fields, standards-compliant percent encoding, generated Content-Type, exact-one-placeholder validation, non-UTF-8 rejection before network, and bounded transformed streaming.
- JSON tests assert recursive object/array preservation, exact full-string placeholder replacement, JSON escaping of file text, generated Content-Type, object-root validation, non-UTF-8 rejection before network, and bounded transformed streaming.
- Strict request validation tests cover incompatible mode fields, forbidden Content-Type overrides, managed headers, empty names, invalid methods/URLs, and one invalid unselected Uploader blocking the entire document before network.
- Every Response Extractor is exercised through the CLI for URL success and provider error extraction where applicable.
- Header tests cover case-insensitive lookup, zero values, one value, multiple values, empty values, and URL validation.
- Regex tests cover configuration-time compile errors, group zero, numeric groups, named groups, absent groups, no match, empty captures, first-match behavior, invalid UTF-8, and URL validation.
- Body tests cover surrounding whitespace trimming, empty bodies, invalid UTF-8, provider error text, and URL validation.
- JSONPath regression tests preserve full RFC 9535 selectors and exact-one-string behavior.
- Cross-extractor tests prove there is no fallback, only 2xx can produce success, redirects remain rejected, the 1 MiB body limit applies even to header extraction, and response Content-Type does not select behavior.
- Diagnostic tests prove provider messages are one printable line, configured values remain redacted, file contents are never included by Upit's own errors, and error-extractor failure preserves the generic or original diagnostic according to the defined precedence.
- Workflow tests prove unchanged plain/JSON output, stage mapping, exit codes, Shortener sequencing and fallback, clipboard ordering, one-attempt behavior, whole-invocation timeout, and exit 130 for parent cancellation during validation or upload.
- Focused lower-level tests are limited to invariants cumbersome to prove through HTTP: streaming UTF-8 validation across chunk boundaries, form percent-encoding, JSON string escaping around a streamed placeholder, placeholder counting, and regex group selection. They must not test wiring or private function names.
- Race-sensitive cancellation coverage runs under the Go race detector.
- Completion requires an actual built binary against a local endpoint using binary upload plus a header URL extractor. The smoke observes raw bytes, Content-Type, the Final URL, process exit, and absence of a resident Upit process.

## Out of Scope

- XML or no-body Request Body Modes.
- New Request Body Modes or Response Extractors for Shorteners.
- Base64, filename, path, environment-variable, or general string-template substitution.
- Placeholder substitution in headers, queries, URLs, keys, multipart fields, string substrings, or binary content.
- JSON array or scalar roots for Uploader data.
- Repeated URL-encoded form keys or ordered duplicate fields.
- Automatic MIME inference from file names or file contents.
- Configurable charset conversion, line-ending normalization, compression, encryption, or request signing.
- Automatic Response Extractor chains, fallback lists, status-specific extractors, XML/XPath, response-URL extraction, or custom scripts.
- Thumbnail URL, deletion URL, response metadata, headers, or raw bodies in the public result or JSON output.
- Uploader retries, resumable uploads, chunk protocols, concurrent files, multiple files, or progress rendering.
- Configuration create/edit/migrate commands, automatic migration, compatibility shims, or published JSON Schema files.
- Environment-variable substitution, operating-system keychains, or per-field secret metadata.
- Upload history, persistence, telemetry, metrics, provider adapters, or a general HTTP-processing framework.
- Wails, WebView, desktop UI, system tray, notifications, shell integration, daemon, or resident processes.
- Release archives, installers, container images, and automated publishing.

## Further Notes

- The canonical domain terms are Uploader, Request Body Mode, Response Extractor, Shortener, Original URL, and Final URL.
- The Uploader schema cutover, UTF-8 text-template decision, and deep-module seam are recorded in ADRs 0006, 0007, and 0008.
- Primary-source research on ShareX behavior and HTTP semantics is recorded with the v0.3 effort. It establishes that file/image upload precedents stream multipart or raw binary, while form and JSON are text-input protocols; Upit's chosen UTF-8 template contract is explicit rather than inferred.
- The v0.1 and v0.2 specifications remain authoritative for behavior not explicitly changed here.
- Implementation follows the approved v0.3 tickets; source changes remain uncommitted pending separate commit authorization.
