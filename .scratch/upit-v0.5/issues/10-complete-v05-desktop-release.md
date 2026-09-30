# 10: Complete and demonstrate v0.5 Desktop Configuration and Manual Upload

**What to build:** Deliver the documented v0.5 desktop product as a cross-platform native runnable release surface. Complete accessibility, documentation, native build, regression, and smoke evidence after Setup, Repair, Configuration Set editing, and Manual Upload are working. This ticket proves the finished product without pulling deferred installers, signing, updates, or OS integrations into scope.

**Blocked by:** 06: Create the first Configuration Set; 07: Repair invalid Configuration Sets; 09: Observe and cancel Manual Upload

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Reuse the completed Wails-free application tests and CLI regression suite. This ticket adds actual built-application verification, platform-native build/smoke evidence, accessibility checks, and documentation accuracy; it does not add a duplicate implementation-level test suite.

**Demo path:** On each available native target, launch `upit-desktop`, complete first-run Setup, edit Global Configuration and definitions, repair an invalid document, run one successful local Manual Upload with progress and Copy, demonstrate failure plus explicit Retry and cancellation, verify close/single-instance behavior, then run the unchanged CLI against the same Configuration Set.

- [x] README, build instructions, architecture documentation, domain terminology, Configuration Set documentation, Wails prerequisites, desktop limitations, and separate executable behavior describe the delivered v0.5 product accurately.
- [x] Documentation identifies exact Wails `v3.0.0-beta.26`, Go 1.25, Vue/TypeScript/Vite/shadcn-vue, Windows/macOS/GTK4-WebKitGTK 6.0 Linux scope, and deferred installer/signing/update/integration scope without claiming Wails 3 stable.
- [ ] Keyboard navigation, visible focus, semantic labels, dialog focus handling, validation associations, progress announcements, and non-color-only status are checked on the running desktop surface.
- [x] Full Go tests, Go vet, frontend typecheck, production frontend build, existing CLI builds, and desktop builds available on host environments pass.
- [ ] Native Windows runtime smoke proves WebView2 desktop launch, picker/drop, clipboard behavior where available, single-instance focus, native replacement, Manual Upload lifecycle, and complete exit.
- [ ] Native Linux GTK4/WebKitGTK 6.0 runtime smoke proves desktop launch, picker/drop, publication, Manual Upload lifecycle, and complete exit. Older-Linux desktop non-support remains documented while CLI support remains intact.
- [ ] Native macOS runtime smoke is performed before claiming verified macOS runtime delivery; when unavailable, Darwin build is compile-only and the limitation is explicitly reported.
- [ ] The full desktop demo covers Setup, valid normal state, structured Uploader/Shortener lifecycle, Repair, Global Configuration Save, stale conflict, one successful local upload, warning, explicit Retry, cancellation, navigation while active, dirty close, and single-instance focus.
- [ ] The same created/edited Configuration Set remains valid for `upit config validate` and a local CLI upload, proving no frontend-only configuration contract exists.
- [ ] No installer, signing, notarization, auto-update, package-manager manifest, shell/OS integration, watcher, tray process, queue, history, backup, lock, transaction journal, compatibility shim, placeholder, or unfinished delivery scaffold remains.

## Comments

- Current host evidence: full Go and race suites, `go vet`, production Vue build, CLI cross-builds, Darwin desktop build, and a live macOS Wails asset-serving smoke pass. The GUI process correctly remains running until the harness timeout.
- Windows WebView2 and Linux GTK4/WebKitGTK 6.0 hosts are unavailable here; their required native picker/drop, Manual Upload lifecycle, single-instance, replacement, and exit smokes remain unverified.
- The running macOS process served the embedded frontend assets. Native WebView control automation is unavailable in this environment, so the full interactive macOS demo and accessibility walkthrough remain unverified.
