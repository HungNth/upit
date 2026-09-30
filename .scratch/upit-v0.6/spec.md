# Upit v0.6 — Windows File Manager Upload

Status: ready-for-agent

## Problem Statement

A Windows user can upload from the headless CLI or deliberately open Upit Desktop for Manual Upload, but cannot invoke an upload naturally from File Explorer. A classic shell verb would be hidden under “Show more options” on Windows 11, while an upload implemented inside Explorer risks freezing the file manager and duplicating the validated upload contract. The user needs a first-class right-click action that uploads one selected file with the Global Configuration defaults, reports the outcome without opening Upit Desktop, and remains safe to install, uninstall, and release.

## Solution

Deliver a signed Windows 11 x64 desktop package that registers `Upload with Upit` in File Explorer’s primary context menu through a package-identity `IExplorerCommand`. The native Explorer adapter hands exactly one selected path to a one-shot Wails-free File Manager Upload helper. The helper performs the existing Configuration Set preflight and upload behavior with the default Uploader, optional default Shortener, and configured clipboard behavior; it reports privacy-minimized native progress and outcomes, supports explicit cancellation and retry, and exits without opening a Wails window or becoming resident.

The integration is packaged with Upit Desktop, is registered during install, and is removed during uninstall. Production artifacts are signed and published only from protected SemVer tags.

## User Stories

1. As a Windows 11 user, I want `Upload with Upit` in the primary File Explorer context menu for one selected file, so that I can start an upload without opening a terminal or Upit Desktop.
2. As a user, I want the action hidden when I select zero or multiple items, so that File Manager Upload remains exactly one-file work.
3. As a user, I want the action unavailable for folder selections, so that Upit does not imply directory or archive upload support.
4. As a user, I want the selected path treated as untrusted and revalidated as one regular file before network activity, so that a shell selection cannot bypass the existing file rules.
5. As a user, I want File Manager Upload to use the default Uploader, so that it matches an ordinary default CLI invocation.
6. As a user, I want File Manager Upload to use the optional default Shortener, so that its Final URL behavior matches the current Configuration Set.
7. As a user, I want File Manager Upload to honor the Global Configuration clipboard default, so that the integration does not create a separate clipboard policy.
8. As a user, I want no upload-specific override controls in File Explorer, so that temporary choices remain exclusive to Manual Upload and the CLI.
9. As a user, I want the helper to reload and strictly validate the current Configuration Set before work starts, so that it never uses stale or malformed configuration.
10. As a user, I want File Manager Upload to open no Upit Desktop window, so that a context-menu action remains direct and noninteractive.
11. As a user, I want Manual Upload to remain available only when I intentionally open Upit Desktop, so that its picker, drag-and-drop, and per-upload overrides retain a distinct role.
12. As a user, I want a native notification when preparation begins, so that I know File Explorer accepted the action.
13. As a user, I want phase and measurable byte progress in native feedback, so that long work is understandable without exposing private upload data.
14. As a user, I want to cancel an active File Manager Upload from native feedback, so that unwanted or stalled work can stop without Task Manager.
15. As a user, I want no second File Manager Upload to start while one is active for me, so that clipboard writes, progress, and notifications cannot race.
16. As a user, I want a second invocation to report that another upload is active and make no endpoint request, so that the result is deterministic rather than silently queued.
17. As a user, I want successful upload feedback to avoid showing the file path, file name, endpoint, Original URL, Final URL, or credentials, so that notifications remain safe on a lock screen.
18. As a user whose clipboard default is enabled, I want the resulting Final URL copied automatically, so that it is immediately ready to paste.
19. As a user whose clipboard default is disabled, I want a native Copy Final URL action, so that I can obtain the result deliberately without exposing it in notification text.
20. As a user whose Shortener falls back or whose automatic clipboard copy fails, I want a completed-with-warning notification, so that successful upload remains distinct from a warning.
21. As a user whose Configuration Set is absent or invalid, I want a sanitized failure notification with an Open Upit Desktop action, so that I can use Setup or Repair rather than guess at files.
22. As a user whose upload fails after starting, I want a sanitized failure notification with explicit Retry, so that I can make one fresh attempt after correcting the cause.
23. As a user, I want Retry to revalidate the file and Configuration Set and make exactly one new request, so that Upit never automatically repeats a potentially non-idempotent upload.
24. As a privacy-conscious user, I want Copy and Retry actions to use opaque notification tokens and temporary private state only, so that paths and URLs are not displayed, logged, or retained as history.
25. As a privacy-conscious user, I want temporary action state consumed on use and deleted ten minutes after completion when unused, so that notification recovery does not create persistent result retention.
26. As a user, I want notification failure or denied notification permission to fall back to a native alert when I otherwise could not observe a failure, cancellation, or un-copied success, so that a direct upload does not fail silently.
27. As a Windows user, I want the installed package to register the context-menu action automatically, so that no registry editing or desktop setting is required.
28. As a Windows user, I want uninstallation to remove the Explorer command and package registration, so that Upit leaves no stale right-click action.
29. As a security-conscious user, I want the File Explorer command and package signed, so that Windows can establish the publisher identity required for primary-menu integration.
30. As a release maintainer, I want signed artifacts only from protected SemVer tags, so that pull requests cannot access publishing credentials.
31. As a maintainer, I want the existing `upit` CLI contract unchanged, so that File Manager Upload does not become a new headless command mode or alter scripts.
32. As a maintainer, I want Explorer’s COM adapter to contain no upload, configuration, or notification policy, so that the shared application behavior remains in one Wails-free module.

