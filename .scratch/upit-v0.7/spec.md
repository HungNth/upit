# Upit v0.7 — macOS File Manager Upload

Status: ready-for-agent

## Problem Statement

A macOS user can use the headless CLI or deliberately open Upit Desktop for Manual Upload, but Finder has no integrated command to upload a selected file. Finder extensions that offer a first-level contextual item are sandboxed and cannot noninteractively read Upit’s fixed Configuration Set. The user needs a native Finder action that runs one default upload without showing Upit Desktop, gives private and actionable feedback, and is distributed through a signed and notarized macOS package.

## Solution

Deliver a Developer ID-signed and notarized macOS 14+ Apple Silicon desktop package containing a background-only `NSServices` helper. Finder exposes `Upload with Upit` under Services or Quick Actions; the helper receives exactly one selected file URL, invokes the shared one-shot Wails-free File Manager Upload behavior, and exits without creating a Wails window, Dock icon, tray process, or resident worker.

The helper preserves the default Uploader, optional default Shortener, clipboard, privacy, progress, cancellation, recovery-action, and single-flight contracts established by v0.6. The package remains outside the Mac App Store sandbox so it can keep using the fixed `~/.config/upit/` Configuration Set noninteractively. Installation registers the Finder Service; uninstallation removes it. Production releases are signed, notarized, stapled, and published only from protected SemVer tags.

## User Stories

1. As a macOS user, I want `Upload with Upit` available from Finder Services or Quick Actions for one selected file, so that I can start an upload without a terminal or an Upit Desktop window.
2. As a user, I want Finder to reject a selection that is empty, multiple, a directory, or not a regular file, so that File Manager Upload retains its one-file contract.
3. As a user, I want the selected Finder file URL converted and validated before network activity, so that a native pasteboard handoff cannot bypass file rules.
4. As a user, I want File Manager Upload to use my default Uploader, so that Finder behavior matches an ordinary default CLI invocation.
5. As a user, I want File Manager Upload to use my optional default Shortener, so that Finder produces the same Final URL policy as the Configuration Set.
6. As a user, I want File Manager Upload to honor my configured clipboard behavior, so that the Finder action does not invent a separate default.
7. As a user, I want no File Manager Upload override form or window, so that Manual Upload remains the explicit interactive surface for picker input, drag-and-drop, and temporary choices.
8. As a user, I want the current Configuration Set reloaded and strictly validated before work starts, so that Finder does not act on stale, incomplete, or invalid configuration.
9. As a user, I want the background helper to start no Wails window, Dock icon, system tray item, or resident process, so that a right-click action remains invisible except for its native feedback.
10. As a user, I want native feedback when preparation starts, so that I know Finder accepted the action.
11. As a user, I want phase and measurable byte progress in native feedback, so that I can understand long-running work without Upit Desktop.
12. As a user, I want Cancel available while the upload is active, so that I can stop unwanted work without Activity Monitor.
13. As a user, I want only one active File Manager Upload, so that clipboard writes, progress, and notifications cannot race.
14. As a user, I want a second Finder action while one is active to report a rejected invocation without making a request, so that Upit never creates a hidden queue.
15. As a privacy-conscious user, I want native feedback to omit local paths, file names, endpoints, request values, response bodies, Original URLs, Final URLs, and credentials, so that lock-screen and Notification Center content is safe by default.
16. As a user whose clipboard default is enabled, I want the Final URL copied after success, so that it is ready to paste.
17. As a user whose clipboard default is disabled, I want a Copy Final URL notification action, so that I can copy deliberately without displaying the URL in notification text.
18. As a user, I want a Shortener fallback or automatic clipboard failure reported as a warning rather than an upload failure, so that the original successful upload is preserved.
19. As a user whose Configuration Set needs Setup or Repair, I want a sanitized failure notification with Open Upit Desktop, so that I can correct it through the desktop product rather than Finder.
20. As a user whose runtime upload fails, I want an explicit Retry action, so that I can start one fresh attempt only after I have corrected the cause.
21. As a privacy-conscious user, I want Copy and Retry to resolve through opaque notification tokens and minimal temporary private state, so that paths and URLs are neither shown nor retained as history.
22. As a privacy-conscious user, I want temporary action state deleted after use or ten minutes after completion, so that native recovery does not become persistent upload retention.
23. As a user denied notification permission or affected by notification failure, I want a native alert for cancellation, failure, or an un-copied success, so that the action never becomes silent.
24. As a macOS user, I want installation to register the Finder Service automatically, so that I do not need to create an Automator workflow or edit system files.
25. As a macOS user, I want clear guidance when macOS requires me to enable the Service in System Settings, so that I can discover the installed action.
26. As a macOS user, I want uninstallation to unregister the Finder Service, so that no stale Upit entry remains in Finder.
27. As a security-conscious user, I want the app and nested helper signed, notarized, and stapled, so that Gatekeeper can verify the downloaded package.
28. As a user, I want the integration to keep the fixed Configuration Set location unchanged, so that Finder, Desktop, and CLI read the same settings.
29. As a maintainer, I want the Finder Service adapter to contain no configuration, upload, or notification policy, so that the shared Wails-free implementation remains authoritative.
30. As a release maintainer, I want notarized artifacts only from protected SemVer tags, so that release credentials stay outside ordinary builds.
31. As a maintainer, I want the current `upit` CLI and `upit-desktop` Manual Upload contracts unchanged, so that Finder integration does not merge product surfaces.

