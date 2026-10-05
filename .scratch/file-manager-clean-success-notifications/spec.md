# Upit — Native Clean-Success Notifications for File Manager Upload

Status: ready-for-human

## Problem Statement

After a user invokes `Upload with Upit`, a successful File Manager Upload can still end in an application-owned dialog that requires acknowledgment. Windows currently presents completion through a Task Dialog or Message Box, and macOS falls back to a modal alert when notification delivery is unavailable. Requiring `OK` after routine success interrupts the direct file-manager workflow even though the Final URL has already been copied to the clipboard and no recovery action remains.

The terminal experience is also inconsistent across platforms and notification-permission states. A clean success should behave like an ordinary operating-system notification, while outcomes that require attention must remain visible and actionable.

## Solution

When File Manager Upload completes successfully without warnings and has copied the Final URL, Upit closes active progress feedback and emits one silent native operating-system notification. macOS uses `UNUserNotificationCenter`; Windows 11 uses Windows Toast. The notification title is `Upload complete` and its body is `Final URL copied to clipboard.`

The operating system controls display duration, Notification Center history, and visual grouping. Selecting the notification performs no Upit action. Each clean success emits a distinct notification; Upit does not replace, aggregate, or count previous successes. If native notification delivery is unavailable or fails, the helper exits silently because the Final URL has already been delivered through the clipboard.

Terminal changes are limited to clean success. macOS progress notifications are also silent, while their display and Cancel action are preserved. Rejection, cancellation, success with warning, failure, and recovery actions retain their existing interactive native feedback, sound, and fallback behavior. CLI and Manual Upload remain unchanged.

## User Stories

1. As a File Manager Upload user, I want a clean success to finish without an `OK` dialog, so that the direct workflow does not require unnecessary acknowledgment.
2. As a File Manager Upload user, I want active progress feedback to close when the upload finishes, so that completed work does not leave a stale progress surface.
3. As a File Manager Upload user, I want a native terminal notification after clean success, so that completion matches the conventions of my operating system.
4. As a macOS user, I want clean success delivered through the macOS notification system, so that it behaves like a normal macOS banner.
5. As a Windows 11 user, I want clean success delivered through Windows Toast, so that it behaves like a normal Windows notification.
6. As a user, I want the notification to be silent, so that routine uploads do not create unnecessary sound.
7. As a user, I want the operating system to control how long the notification remains visible, so that Upit respects my notification settings.
8. As a user, I want the operating system to control whether the notification remains in Notification Center, so that Upit does not invent a separate notification history.
9. As a user, I want the notification title to be `Upload complete`, so that the successful outcome is immediately clear.
10. As a user, I want the notification body to be `Final URL copied to clipboard.`, so that I know where the result was delivered.
11. As a privacy-conscious user, I want the notification to omit the selected file name and path, so that local information is not exposed on the lock screen.
12. As a privacy-conscious user, I want the notification to omit the Uploader endpoint, request values, and response content, so that remote-service details remain private.
13. As a privacy-conscious user, I want the notification to omit the Original URL, Final URL, and credentials, so that sensitive values are not rendered by the operating system.
14. As a user, I want selecting a clean-success notification to perform no Upit action, so that a routine acknowledgment cannot unexpectedly open another surface.
15. As a user, I do not want selecting the notification to open Upit Desktop, so that ordinary File Manager Upload remains background-only.
16. As a user, I do not want selecting the notification to open the Final URL, so that Upit does not retain or dispatch a result that I did not ask to open.
17. As a user, I do not want selecting the notification to copy the Final URL again, so that notification activation has no hidden clipboard side effect.
18. As a user performing several uploads in sequence, I want each clean success to emit one distinct notification, so that each completed invocation has one terminal event.
19. As a user performing several uploads in sequence, I want macOS or Windows to group those notifications according to system behavior, so that Upit respects platform conventions.
20. As a user performing several uploads in sequence, I do not want Upit to replace an earlier clean-success notification, so that a later completion does not erase an earlier event.
21. As a user performing several uploads in sequence, I do not want Upit to aggregate them into a counter, so that the one-shot helper does not create cross-invocation state.
22. As a user who has disabled notifications, I want a clean success to exit silently after copying the Final URL, so that Upit does not replace a disabled notification with a blocking dialog.
23. As a user affected by a native notification delivery failure, I want the completed upload to remain successful, so that presentation failure does not invalidate irreversible remote work.
24. As a user affected by a native notification delivery failure, I do not want an `OK` fallback for clean success, so that the direct workflow remains non-blocking.
25. As a user, I want the one-shot helper to exit after posting or skipping the clean-success notification, so that File Manager Upload does not become a resident process.
26. As a user starting a second File Manager Upload while one is active, I want rejection feedback to remain explicit, so that hidden work is not silently dropped.
27. As a user canceling an upload, I want cancellation feedback to retain its existing visibility, so that cancellation is not confused with clean success.
28. As a user whose Shortener fails after upload, I want the success-with-warning outcome to retain its existing warning feedback, so that fallback to the Original URL is visible.
29. As a user whose automatic clipboard copy fails, I want the completed upload to retain `Copy Final URL`, so that I can recover the existing result without uploading again.
30. As a user whose runtime upload fails, I want `Retry` to remain explicit and interactive, so that Upit never repeats a potentially non-idempotent request automatically.
31. As a user whose Configuration Set is absent or invalid, I want `Open Upit` to remain available, so that I can deliberately enter Setup or Repair.
32. As a privacy-conscious user, I want warning, failure, and recovery actions to keep their opaque-token and short-lived private-state contracts, so that this notification change does not weaken existing protections.
33. As a CLI user, I want CLI upload output and behavior unchanged, so that a File Manager Upload presentation change does not alter automation.
34. As a Manual Upload user, I want Desktop upload progress, results, and controls unchanged, so that the direct file-manager workflow remains a separate surface.
35. As a maintainer, I want clean-success presentation policy defined once in the one-shot helper, so that macOS and Windows do not drift on outcome classification or fallback behavior.
36. As a maintainer, I want native adapters limited to delivering platform notifications and reporting delivery availability, so that they do not duplicate upload policy.
37. As a maintainer, I want deterministic tests to distinguish clean success from warning, failure, cancellation, and rejection, so that future changes cannot make actionable outcomes disappear.
38. As a release maintainer, I want native smoke evidence on macOS 14+ and Windows 11, so that a passing unit test is not mistaken for proof of the real operating-system surface.

