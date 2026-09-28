# Upit v0.2 — URL Shortener

Status: ready-for-agent

## Problem Statement

Upit v0.1 can upload a file and return the remote file URL, but users who publish or automate those URLs must call a second tool when they need a compact shareable link. That separate step duplicates credential, timeout, error, and clipboard handling, and it can obscure the Original URL that proves where the file was uploaded.

The next release must add optional URL shortening without changing the default v0.1 upload behavior, turning a completed upload into a failure when a shortening service is unavailable, or pulling later upload protocols, configuration commands, and desktop work into scope.

## Solution

Add named Shorteners as an optional post-processing step in the existing on-demand CLI workflow. A user may select a Shortener through global configuration or per invocation, or disable the configured default for one invocation. After a successful upload, Upit sends the Original URL to the selected Shortener as a configurable JSON HTTP request, extracts one absolute HTTP(S) URL from the bounded JSON response, preserves the Original URL, sets the extracted URL as the Final URL, then applies the existing clipboard and output behavior.

Shortener configuration is validated before upload. Ordinary runtime shortening failures fall back to the Original URL, emit a sanitized warning, and preserve successful exit and output. Explicit interruption during shortening remains an interruption.

## User Stories

1. As a CLI user, I want Upit to shorten an uploaded file URL in the same invocation, so that I do not need a second command.
2. As an existing user, I want uploads without a selected Shortener to behave exactly as v0.1, so that current automation is not silently changed.
3. As a user, I want named Shorteners, so that I can configure several shortening services.
4. As a user, I want an optional default Shortener in global configuration, so that routine uploads can shorten without repeating a flag.
5. As a user, I want `--shortener <name>` to override the configured default for one invocation, so that destination choice remains explicit.
6. As a user, I want `--no-shorten` to disable the configured default for one invocation, so that I can obtain the direct upload URL.
7. As a user, I want using `--shortener` and `--no-shorten` together to be a usage error, so that flag order cannot hide a contradictory command.
8. As an automation author, I want flags to remain valid before or after the file operand, so that the existing CLI contract remains stable.
9. As a user with no default or flag selection, I want Upit not to read Shortener configuration, so that the new feature is optional.
10. As a user who selects a Shortener, I want missing, invalid, or unknown Shortener configuration rejected before upload, so that a known configuration mistake does not occur after an irreversible upload.
11. As a user, I want global configuration version 2 to express the optional default Shortener, so that the strict schema has an unambiguous version.
12. As a user, I want named Shorteners stored in a dedicated strict version-1 document, so that Uploader and Shortener request contracts remain distinct.
13. As a security-conscious user, I want the Shortener document protected by the same Unix file-permission policy as the Uploader document, so that API keys are not knowingly read from a broadly accessible file.
14. As a Windows user, I want Upit not to present a Unix-only permission remedy, so that diagnostics remain applicable to my platform.
15. As a Shortener user, I want the HTTP method, endpoint URL, headers, query parameters, and JSON data object to be configurable, so that established services can be described without provider-specific code.
16. As a Shortener user, I want request method and URL validation before upload, so that malformed requests fail without uploading the file.
17. As a Shortener user, I want static header and query values sent literally, so that credentials and provider parameters are predictable.
18. As a Shortener user, I want Upit to own `Content-Type: application/json`, so that request encoding cannot disagree with configured headers.
19. As a Shortener user, I want exactly one JSON string value equal to `{input}` replaced with the Original URL, so that the template has one clear destination field.
20. As a Shortener user, I want `{input}` inside a larger string, an object key, or a non-string value left unsupported, so that substitution cannot mutate unrelated data.
21. As a Shortener user, I want JSON objects and arrays inside the data object preserved, so that provider-specific optional fields remain expressible.
22. As a security-conscious user, I want Shortener header, query, body, and endpoint-query values scrubbed from diagnostics, so that reflected credentials are not exposed.
23. As a user, I want redirects rejected for Shortener requests, so that credentials and the Original URL are not forwarded to an unintended host.
24. As a user, I want Shortener responses bounded to 1 MiB, so that a faulty endpoint cannot consume unbounded memory.
25. As a user, I want only HTTP 2xx Shortener responses eligible for success extraction, so that error responses are not mistaken for short links.
26. As a user, I want RFC 9535 JSONPath response extraction, so that varied provider payloads use the same proven expression language as Uploaders.
27. As a user, I want the success extractor to select exactly one JSON string, so that missing, ambiguous, null, or non-string results are not guessed or coerced.
28. As a user, I want the extracted Final URL validated as absolute HTTP(S), so that successful output remains directly usable.
29. As a user, I want an optional error extractor, so that provider error text can produce an actionable warning when safe.
30. As a user, I want the Original URL preserved when shortening succeeds, so that I can distinguish the upload destination from its shortened alias.
31. As an automation author, I want the existing plain and JSON success shapes unchanged, so that v0.1 consumers do not need a new parser.
32. As a user, I want ordinary Shortener network, HTTP, parse, extraction, validation, and deadline failures to fall back to the Original URL, so that a completed upload still yields a usable result.
33. As a user, I want shortening fallback reported as `Warning: shorten URL: <sanitized error>` on stderr, so that the degraded result is visible without contaminating stdout.
34. As an automation author, I want fallback to exit 0, so that a usable uploaded URL remains a successful result.
35. As a user, I want Ctrl-C during shortening to exit 130 with stage `shortener` and no stdout, so that explicit interruption is not converted into success.
36. As a user, I want one `--timeout` budget to cover upload, shortening, and clipboard work, so that the flag keeps its existing whole-invocation meaning.
37. As a user, I want no automatic Shortener retries, so that an uncertain request outcome cannot create duplicate links or consume extra quota.
38. As a user, I want the clipboard to receive the Final URL after shortening or fallback, so that clipboard and stdout agree.
39. As a headless user, I want URL shortening to add no GUI, Wails, WebView, daemon, or resident-process dependency, so that server use remains unchanged.
40. As a maintainer, I want the complete upload-to-shortening flow covered through the same CLI seam as production, so that configuration, precedence, requests, outputs, warnings, and exit codes are proven together.

