# 02: Stream binary uploads and extract URLs from headers

**Parent specification:** [Upit v0.3 — More Upload Protocols](../spec.md)

**What to build:** Let a version-2 Uploader stream the selected file as the complete HTTP request body and obtain the uploaded file URL from one configured response header. This is the first new protocol path and must remain behind the existing deep upload module and CLI seam.

**Blocked by:** 01: Cut over existing multipart Uploaders to schema v2

Status: resolved

- [x] `body: "binary"` sends the selected file bytes exactly once as the complete request body, with no multipart framing, file field, static fields, JSON wrapper, or Content-Disposition.
- [x] Binary upload remains bounded-memory and context-aware for large files, timeouts, and parent cancellation.
- [x] Binary requests default to exactly `application/octet-stream` without filename-extension MIME inference.
- [x] A non-empty configured binary Content-Type is accepted only when it is a valid media type and is sent as configured; other Request Body Modes still own and protect their generated Content-Type.
- [x] Binary mode rejects file-field, form-field, and JSON-data configuration before network access.
- [x] Any syntactically valid configured HTTP method remains accepted and sends the binary body.
- [x] `type: "header"` requires a valid non-empty header name and extracts exactly one non-empty case-insensitive response-header value; zero or multiple values fail at stage `parse`.
- [x] Header extraction works for required URL and optional provider error extraction; error extraction improves diagnostics only and cannot create success.
- [x] Header URL results still require absolute HTTP(S), only 2xx can succeed, redirects remain rejected, and the complete response body is still read and limited to 1 MiB before success.
- [x] Multipart plus JSONPath behavior remains green after the private request/extraction variation is introduced; no request/response subpackages, strategy interfaces, factories, or provider adapters become caller-facing interfaces.
- [x] CLI-seam tests assert raw bytes, absence of multipart headers, default and configured Content-Type, method/query/header handling, header cardinality, URL validation, error extraction, output shapes, and exit codes against a real local endpoint.
- [x] A demo invocation streams a binary file and prints the absolute URL supplied in one response header.

## Comments

- Implemented with raw streaming, header extraction, managed-header validation, body bounds, cancellation, and boundary coverage.
- Changes remain uncommitted pending separate commit authorization.