## Implementation Decisions

- This specification implements ADR 0020 and supersedes only the clean-success terminal-presentation clauses of the Unified File Manager Integration specification. Its upload, privacy, cancellation, single-flight, recovery-action, helper-lifecycle, and platform-adapter contracts otherwise remain authoritative.
- A clean success is a File Manager Upload result that succeeded without warnings or recovery actions under the always-copy contract. Warning, failure, cancellation, and rejection are not clean success.
- The existing `FileManagerUploadService` remains authoritative for upload execution, outcome status, warnings, clipboard delivery, and recovery actions. It does not acquire operating-system presentation policy.
- The private one-shot helper gains one platform-neutral terminal-feedback policy as the authoritative presentation seam. It classifies the result once and selects either clean-success notification delivery or the existing interactive terminal path.
- Clean success closes active progress feedback before terminal notification delivery is attempted.
- macOS delivers clean success through `UNUserNotificationCenter`.
- Windows 11 delivers clean success through Windows Toast using the product's package identity. Task Dialog and Message Box remain available only where the preserved interactive feedback contract requires them.
- The clean-success notification is silent on both platforms. Focus modes, notification permissions, sound policy, display duration, history, and visual grouping remain under operating-system control.
- macOS active progress notifications are silent so their sound cannot overlap the clean-success banner. Their visibility and Cancel action are unchanged; actionable terminal notification sound is preserved. This scope extension was explicitly approved during native testing.
- The clean-success notification title is exactly `Upload complete` and its body is exactly `Final URL copied to clipboard.`
- The clean-success notification contains no file name, path, endpoint, request value, response content, Original URL, Final URL, credential, or opaque action token.
- Selecting or activating the clean-success notification performs no product action. It does not open Upit Desktop, open a URL, copy again, retry, or create history.
- The macOS helper observes AppKit launch context before dispatching invocation arguments. A notification-only launch is ignored only when `NSApplicationLaunchUserNotificationKey` is present; ordinary no-argument and malformed invocations retain their existing usage-error behavior.
- Each clean success uses a distinct notification event. Upit does not use a stable replacement tag, aggregate count, or cross-invocation state; the operating system may group events by application.
- If clean-success notification delivery is denied, unavailable, or fails, the helper exits silently after closing progress feedback. It does not show an alert, Task Dialog, Message Box, custom transient panel, or Desktop window.
- Notification delivery failure does not change the upload result or exit status because the Final URL has already been copied.
- Rejection, cancellation, success with warning, failure, `Copy Final URL`, `Retry`, and `Open Upit` retain their existing interactive feedback, native alert fallback, action-token, consumption, and expiry contracts.
- The post-action confirmation shown after `Copy Final URL` is not reclassified as File Manager Upload clean success by this specification.
- No Global Configuration field, Configuration Set schema change, migration, separate File Manager notification preference, background process, or upload-history store is introduced.
- CLI and Manual Upload presentation and behavior remain unchanged.

## Testing Decisions