## Implementation Decisions

- v0.2 contains only optional URL shortening in the existing CLI. Additional upload body types and response parsers, configuration-management commands, release packaging, and desktop work remain deferred.
- The canonical domain term is Shortener: a named definition of a remote URL-shortening destination. It is distinct from an Uploader and executes only after upload success.
- Global configuration makes a clean cutover from version 1 to version 2. Version 2 retains the default Uploader and clipboard default and adds an optional default Shortener. Global version 1 is rejected; no compatibility reader or automatic migration is added.
- Named Shorteners live in a dedicated `custom-shortener.json` document with schema version 1. Its top-level collection is `shorteners`. Uploader configuration remains version 1 and does not gain Shortener fields.
- Shortener configuration is lazy. If no default or CLI Shortener is selected, or `--no-shorten` is present, the Shortener document is not opened or validated. When shortening is selected, the complete document is strict-decoded and every named Shortener is semantically validated before upload.
- Selection precedence is: contradictory `--shortener` plus `--no-shorten` is usage error exit 2; otherwise `--no-shorten` disables shortening; otherwise `--shortener` overrides the global default; otherwise the optional global default applies; otherwise no shortening occurs.
- The application workflow module continues to coordinate configuration, upload, optional post-processing, clipboard, warnings, and the final result behind its existing command-running seam. Shortening is added behind that deep module rather than exposed as a new CLI-facing interface.
- The Shortener HTTP implementation remains concrete. No interface, provider adapter, factory, or public request-builder abstraction is introduced because v0.2 has one protocol and one production implementation.
- A Shortener request defines a required syntactically valid HTTP method, an absolute HTTP(S) endpoint, optional static headers, optional static query parameters, and a required JSON object named `data`. There is no constant `body` discriminator.
- The configured data tree must contain exactly one string value whose complete content is `{input}`. Upit recursively copies the object and arrays, replacing only that value with the Original URL. Substrings, object keys, and non-string values are not substituted.
- Upit JSON-encodes the substituted data and owns `Content-Type: application/json`. A configured Content-Type is rejected. Other headers and query values are literal.
- The existing redirect policy, concrete configurable HTTP client, 1 MiB response limit, 2xx-only success policy, JSON decoding, RFC 9535 implementation, exactly-one-string extraction rule, optional error extraction, and absolute HTTP(S) URL validation are reused for Shorteners.
- Shortener requests are attempted once. No automatic retry or backoff is added.
- Shortener configuration may contain credentials. On Unix, the file must have no group or other permission bits and failure guidance uses `chmod 600`; POSIX mode checks are not applied on Windows.
- Diagnostics scrub every non-empty configured header value, query value, string data value, and endpoint query value. Endpoint-provided error text is sanitized before it can appear in a warning.
- Missing Shortener files, unsupported versions, unknown fields, invalid definitions, absent selected names, malformed templates, and permission failures are configuration or validation errors before upload and exit 1.
- After upload success, ordinary Shortener request construction, network, non-2xx response, oversized response, JSON parsing, extraction, Final URL validation, and context-deadline errors preserve the Original URL as the Final URL. They append one warning with the exact prefix `shorten URL: ` and preserve exit 0.
- A parent-context cancellation during shortening is not fallback. It returns a failure with stage `shortener`; the CLI preserves the existing interruption contract of exit 130 and empty stdout.
- One timeout context continues to wrap the complete application workflow. A Shortener receives only the time remaining after upload. A deadline reached during shortening is a fallback warning; an explicit parent cancellation is interruption.
- Successful shortening updates only Final URL. Original URL and the upload HTTP status remain preserved. Fallback leaves Original URL and Final URL equal.
- Clipboard processing remains after shortening. It receives the Final URL, including the Original URL when fallback occurs. Clipboard failures retain their existing nonfatal warning behavior.
- Plain success output remains exactly the Final URL and a newline. JSON success remains exactly `success`, `originalUrl`, and `finalUrl`. Shortener fallback does not add fields to success JSON and emits its human-readable warning on stderr in both output modes.
- CLI help adds Shortener selection and disable flags while preserving interspersed flag parsing and every existing flag and exit code.

