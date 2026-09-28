# 01: Upload one file with the default Uploader

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Deliver the first complete Upit path: a user places valid version-1 configuration under the fixed user configuration directory, runs `upit upload` for one readable file, and receives the absolute uploaded-file URL on stdout. The request must stream a multipart file to the configured default Uploader and extract one JSON string through the pinned RFC 9535 implementation.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] A Go 1.23+ CLI module builds an `upit` executable without Wails, WebView, or GUI dependencies.
- [ ] `upit upload <file>` resolves the current user's fixed Upit configuration directory and loads version-1 global and Uploader configuration.
- [ ] One configured default Uploader can send a multipart request with its method, endpoint, and file-field name.
- [ ] File bytes are streamed from disk; the complete file is not buffered in memory.
- [ ] `github.com/theory/jsonpath` is pinned to `v0.12.1`, and a configured response path extracts exactly one JSON string for the successful path.
- [ ] The extracted value must be an absolute HTTP(S) URL.
- [ ] Success prints exactly the URL and a trailing newline to stdout, then the process exits.
- [ ] The primary CLI-seam test uses temporary configuration, a real temporary file, and a local HTTP server to verify multipart metadata and bytes end to end.
- [ ] A demo command against the local test endpoint proves the first vertical slice outside a mocked transport.
