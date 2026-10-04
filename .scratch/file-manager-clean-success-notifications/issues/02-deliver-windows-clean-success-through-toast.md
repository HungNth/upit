# 02: Deliver Windows clean success through Windows Toast

**What to build:** Make a clean File Manager Upload invoked from Windows 11 File Explorer close its progress dialog and finish through one silent native Windows Toast instead of waiting for a Task Dialog or Message Box acknowledgment. Reuse the shared terminal-feedback policy established by the macOS slice, while preserving every actionable Windows outcome.

**Blocked by:** 01: Make macOS clean success non-blocking through the shared terminal policy.

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the shared terminal-feedback policy for platform-neutral behavior and keep Windows-native tests limited to Toast payload and identity, package-identity delivery, activation behavior, unavailable/failure reporting, and modal-fallback exclusion.

**Demo path:** Install a development Windows 11 package, invoke `Upload with Upit` from the primary File Explorer context menu against a local endpoint, and observe the progress Task Dialog close followed by the silent native Toast `Upload complete` / `Final URL copied to clipboard.` Invoke a second upload to observe a distinct event, select the Toast to prove no Upit action occurs, then exercise Toast-unavailable and forced delivery-failure cases to prove the helper exits without a Task Dialog or Message Box fallback. Trigger one actionable non-clean outcome to prove its existing interaction remains available.

- [ ] Windows clean-success presentation consumes the shared terminal-feedback policy without redefining clean success, notification copy, privacy, fallback, action, or event-identity rules.
- [ ] A clean success closes the active progress Task Dialog before terminal notification delivery is attempted.
- [ ] Windows 11 emits one silent native Toast using Upit's existing package identity, without adding a resident notifier or separate user-visible application.
- [ ] The Toast title is `Upload complete` and its body is `Final URL copied to clipboard.`
- [ ] The Toast contains no file name, path, endpoint, request value, response content, Original URL, Final URL, credential, or action token.
- [ ] The Toast has no activation behavior that opens Upit Desktop, opens a URL, copies again, retries, or creates history.
- [ ] Sequential clean successes use distinct Toast event identities and are not replaced or aggregated by Upit; Windows remains free to group them.
- [ ] Toast unavailable/disabled and Toast API delivery failure are independently controllable test cases.
- [ ] In both unavailable-delivery cases, clean success remains successful, active progress closes, no Task Dialog or Message Box fallback appears, and the one-shot helper exits without waiting for acknowledgment.
- [ ] Rejection, cancellation, success with warning, failure, `Copy Final URL`, `Retry`, and `Open Upit` retain their existing Task Dialog or Message Box interaction where required, including action-token privacy, consumption, and expiry behavior.
- [ ] CLI and Manual Upload behavior, Configuration Set schema, notification preferences, upload history, and background-process behavior remain unchanged.
- [ ] Deterministic shared-policy tests and narrow Windows adapter tests prove the contract without duplicating upload execution tests or asserting source layout/native call ordering.

## Comments

- Implementation is present: native WinRT Toast delivery through a hidden, bounded one-shot PowerShell process, the installed `HungNth.Upit` package AUMID, distinct shared event tags, silent audio, and progress-dialog readiness/closure synchronization. Existing non-clean Task Dialog/Message Box interactions are retained.
- Per explicit implementation-time user instruction, Toast XML has only visual text and silent audio: no `activationType`, `launch`, or actions. Actual selection behavior is an outstanding native acceptance check, not proven by payload inspection.
- Payload/process-boundary tests pass. The Windows helper and Windows test binary cross-compile; the native Task Dialog test has not run on Windows. The actual PowerShell script parses and its missing-package and unavailable-native-API process branches were exercised on macOS as supporting checks only.
- No Windows 11 x64 runtime or configured SSH host is available in this environment. Actual Toast display, disabled-notification delivery, API failure, progress-window closure, selection behavior, and preserved actionable outcomes require the native Windows runner.
- Implementation remains uncommitted. Native acceptance checkboxes remain open; neither compilation nor supporting process checks claim production readiness.