## Implementation Decisions

- v0.7 supports macOS 14+ on Apple Silicon. Intel Macs, older macOS releases, and Mac App Store distribution are not release targets.
- Finder integration uses an `NSServices` provider in a background-only helper bundled with the macOS desktop package. The user-visible action is `Upload with Upit` in Finder Services or Quick Actions; it is not required to occupy Finder’s top-level context menu.
- The Service receives one selected file URL through the native pasteboard. It rejects zero, multiple, directory, and non-regular selections before network work; selected URLs are untrusted input.
- The background helper delegates File Manager Upload to the shared Wails-free application module established in v0.6. It does not launch Wails, parse CLI output, duplicate upload rules, or leave a resident process after the operation or action window.
- File Manager Upload uses only the current Global Configuration defaults for Uploader, Shortener, and clipboard behavior. It has no overrides and inherits the existing no-timeout default.
- The one-active-operation, progress, cancellation, warning, final-result, native alert fallback, opaque action-token, short-lived private state, and privacy-minimized notification rules are identical to v0.6. Private action state has restrictive permissions, stores only action-required values, is consumed on use, and is deleted ten minutes after completion when unused.
- Copy Final URL, Open Upit Desktop, Retry, and Cancel are contextual native actions only. They do not render or log private values, auto-open Upit Desktop, add automatic retry, or create upload history.
- The package remains unsandboxed and outside the Mac App Store. App Sandbox, Finder Sync, Share Extensions, and Action Extensions would require a different Configuration Set access model and are rejected.
- Installation registers the Service with Launch Services and documents any user-controlled System Settings enablement. Uninstall removes the app and unregisters the Service.
- The package contains the desktop application and background helper. Every executable is Developer ID signed with Hardened Runtime, the final package is notarized and stapled, and protected-tag CI publishes checksums and artifacts. Normal CI builds unsigned verification artifacts and never receives signing credentials.

## Testing Decisions

- The Wails-free File Manager Upload operation is the permanent behavior seam, reused from v0.6. Tests use temporary homes, real Configuration Set documents, real local HTTP endpoints, deterministic clipboard/notification adapters, and controllable contexts/readers.
- Tests prove the default selection, Configuration Set preflight, one regular file behavior, no request on rejected input, progress, cancellation, single-flight rejection, Shortener fallback, clipboard warning, sanitized failure, Copy, Retry, Open Upit Desktop, and native alert fallback contracts.
- Tests prove all notification and temporary action-state payloads exclude file paths, names, URLs, endpoints, request data, response data, and credentials; private state is consumed or deleted ten minutes after completion and never becomes history.
- The NSServices adapter receives narrow native tests for pasteboard file-URL translation, exact-one selection, launch-without-Wails behavior, cancellation/action dispatch, and registration/unregistration. It does not duplicate the shared upload tests.
- Native macOS 14+ Apple Silicon smoke installs the signed, notarized, stapled package; observes Finder Services or Quick Actions discovery; performs local-endpoint success, warning, failure, Copy, Retry, configuration recovery, progress, cancellation, no-window behavior, and uninstall cleanup; then proves the unchanged CLI against the same Configuration Set.
- CI runs the Go, frontend, package, signature, checksum, and native-smoke checks appropriate to the event. Only protected SemVer tags can access Developer ID/notarization credentials or publish artifacts.

## Out of Scope

- Opening Upit Desktop automatically, showing any Wails window, a Dock icon, a tray process, or a resident worker during File Manager Upload.
- Multi-file or directory upload, batch handling, queues, concurrent uploads, upload history, scheduling, automatic retry, picker input, drag-and-drop, per-upload overrides, or a new timeout policy.
- Intel Macs, macOS versions before 14, Windows File Explorer, Linux integration, launchers, browsers, mobile clients, or cloud integration.
- Finder Sync, Share Extensions, Action Extensions, Mac App Store distribution, App Sandbox, App Groups, security-scoped bookmarks, Automator workflows, or a first-level Finder context-menu requirement.
- Changing the fixed Configuration Set location, adding Global Configuration fields, schema migrations, credential stores, upload protocol features, or parsing CLI output as integration IPC.
- Rendering or logging file paths, names, URLs, endpoints, request values, response contents, or credentials in Notification Center, alerts, logs, or history.

## Further Notes

- v0.7 follows v0.6 so both native platforms share one File Manager Upload implementation contract rather than creating separate upload paths.
- ADR 0013 records the release sequence; ADR 0014 records the one-shot helper; ADR 0016 records the NSServices route and sandbox decision.
- Primary-source research for Finder Services, sandbox constraints, signing, notarization, and Wails packaging is retained in `research/macos-finder-integration.md`.
