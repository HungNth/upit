# 01: Deliver one-shot File Manager Upload

**What to build:** Establish the shared Wails-free File Manager Upload behavior that a native file-manager adapter can invoke for one regular file. It must revalidate the current Configuration Set, apply default Uploader, optional default Shortener, and configured clipboard behavior; maintain one active operation; expose privacy-minimized progress and cancellation; and produce the contextual recovery behavior needed by native notifications without opening Upit Desktop or parsing CLI output.

**Blocked by:** v0.5 / 10: Complete and demonstrate v0.5 Desktop Configuration and Manual Upload

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the File Manager Upload application module with temporary homes, real Configuration Set documents, local HTTP endpoints, deterministic clipboard and notification adapters, and controlled contexts/readers. Do not create a second HTTP/configuration implementation or test generated bindings.

**Demo path:** Invoke the helper with one local regular file and default configuration against a local endpoint; observe sanitized preparation/progress/outcome, cancel a slow request, reject a second simultaneous invocation, use Copy and explicit Retry through opaque action tokens, and observe all temporary action state removed on use or ten minutes after completion.

- [x] Exactly one regular file reaches the shared operation; empty, multiple, directory, non-regular, missing, and changed paths make no endpoint request.
- [x] The operation reloads and strictly validates the current Configuration Set, uses default Uploader/Shortener/clipboard behavior with no overrides or added timeout, and retains an immutable configuration snapshot for one attempt.
- [x] One active File Manager Upload is enforced per user; a second invocation is rejected without queuing, concurrency, a daemon, or endpoint activity.
- [x] Ordered preparation, transfer bytes where measurable, response, Shortener, and clipboard progress is cancellable and contains no file path/name, endpoint, request value, response data, Original URL, Final URL, or credential.
- [x] Success, Shortener fallback, clipboard warning, failure, and cancellation preserve the existing application semantics; configuration failures expose Open Upit Desktop and runtime failures permit exactly one explicit Retry after fresh preflight.
- [x] Copy Final URL and Retry use opaque tokens with minimal private action state; state is consumed on use or deleted ten minutes after completion and never becomes upload history or logs private values.
- [x] Deterministic tests prove the complete behavior at the application seam and preserve the existing CLI upload contract.

## Comments

- Blocker note: v0.5 ticket 10 retains pending OS runtime smokes (Windows/Linux GUI smoke and full demo path), but core Wails-free desktop configuration, manual upload, and CLI regression are complete and accepted as non-blocking for headless File Manager Upload.
- Automated tests passing at the application seam: all 11 test cases in `internal/app/file_manager_upload_test.go` and 5 test cases in `cmd/upit-file-manager/main_test.go`. CLI contract regression tests verified via `go test ./...`.
