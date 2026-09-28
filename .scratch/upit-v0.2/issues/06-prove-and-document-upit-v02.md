# 06: Prove and document Upit v0.2

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Ship a coherent, reproducible v0.2 user experience. Documentation and examples must describe the exact versioned configuration, selection precedence, request template, success/fallback behavior, security requirements, and headless operation that the executable actually provides.

**Blocked by:** 03: Apply the default and per-invocation Shortener overrides; 04: Fall back safely when runtime shortening fails; 05: Interrupt shortening and copy the resolved Final URL

Status: resolved

- [x] User documentation explains the global version-2 cutover, dedicated version-1 Shortener document, Unix permission requirement, default/override/disable flags, and unchanged no-selection behavior.
- [x] Credential-free examples include at least one established JSON Shortener shape with exactly one full-string `{input}` placeholder and strict RFC 9535 response extraction.
- [x] Documentation states that runtime Shortener failures fall back with exit 0 and a stderr warning, while invalid configuration fails before upload and Ctrl-C during shortening exits 130.
- [x] Architecture and domain documentation use the canonical Shortener, Original URL, and Final URL language and mark v0.2 decisions as implemented rather than proposed.
- [x] The complete permanent suite passes, including black-box upload-plus-Shortener flows, race-sensitive cancellation coverage, and unchanged v0.1 upload behavior under config v2.
- [x] Static analysis passes and native, Linux amd64, and Windows amd64 CLI builds succeed without Wails or WebView dependencies.
- [x] An actual-binary smoke uses local upload and Shortener endpoints to observe a shortened Final URL in plain mode and distinct Original/Final URLs in JSON mode.
- [x] A fallback smoke observes Original URL output, sanitized stderr warning, exit 0, process exit, and no resident Upit process.
- [x] No throwaway smoke fixtures, temporary servers, generated binaries, or obsolete version-1 global examples remain in the repository.
