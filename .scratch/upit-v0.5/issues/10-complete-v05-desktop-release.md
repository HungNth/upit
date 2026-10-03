# 10: Complete and demonstrate v0.5 Desktop Configuration and Manual Upload

**What to build:** Deliver the documented v0.5 desktop product as a cross-platform native runnable release surface. Complete accessibility, documentation, native build, regression, and smoke evidence after Setup, Repair, Configuration Set editing, and Manual Upload are working. This ticket proves the finished product without pulling deferred installers, signing, updates, or OS integrations into scope.

**Blocked by:** 06: Create the first Configuration Set; 07: Repair invalid Configuration Sets; 09: Observe and cancel Manual Upload

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Reuse the completed Wails-free application tests and CLI regression suite. This ticket adds actual built-application verification, platform-native build/smoke evidence, accessibility checks, and documentation accuracy; it does not add a duplicate implementation-level test suite.

**Demo path:** On each available native target, launch `upit-desktop`, complete first-run Setup, edit Global Configuration and definitions, repair an invalid document, run one successful local Manual Upload with progress and Copy, demonstrate failure plus explicit Retry and cancellation, verify close/single-instance behavior, then run the unchanged CLI against the same Configuration Set.

- [x] README, build instructions, architecture documentation, domain terminology, Configuration Set documentation, Wails prerequisites, desktop limitations, and separate executable behavior describe the delivered v0.5 product accurately.
- [x] Documentation identifies exact Wails `v3.0.0-beta.26`, Go 1.25, Vue/TypeScript/Vite/shadcn-vue, Windows/macOS/GTK4-WebKitGTK 6.0 Linux scope, and deferred installer/signing/update/integration scope without claiming Wails 3 stable.
- [x] Keyboard navigation, visible focus, semantic labels, dialog focus handling, validation associations, progress announcements, and non-color-only status are checked on the running desktop surface.
- [x] Full Go tests, Go vet, frontend typecheck, production frontend build, existing CLI builds, and desktop builds available on host environments pass.
- [ ] Native Windows runtime smoke proves WebView2 desktop launch, picker/drop, clipboard behavior where available, single-instance focus, native replacement, Manual Upload lifecycle, and complete exit.
- [ ] Native Linux GTK4/WebKitGTK 6.0 runtime smoke proves desktop launch, picker/drop, publication, Manual Upload lifecycle, and complete exit. Older-Linux desktop non-support remains documented while CLI support remains intact.
- [x] Native macOS runtime smoke is performed before claiming verified macOS runtime delivery; when unavailable, Darwin build is compile-only and the limitation is explicitly reported.
- [x] The full desktop demo covers Setup, valid normal state, structured Uploader/Shortener lifecycle, Repair, Global Configuration Save, stale conflict, one successful local upload, warning, explicit Retry, cancellation, navigation while active, dirty close, and single-instance focus.
- [x] The same created/edited Configuration Set remains valid for `upit config validate` and a local CLI upload, proving no frontend-only configuration contract exists.
- [ ] No installer, signing, notarization, auto-update, package-manager manifest, shell/OS integration, watcher, tray process, queue, history, backup, lock, transaction journal, compatibility shim, placeholder, or unfinished delivery scaffold remains.

## Comments

- Earlier host evidence covered Go/race/vet, Vue build, cross-builds, and live macOS asset serving. The later runs below explicitly observed clean native process exit rather than relying on a harness timeout.
- Windows WebView2 and Linux GTK4/WebKitGTK 6.0 hosts are unavailable here; their required native picker/drop, Manual Upload lifecycle, single-instance, replacement, and exit smokes remain unverified.
- Earlier macOS smoke covered launch, configuration, Manual Upload, and CLI regression; the subsequent evidence below completes stale conflict, dirty close, single-instance, and running-surface accessibility checks.
- 2026-10-03: the final actual macOS Wails WebView passed the native lifecycle regression, including URL/error extractor transitions, exact-name validation, Rename/Delete publication, keyboard focus containment/restoration, and validation-error associations. Native Setup audited accessible names across all 16 Request Body Mode/extractor combinations. Uploaders/Shorteners/Global Configuration had no unnamed visible controls; focused navigation had a visible 3px outline. Dirty-navigation, dirty-close, and active-upload-close dialogs used safe initial focus, contained Tab/Shift+Tab, and handled Escape. Real progress had polite live announcement and explicit text/byte status rather than color alone.
- Native demo evidence: Setup-created documents, normal state, structured Uploader/Shortener Save/Rename/Delete, Repair unlock/kind-change/session-close/reopen relock, all three Global Configuration fields, dirty Save and stale no-write rejection, local success/fallback/clipboard-warning/failure/Retry, progress/navigation/cancellation, confirmed active close and dirty close, second-instance foreground focus, actual system light/dark response with appearance restored, and clean process exits 0. Every native run used an isolated temporary HOME; the existing file-drop event was injected, so OS picker/drop delivery is not claimed. Actual CLI uploads proved all four native-edited body modes and all four extractors.
- Final host checks passed: `go test -race ./...`, `go vet ./...`, Vue typecheck/production build, ordinary CLI/desktop builds, and the separate MCP-tagged native regression binary. Production builds do not enable MCP.
- Remaining blockers: required native Windows and Linux command/publication/picker/drop/runtime evidence is unavailable without those runners. Criterion 24's prohibition on installer/signing/shell integration conflicts with the approved, already-implemented v0.6/v0.7 scope; no later implementation was deleted to satisfy that historical release boundary. A planning disposition is required for that criterion before this ticket can be resolved.
