# 01: Make macOS clean success non-blocking through the shared terminal policy

**What to build:** Make a clean File Manager Upload invoked from Finder finish through one silent native macOS notification instead of a modal success alert. Establish the shared terminal-feedback policy as part of this complete macOS path so clean success is classified once, notification delivery failure exits silently after clipboard delivery, and outcomes requiring attention retain their existing interactive behavior.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Exercise one platform-neutral terminal-feedback policy with representative real File Manager Upload results, then keep macOS-native tests limited to notification delivery, unavailable/failure reporting, identifier/action shape, and modal-fallback exclusion.

**Demo path:** Package a development Upit application on macOS, invoke `Upload with Upit` from Finder against a local endpoint, and observe progress close followed by the silent native notification `Upload complete` / `Final URL copied to clipboard.` Invoke a second upload to observe a distinct event, select the notification to prove no Upit action occurs, then exercise denied permission and forced delivery failure to prove the helper exits without an `NSAlert`. Trigger one actionable non-clean outcome to prove its existing interaction remains available.

- [ ] The one-shot helper has one platform-neutral terminal-feedback policy that defines clean success as a succeeded File Manager Upload with no warnings or recovery actions under the always-copy contract.
- [ ] The shared policy owns the exact clean-success title, body, silent presentation, no-product-action rule, distinct-event rule, and silent delivery-failure outcome; native adapters do not duplicate those product decisions.
- [ ] `FileManagerUploadService` remains authoritative for upload status, warnings, clipboard delivery, and recovery actions and does not acquire operating-system presentation policy.
- [ ] A clean success closes active macOS progress feedback before terminal notification delivery is attempted.
- [ ] Initial and updated macOS progress notifications are silent; their display and Cancel action remain, and actionable terminal notifications retain their existing sound.
- [ ] macOS emits one silent `UNUserNotificationCenter` notification with title `Upload complete` and body `Final URL copied to clipboard.`
- [ ] The notification contains no file name, path, endpoint, request value, response content, Original URL, Final URL, credential, or action token.
- [ ] The notification has no Copy, Retry, Open Upit, URL-opening, clipboard, or Desktop activation behavior; selecting it performs no Upit action.
- [ ] Sequential clean successes use distinct notification identifiers and are not replaced or aggregated by Upit; macOS remains free to group them.
- [ ] Notification permission denied/unavailable and notification API delivery failure are independently controllable test cases.
- [ ] In both unavailable-delivery cases, clean success remains successful, active progress closes, no `NSAlert` or custom fallback appears, and the one-shot helper exits without waiting for acknowledgment.
- [ ] Rejection, cancellation, success with warning, failure, `Copy Final URL`, `Retry`, and `Open Upit` retain their existing interactive feedback, action-token privacy, consumption, and expiry behavior.
- [ ] CLI and Manual Upload behavior, Configuration Set schema, notification preferences, upload history, and background-process behavior remain unchanged.
- [ ] Deterministic policy tests and narrow macOS adapter tests prove the contract without duplicating upload execution tests or asserting source layout/native call ordering.

## Comments

- Implementation is present: shared terminal policy, both upload/retry callers, and the native macOS adapter. `FileManagerUploadService` is unchanged.
- Focused policy tests pass for clean success, unavailable/API-failure delivery, preserved non-clean interaction, and distinct sequential events. `bash native/macos/feedback/test.sh` passes using the same non-ARC ownership model as cgo.
- The real isolated packaged helper was exercised against a delayed local endpoint: exit 0, exactly one upload request, actual Final URL clipboard delivery, empty stdout/stderr. Before/after screenshots showed native progress open and then closed with no modal success alert.
- A runtime-discovered autoreleased progress identifier crash is fixed with matching copy/release ownership; the non-ARC native test failed before and passed after the fix.
- Remaining human proof: real Finder invocation, authorized banner appearance/dismissal, sequential visible events, notification selection, and a preserved actionable outcome. Supporting checks above do not replace signed/notarized release proof; acceptance checkboxes remain open pending that evidence.
- During real Finder testing, the user reported audio at completion. Native logs correlated the last audible progress delivery immediately before the clean banner. The user explicitly approved making macOS progress notifications silent while preserving actionable terminal feedback; this is now included in the parent spec.
- Real Finder testing of the updated, locally ad-hoc-signed test bundle now confirms exact completion text, quiet progress/completion, no OK acknowledgment, and operating-system banner dismissal. This is local native evidence, not protected signed/notarized release proof.
- A real click initially produced the invalid-file alert. Startup now waits for AppKit launch context and recognizes notification-only activation before normal argument dispatch. Native launch-context tests and an actual ordinary no-argument helper invocation preserve exit 2; the user later observed no app/dialog on a new notification click and the launched helper exited.
- A marker installed after upload remained unchanged when the new completion notification was selected; no product surface appeared and the notification-launched helper exited. This confirms no copy-again side effect for the isolated local case.
- Notifications-disabled testing confirmed the completed upload still copied its Final URL, closed progress, showed no banner or modal fallback, and exited. Two subsequent authorized uploads produced two separate visible completion events with distinct native IDs.
- First-time notification authorization remains an open issue: the 500ms wait can let the helper finish before the user answers the initial permission prompt. Enabling Notifications through System Settings permits later delivery but does not fix first-use UX.
- Actionable failure remained visible with Retry. An in-window Retry returned HTTP 200, copied its Final URL, and queued completion delivery, but the user did not observe the completion banner; Retry acceptance remains partial.
- Local evidence: `../macos-local-smoke-evidence.json`. Remaining native acceptance includes selection validation, cancellation, warning/Copy recovery, Retry completion presentation, and protected signed/notarized release proof. Repository changes remain uncommitted.