## Implementation Decisions

- **File Manager Upload** is the canonical term for this direct file-manager invocation; it is distinct from Manual Upload and CLI invocation.
- v0.6 supports Windows 11 x64. Windows 10, Windows ARM64, and classic “Show more options” shell verbs are not release targets.
- A signed package with identity registers a native `IExplorerCommand` in the Windows 11 primary File Explorer menu. The in-process COM adapter validates the selection count at the Explorer seam, passes one path to the helper, and never performs network or configuration work in Explorer.
- The package contains Upit Desktop and the helper. Install automatically registers the command; uninstall removes every package and Explorer registration artifact.
- A deep Wails-free application module owns File Manager Upload preflight, default selection, regular-file validation, immutable configuration use, execution, Shortener behavior, clipboard behavior, structured outcomes, progress, cancellation, single-flight exclusion, and temporary action state. It has one small interface for native adapters rather than a separate Windows upload implementation.
- File Manager Upload permits no Uploader, Shortener, clipboard, or timeout override. It inherits the existing no-timeout default and the current Global Configuration defaults.
- Exactly one regular file is accepted. File Explorer suppresses the command for multi-selection; all handed-off values are still validated before any request.
- At most one File Manager Upload is active per user. A second invocation is rejected without queueing, background workers, or concurrent uploads.
- Native feedback exposes only phase, byte counts where measurable, outcome category, and sanitized diagnostics. It never displays or logs local paths, file names, endpoints, request values, response bodies, Original URLs, Final URLs, or credentials.
- If automatic clipboard copying is disabled, Copy Final URL and explicit Retry use opaque notification tokens that resolve through minimal per-user private state. That state stores only the action’s required path or URL, has private permissions, is consumed on use, is deleted ten minutes after completion when unused, and is never surfaced as upload history.
- Native notification actions are contextual only: Cancel while active; Copy Final URL after a non-copied success; Open Upit Desktop for Configuration Set failures; Retry for runtime failures. No action opens Upit Desktop automatically, no retry is automatic, and no URL is rendered in notification text.
- When native notifications cannot be shown, a transient native alert reports cancellation or failure; it also exposes a successful Final URL Copy action when automatic clipboard copying is disabled. The upload outcome itself is not changed by notification failure.
- The same application implementation remains the authority used by the CLI, Manual Upload, and File Manager Upload. Wails and COM are adapters over that implementation; generated bindings and the CLI text contract are not IPC mechanisms.
- The release pipeline builds unsigned verification artifacts for normal changes. Only a protected SemVer tag can use signing credentials to sign, timestamp, package, checksum, and publish the Windows release.

## Testing Decisions

- The permanent seam is the Wails-free File Manager Upload operation. Tests use temporary homes, real Configuration Set documents, real local HTTP endpoints, deterministic clipboard and notification adapters, and controlled contexts/readers.
- Tests prove default Uploader/Shortener/clipboard selection, missing or invalid Configuration Set handling, regular-file and selection validation, no endpoint request on rejected input, fresh preflight, default no-timeout behavior, Shortener fallback, clipboard warnings, sanitized failures, progress phase ordering, cancellation, and one active operation.
- Tests prove notification payloads and temporary action state do not disclose paths, names, URLs, endpoints, request values, response bodies, or credentials; state is private, consumed on action, and deleted on expiry.
- Tests prove Copy Final URL, Retry, and Open Upit Desktop actions reach the correct outcome without automatic retry or persistent history. Retry must perform a new preflight before one new request.
- The Explorer adapter receives narrow native tests for one-selection handoff and cancellation/action dispatch only. Tests do not mock or duplicate upload behavior inside COM.
- Native Windows 11 x64 smoke installs a signed package, verifies the primary context-menu action, starts an upload through a local endpoint, observes privacy-minimized progress, cancellation, success, warning, failure, Copy, Retry, configuration recovery, uninstall cleanup, and unchanged CLI behavior.
- CI runs Go tests, race detection where supported, static analysis, frontend typecheck/build, unsigned package checks, and release artifact checksum checks. Protected-tag CI additionally verifies package signature and the signed native smoke; ordinary pull requests receive no signing or publishing credentials.

## Out of Scope

- Opening, focusing, embedding, or otherwise showing Upit Desktop during File Manager Upload.
- Per-upload overrides, file picker input, drag-and-drop, batch selection, queues, concurrent uploads, upload history, scheduled uploads, automatic retries, or a new timeout policy.
- Windows 10 support, Windows ARM64 support, classic registry shell-verb fallback, portable integration registration, or manual registry instructions.
- Running configuration or network work in Explorer, parsing CLI stdout as integration IPC, a daemon, a tray process, or a persistent notification worker.
- New Uploader Request Body Modes, Response Extractors, Shortener protocols, Global Configuration fields, configuration locations, migrations, or credential stores.
- Displaying local paths, file names, URLs, endpoints, response content, request values, or credentials in native notifications, alerts, logs, or history.
- Linux, macOS, Finder, launcher, browser, mobile, or cloud integration.

## Further Notes

- v0.6 follows the completed v0.5 desktop release and its stable headless CLI regression proof.
- ADR 0013 records the release sequence; ADR 0014 records the one-shot Wails-free helper; ADR 0015 records the Windows 11 primary-menu route.
- Primary-source research for Windows shell extensibility, package identity, signing, and Wails packaging is retained in `research/windows-explorer-integration.md`.