## Testing Decisions

- The primary permanent seam remains the CLI command-running interface. Tests invoke it with arguments, stdout/stderr buffers, a temporary home directory, real versioned configuration, and a temporary upload file.
- End-to-end Shortener tests use two real local HTTP endpoints: the upload endpoint returns a controlled Original URL, and the Shortener endpoint inspects the actual method, query, headers, JSON body, and substituted Original URL before returning controlled success or failure responses.
- Existing v0.1 fixtures and examples move to global configuration version 2. Tests prove that global version 1 is rejected and Uploader configuration remains version 1.
- CLI tests cover no-selection lazy loading, configured default selection, per-invocation override, disable override, contradictory flags, missing selected names, missing files, strict unknown-field rejection, and Unix permission rejection before the upload endpoint is called.
- Request tests cover configurable methods, endpoint query merging, static headers and query values, automatic JSON Content-Type, configured Content-Type rejection, recursive nested object/array preservation, and exactly one full-string `{input}` replacement.
- Focused lower-level tests are limited to template substitution invariants that are cumbersome to enumerate through the CLI seam: zero placeholders, multiple placeholders, substring non-matches, key non-matches, and non-string values.
- Response tests cover redirect rejection, non-2xx fallback, extracted provider errors, oversized responses, invalid JSON, zero/multiple/non-string JSONPath results, invalid Final URLs, and diagnostic redaction.
- Workflow tests prove successful Original/Final URL divergence, runtime fallback equality, warning placement, unchanged JSON success shape, clipboard use of Final URL, no retries, whole-invocation timeout behavior, and Ctrl-C during shortening returning stage `shortener` and exit 130.
- Completion requires an actual binary smoke against local upload and Shortener endpoints. The smoke observes the shortened Final URL, preserved Original URL in JSON mode, process exit, and absence of a resident Upit process.

## Out of Scope

- Binary, form, or JSON Uploader bodies and new Uploader response extractors.
- Header, regex, raw-body, XML, or response-URL Shortener extractors.
- Provider-specific Kutt, Bitly, Shlink, Dub, or authentication adapters.
- Placeholder substitution in endpoint URLs, query parameters, headers, object keys, string substrings, or non-string JSON values.
- Multiple placeholders or status-specific/fallback extractor lists.
- Structured warning fields in success JSON or JSON warning objects on stderr.
- Configurable fallback policy, retry count, backoff, or Shortener-specific timeout.
- Configuration create/edit/migrate commands, automatic migration, JSON Schema publication, environment-variable substitution, and keychain integration.
- Upload history, Shortener history, persistence, telemetry, progress display, or metrics.
- Wails, WebView, desktop UI, system tray, notifications, shell integration, daemon, or resident processes.
- Release archives, installers, container images, and automated publishing.

## Further Notes

- The four researched first-party APIs—Kutt, Bitly, Shlink, and Dub—can express their basic shortening operation with configurable method/endpoint, static headers, a JSON body containing one Original URL field, and RFC 9535 JSONPath response extraction. The evidence is recorded in the v0.2 research note.
- The dedicated Shortener configuration and runtime fallback decisions are recorded in ADRs. Domain language is recorded in `CONTEXT.md`.
- The existing v0.1 specification remains authoritative for upload behavior not explicitly changed here.
- Implementation remains unapproved until the ticket breakdown is accepted by the user.