- Permanent tests assert consumer-visible behavior: which terminal surface is selected, notification content and privacy, whether interaction is required, whether fallback occurs, and whether the helper exits. They do not pin source layout, native call ordering, generated bindings, or copied adapter plumbing.
- The primary behavioral seam is the platform-neutral terminal-feedback policy in the one-shot helper. This is the one place that proves clean-success classification and fallback policy for both operating systems.
- Terminal-feedback policy tests use representative real File Manager Upload results from the existing application contract. They prove that clean success closes progress, requests one silent notification with the exact title and body, exposes no action, does not wait for a response, and exits silently when delivery is unavailable.
- The same policy tests prove rejection, cancellation, success with warning, failure, and action-bearing outcomes continue to select the existing interactive path.
- Existing `FileManagerUploadService` tests remain the prior art and authority for producing clean success, Shortener fallback, clipboard warning, failure, cancellation, rejection, and recovery actions. Presentation assertions are not duplicated in that service.
- Existing helper summary privacy tests remain prior art for sanitized user-facing text. Add terminal-notification coverage for file names, paths, endpoints, request values, response content, Original URLs, Final URLs, credentials, and action tokens.
- Add a sequential-success test proving two clean successes request two distinct notification events and do not use replacement or aggregation state.
- Add a notification-delivery-failure test proving clean success remains successful, no modal fallback is requested, no product action is exposed, and the helper exits.
- macOS native tests remain narrow. They prove the adapter creates one silent `UNUserNotificationCenter` request with the confirmed title/body, no category action or product activation, and a distinct identifier, and reports unavailable delivery without creating an `NSAlert` for clean success.
- Narrow macOS adapter tests prove both initial and updated progress requests omit sound and actionable terminal requests retain sound.
- Launch-context regressions distinguish Notification Center activation from ordinary invocation, including the no-argument exit-2 contract and malformed arguments under notification context.
- Windows native tests remain narrow. They prove the adapter creates one silent Windows Toast with the confirmed title/body, no activation behavior, and a distinct event identity, and reports unavailable delivery without creating a Task Dialog or Message Box for clean success.
- Native adapter tests do not repeat upload execution, clipboard behavior, Shortener behavior, retry semantics, Configuration Set loading, or action-store tests.
- macOS 14+ Apple Silicon native smoke invokes `Upload with Upit` from Finder, observes progress close, observes the silent native notification, confirms it auto-hides according to macOS behavior, selects it without opening Upit, and confirms no `OK` alert appears. A second sequential upload proves a distinct terminal event.
- Windows 11 x64 native smoke invokes `Upload with Upit` from File Explorer, observes progress close, observes the silent Windows Toast, confirms it auto-hides according to Windows behavior, selects it without opening Upit, and confirms no Task Dialog or Message Box remains for clean success. A second sequential upload proves a distinct terminal event.
- Native smoke also samples at least one actionable non-clean outcome on each platform to prove warning, failure, or recovery feedback did not become silent.
- A direct helper invocation, source inspection, unit test, mocked notification API, or compile check is not accepted as proof of the real Finder/Explorer and operating-system notification surface.

## Out of Scope

- Auto-hiding, silencing, or otherwise redesigning rejection, cancellation, warning, failure, `Copy Final URL`, `Retry`, or `Open Upit` feedback.
- Changing the post-action confirmation shown after a successful `Copy Final URL` recovery action.
- A custom display timeout, custom transient notification window, custom notification center, or custom sound.
- Opening Upit Desktop, opening the Final URL, copying again, or performing any other action when a clean-success notification is selected.
- Replacing earlier success notifications, aggregating a success count, or retaining cross-invocation notification state.
- Upload history, notification history owned by Upit, queues, batching, concurrent File Manager Uploads, scheduling, or automatic retry.
- Changes to File Manager Upload selection, Configuration Set loading, Uploader selection, Shortener behavior, clipboard policy, timeout policy, or recovery-token lifetime.
- Changes to CLI or Manual Upload behavior.
- Notification-settings management inside Upit Desktop.
- Windows 10, Windows ARM64, macOS versions before 14, Intel macOS, Linux file-manager notifications, or other operating systems.

## Further Notes

- ADR 0020 records the native, silent, non-activating clean-success notification and silent delivery-failure contract.
- The Unified File Manager Integration confirmed design is the design source for this specification. Its clean-success fallback clause is narrowed by ADR 0020 and this specification; its remaining contracts are preserved.
- The current runtime is platform-specific: authorized macOS notification delivery already uses a native banner, denied macOS delivery falls back to a blocking alert, and Windows completion uses Task Dialog or Message Box rather than Windows Toast.
- Windows package identity already provides the prerequisite product identity for Windows Toast; this specification does not introduce a separate resident notifier.
- Signed/notarized target-platform smoke remains required before production readiness. Unit and unsigned package checks cannot prove real operating-system notification presentation.
