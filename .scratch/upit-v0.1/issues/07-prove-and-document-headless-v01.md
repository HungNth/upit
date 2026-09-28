# 07: Prove and document the complete headless v0.1 flow

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Turn the completed slices into a coherent v0.1 user experience with accurate examples, CLI help, build proof, and an actual end-to-end smoke against a local upload endpoint. Documentation must describe only behavior that exists and must remove conflicting future fields from v0.1 examples.

**Blocked by:** 04 — Stabilize the automation output contract; 05 — Cancel and time-limit uploads safely; 06 — Copy successful results to the clipboard optionally

**Status:** ready-for-agent

- [ ] User documentation explains installation/build, the fixed configuration location on every OS, required Unix permissions, Uploader selection, all v0.1 flags, stdout/stderr behavior, exit codes, timeout, cancellation, and clipboard warnings.
- [ ] The version-1 example configuration delivered by ticket 02 remains accurate, contains no real credentials or future fields, and passes the same strict loader used by the CLI.
- [ ] Broader architecture documentation no longer presents future shortener, open-after-upload, schema, or desktop fields as executable v0.1 behavior.
- [ ] CLI help matches the implemented syntax and accepts flags before or after the file operand.
- [ ] The native CLI build succeeds without Wails or WebView dependencies, and headless Linux and Windows cross-builds succeed from the same Go module.
- [ ] The permanent test suite passes and remains centered on the approved CLI seam rather than implementation-specific mocks.
- [ ] An actual built binary uploads a temporary file to a local endpoint using a mode-0600 Uploader file, emits the expected plain and JSON results, and exits completely.
- [ ] The smoke observes no resident Upit process after success, failure, or cancellation.
- [ ] No URL shortener, GUI, daemon, tray, watcher, automatic retry, multi-file support, or release packaging is introduced.
