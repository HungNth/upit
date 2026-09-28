# Upit v0.1 — Headless Upload CLI

Status: ready-for-agent

## Problem Statement

Users need a small cross-platform command that uploads one file to a configurable HTTP endpoint, returns the uploaded file URL in a script-safe form, and exits. Existing upload tools often require a GUI, a resident process, a fixed provider, or enough runtime integration that they are unsuitable for SSH sessions, CI, containers, and headless Linux servers.

Upit currently has an architecture document but no executable application. The architecture also mixes the long-term product with later roadmap items. This specification defines the complete v0.1 contract: a headless CLI supporting streaming multipart uploads and RFC 9535 JSONPath response extraction, without URL shortening or a desktop application.

## Solution

Build the `upit` CLI as an on-demand Go application. A user configures one or more named Uploaders, selects one by default or per invocation, and runs `upit upload` with one file path. Upit streams the file as multipart HTTP, extracts an absolute HTTP(S) URL from a bounded JSON response, optionally copies that URL to the clipboard, writes a stable result to stdout, and exits.

The CLI remains independent of Wails, WebView, desktop sessions, and resident processes. Configuration is strict and user-scoped. Credentials remain in the Uploader configuration file, which is protected by Unix file permissions and conservative diagnostic redaction.

## User Stories

1. As a command-line user, I want to upload one file with `upit upload`, so that I can receive a shareable URL without opening a GUI.
2. As a headless Linux user, I want Upit to run without X11, Wayland, GTK, WebView, or a desktop session, so that it works over SSH and on servers.
3. As an automation author, I want a successful plain invocation to print only the Final URL to stdout, so that command substitution and pipelines remain reliable.
4. As an automation author, I want `--json` success output with `success`, `originalUrl`, and `finalUrl`, so that scripts can consume a stable structured result.
5. As an automation author, I want `--json` failures to produce a structured object on stderr and a nonzero exit code, so that I can distinguish result data from diagnostics.
6. As a shell user, I want usage, runtime, and interruption outcomes to have conventional exit codes, so that scripts can react without parsing prose.
7. As a user, I want flags to work before or after the file path, so that the documented examples and common CLI styles both work.
8. As a user with several upload destinations, I want multiple named Uploaders, so that one installation can target different services.
9. As a user, I want a default Uploader in global configuration, so that routine uploads do not require a repeated flag.
10. As a user, I want `--uploader` to override the configured default for one invocation, so that temporary destination changes are explicit.
11. As a user, I want missing configuration to report the exact expected location and point me to examples, so that setup failures are actionable.
12. As a maintainer, I want both configuration documents to declare version 1 and reject unknown fields, so that typos and unsupported future options fail early.
13. As a cross-platform user, I want Upit configuration at `~/.config/upit/` on every operating system, so that deployment instructions and automation use one location.
14. As a security-conscious user, I want the credential-bearing Uploader file rejected on Unix when group or other permissions are present, so that Upit does not knowingly use a broadly readable secret file.
15. As a security-conscious user, I want request configuration values treated as sensitive in diagnostics, so that headers, query values, multipart fields, and URLs containing queries are not exposed.
16. As a user of a custom HTTP endpoint, I want the request method, URL, headers, query parameters, multipart file field, and static multipart fields to be configurable, so that common upload APIs can be described without hardcoded provider logic.
17. As a user, I want static request values sent literally in v0.1, so that configuration behavior is predictable and does not imply an undocumented template language.
18. As a user uploading large files, I want multipart data streamed from disk, so that memory use does not grow with file size.
19. As a user, I want Ctrl-C to cancel an in-flight upload and release its resources, so that I can stop slow or mistaken requests promptly.
20. As an automation author, I want an optional `--timeout` using Go duration syntax and no default overall deadline, so that each workflow controls its own maximum upload time.
21. As a security-conscious user, I want HTTP redirects rejected, so that a streaming upload is not replayed or redirected to an unintended host.
22. As a user, I want only HTTP 2xx responses treated as upload success, so that error and redirect responses cannot be mistaken for completed uploads.
23. As a user, I want response bodies limited to 1 MiB, so that a faulty or hostile endpoint cannot make response parsing consume unbounded memory.
24. As a user of varied JSON responses, I want RFC 9535 JSONPath expressions for URL and error extraction, so that nested objects, arrays, wildcards, and filters are available through configuration.
25. As a user, I want a required extractor to select exactly one JSON string, so that missing, ambiguous, null, and non-string results fail instead of being guessed or coerced.
26. As a user, I want the extracted upload URL validated as an absolute HTTP(S) URL, so that success always returns a directly usable link.
27. As a future URL-shortener user, I want v0.1 to preserve both Original URL and Final URL even though they are equal, so that the result contract does not change when post-processing is added later.
28. As a desktop user, I want clipboard copying to be opt-in through configuration or CLI flags, so that headless execution has no default desktop side effect.
29. As a user who enabled clipboard copying, I want upload success preserved when clipboard access is unavailable, so that I still receive the URL and an actionable warning.
30. As a resource-conscious user, I want every invocation to exit after completing or failing, so that Upit has no idle process or background memory use.
31. As a container user, I want the CLI build to exclude Wails and WebView dependencies, so that the binary can run in minimal headless images.
32. As a maintainer, I want server-provided error text sanitized before display, so that configured request values cannot be reflected back as leaked credentials.

