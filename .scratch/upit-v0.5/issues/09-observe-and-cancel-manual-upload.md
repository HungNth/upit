# 09: Observe and cancel Manual Upload

**What to build:** Complete Manual Upload lifecycle visibility and interruption. A user sees truthful phases and byte progress, cancels one active upload, navigates without stopping it, and closes the desktop safely. The slice provides a global active-upload surface without adding queues, background workers, or a second operation.

**Blocked by:** 08: Perform one Manual Upload

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Verify progress/cancellation through the Wails-free application module with controllable readers, local HTTP servers, and contexts. Use narrow Wails adapter checks for cancellable binding propagation and event payload shape. A real desktop smoke proves global status, navigation, Cancel, and close confirmation.

**Demo path:** Start a deliberately slow local upload, observe preparation then monotonic byte transfer progress, navigate to a configuration area while progress remains visible, cancel it, retry successfully, then confirm that closing during a second slow upload asks before cancellation and exits only after release.

- [ ] Manual Upload exposes ordered preparation, transfer, response, Shortener, and clipboard phases without inventing percentage progress for indeterminate work.
- [ ] Transfer progress reports monotonic processed-byte counts against the selected file size whenever the active Request Body Mode can measure file progression.
- [ ] Progress payloads and visible state never include endpoints, headers, query values, static request data, response bodies, URLs beyond the eventual result contract, or credentials.
- [ ] Exactly one upload is active. Navigation remains available and a global status surface retains phase, progress, and Cancel access outside the Manual Upload area.
- [ ] Cancel propagates through validation, request streaming, response wait, and Shortener work; an intentional cancellation is presented as canceled, not as ordinary failure or fallback success.
- [ ] Active operations use their starting immutable configuration snapshot; saves during the operation affect only later uploads.
- [ ] Selected file and overrides remain available after cancellation or failure for an explicit later Retry.
- [ ] Close during an active upload asks for confirmation; confirmed close cancels, waits for resource release, and then exits with no tray or resident process.
- [ ] Deterministic application tests and a real desktop smoke verify phase order, byte progress, cancellation, navigation while active, close behavior, and complete process exit.