## Implementation Decisions

- The supported runtime is Go 1.23 or newer.
- The v0.1 product contains one executable CLI and no desktop executable. The CLI must not import or initialize Wails, WebView, or GUI packages.
- The CLI module owns argument parsing, help and usage behavior, stdout/stderr formatting, and exit-code mapping. It exposes one high-leverage command-running seam so callers and tests exercise the same interface.
- The application workflow module coordinates configuration loading, Uploader selection, upload execution, optional clipboard copying, and warning collection. It returns data; it does not print or terminate the process.
- The upload module is deep: it owns v0.1 multipart request construction, streaming, HTTP execution, bounded response reading, JSON decoding, JSONPath extraction, URL validation, and structured upload errors behind one upload interface. Separate public request-builder and response-parser modules are deferred until a second protocol or parser creates real variation.
- No HTTP-client interface is introduced. The upload module accepts a concrete configurable HTTP client; tests use a real local HTTP server instead of a mocked transport.
- Clipboard access is a real seam because production and deterministic test adapters both exist. The production adapter invokes the operating system's available clipboard command without linking GUI toolkits; absence or failure becomes a warning, not an upload error.
- Configuration is resolved from the current user's home directory and always uses the `.config/upit` child directory, including on macOS and Windows.
- Global configuration version 1 contains only the default Uploader name and the clipboard default. Open-after-upload and URL-shortener settings are invalid in v0.1.
- Uploader configuration version 1 contains only named Uploaders. URL-shortener definitions are invalid in v0.1.
- An Uploader request defines method, absolute HTTP(S) endpoint URL, optional headers, optional query parameters, multipart body type, file-field name, and optional static multipart fields. All configured values are literal strings.
- The upload implementation owns the multipart `Content-Type` boundary. A configured request must not override that header.
- The URL response extractor is required. The error extractor is optional and is consulted for endpoint failures or unsuccessful extraction when usable JSON is available.
- Both configuration documents use strict decoding and semantic validation before any network request. Invalid versions, unknown fields, absent defaults, missing Uploaders, unsupported body types, invalid methods or URLs, empty file fields, and invalid JSONPath expressions fail at the configuration or validation stage.
- The Uploader file stores credentials directly. Upit adds no environment-variable substitution, keychain integration, or secret metadata in v0.1.
- On Unix, the Uploader file must have no group or other permission bits. Upit reports an actionable `chmod 600` correction and performs no network request when the check fails. POSIX mode enforcement is not applied on Windows.
- Diagnostics never print request headers, query strings, multipart field values, or a request URL containing its query. Every non-empty configured header, query, and field value is scrubbed from endpoint-provided error text before output.
- Multipart upload data is streamed from an open file through the HTTP request. The complete file is never read into memory.
- One file path is accepted per invocation. Directories and unreadable files fail validation before the request starts.
- A signal-aware context handles interruption. `--timeout` accepts Go duration syntax; zero means no overall deadline. Cancellation closes files, response bodies, and multipart pipe endpoints promptly.
- Automatic HTTP redirects are disabled. Any 3xx response is an unsuccessful endpoint response.
- Only 2xx responses continue to success extraction. Response bodies are limited to 1 MiB after transport decoding; exceeding the limit is a response-stage error.
- JSON responses are decoded into Go's standard generic JSON tree.
- RFC 9535 expressions are implemented by `github.com/theory/jsonpath` pinned to `v0.12.1`. Full RFC selectors are accepted.
- Every configured extractor must select exactly one node and that node must be a JSON string. Zero matches, multiple matches, JSON null, objects, arrays, booleans, and numbers are distinct extraction errors; no value is implicitly stringified.
- A successful extracted URL must be absolute and use `http` or `https`.
- v0.1 results contain Original URL, Final URL, and HTTP status code. Original URL and Final URL are equal. Thumbnail and deletion URLs are not represented until corresponding extractors exist.
- Plain success output is exactly the Final URL followed by a newline on stdout.
- JSON success output is one object on stdout containing `success: true`, `originalUrl`, and `finalUrl`.
- Plain failures are human-readable on stderr. With `--json`, failures are one object on stderr containing `success: false`, `stage`, `message`, and `statusCode` only when an HTTP status is available. Stdout remains empty on failure.
- Error stages exposed by v0.1 are config, validation, request, network, response, and parse. Clipboard failures are successful-upload warnings rather than command failures.
- Exit codes are 0 for success, 1 for runtime failure, 2 for invalid command usage, and 130 for interruption.
- Clipboard copying is disabled by default. CLI flags override global configuration in both directions. Clipboard failure preserves exit code 0 and the normal success output, and adds a warning to stderr.
- Flags may appear before or after the file operand. The supported upload flags are Uploader override, JSON output, clipboard enable, clipboard disable, and timeout.
- No automatic request retry is performed. Retrying a streaming upload can duplicate server-side data and is outside the stated contract.
- The module path follows the repository identity. Dependencies are pinned in the Go module rather than fetched by floating tags.

## Testing Decisions

- The primary permanent seam is the CLI command-running interface. Tests provide arguments, stdout/stderr buffers, a temporary home directory, real temporary configuration and file data, and a local HTTP test server. Assertions target exit codes and externally visible output rather than internal calls.
- The local HTTP server verifies the actual multipart request: method, query, headers, static fields, file-field name, filename, and streamed bytes. It returns controlled status codes and JSON bodies to exercise extraction and error behavior.
- The same CLI seam covers default and overridden Uploader selection, strict configuration failures, missing files, Unix permission rejection, flag positions, timeout, interruption, response-size rejection, redirect rejection, RFC 9535 array/filter expressions, zero/multiple/type mismatches, URL validation, plain output, JSON output, redaction, and exit-code mapping.
- Clipboard behavior uses a deterministic test adapter at the clipboard seam. Tests prove opt-in behavior, CLI-over-config precedence, success copying, and nonfatal warning behavior without depending on a real desktop session.
- Focused lower-level tests are added only for invariants that cannot be observed reliably through the CLI seam. They must test behavior, not package wiring, forwarding, source text, or implementation-specific defaults.
- No prior test conventions exist in the repository. The initial test layout should keep fixtures local, deterministic, isolated, and safe for parallel full-suite execution.
- Completion requires an actual binary smoke run against a local upload endpoint, in addition to the permanent tests. The smoke must observe the final URL, process exit, and absence of any resident Upit process.

## Out of Scope

- URL shortening and `{input}` substitution.
- Binary, form, or JSON upload bodies.
- Header, regex, raw-body, XML, or response-URL extractors.
- Multiple files or concurrent uploads.
- Automatic retries or resumable uploads.
- Upload progress rendering.
- Configuration creation, editing, migration commands, or automatic bootstrap.
- Published JSON Schema documents.
- Environment-variable substitution, operating-system keychains, or per-field secret metadata.
- Open-after-upload behavior.
- Upload history or persistence of results.
- Notifications, system tray, daemon, file watcher, resident worker, or background event loop.
- Wails desktop application, desktop settings editor, and the Wails version decision.
- Explorer, Finder, launcher, shell, or desktop integration.
- Container images, installers, cross-platform release archives, and release automation.
- Thumbnail and deletion URL extraction.
- Compatibility shims for future configuration versions.

## Further Notes

- This specification is normative for v0.1 where the broader architecture document describes later roadmap capabilities.
- Domain language is recorded in `CONTEXT.md`.
- The fixed configuration directory, plaintext credential file, and RFC 9535 implementation decisions are recorded in ADRs.
- Primary-source research for established custom-uploader contracts, Go configuration conventions, and the JSONPath library selection is stored with this effort.
- Implementation remains unapproved until the ticket breakdown is accepted by the user.
