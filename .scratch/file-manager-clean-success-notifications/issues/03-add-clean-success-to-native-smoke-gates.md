# 03: Add clean-success behavior to native smoke gates

**What to build:** Extend the existing macOS and Windows native smoke gates so production readiness requires proof that a real `Upload with Upit` invocation finishes clean success through the confirmed native notification behavior, while outcomes requiring attention remain interactive. This focused proof augments rather than replaces the broader unified-product signed-release gates.

**Blocked by:** 01: Make macOS clean success non-blocking through the shared terminal policy; 02: Deliver Windows clean success through Windows Toast.

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the real installed Finder and File Explorer integrations against a local upload endpoint. Direct helper invocation, unit tests, package inspection, and mocked notification APIs are supporting evidence only and cannot substitute for the operating-system surface.

**Demo path:** On macOS 14+ Apple Silicon and Windows 11 x64, install the appropriate development or protected package, invoke `Upload with Upit` from the real file-manager surface, observe progress close and the silent native notification, select it without causing an Upit action, repeat the upload for a distinct event, and confirm no modal clean-success dialog remains. On each platform, also trigger at least one warning, failure, or recovery action and confirm it remains interactive.

- [x] The macOS smoke starts File Manager Upload from Finder Services or Quick Actions rather than invoking the private helper directly.
- [x] The macOS smoke observes active progress close, one silent native notification with the confirmed title/body, operating-system-controlled dismissal, no Upit action on selection, helper exit, and no clean-success `NSAlert`.
- [ ] The Windows smoke starts File Manager Upload from the Windows 11 primary File Explorer context menu rather than invoking the private helper directly.
- [ ] The Windows smoke observes the progress Task Dialog close, one silent Windows Toast with the confirmed title/body, operating-system-controlled dismissal, no Upit action on selection, helper exit, and no clean-success Task Dialog or Message Box fallback.
- [ ] Each platform smoke performs two sequential clean successes and observes two distinct terminal events without Upit replacement or aggregation.
- [ ] Each platform smoke confirms notification content omits file names, paths, endpoints, request values, response content, Original URLs, Final URLs, credentials, and action tokens.
- [ ] Each platform smoke exercises at least one rejection, cancellation, warning, failure, or recovery-action path and confirms it remains visible and interactive rather than adopting the clean-success silent fallback.
- [x] Ordinary File Manager Upload still opens no Upit Desktop window, Dock icon, tray process, or resident worker during the smoke.
- [x] CLI and Manual Upload remain operational against the same Configuration Set after the native smoke.
- [x] The smoke gates state clearly that unsigned or local evidence does not prove publisher trust, notarization, package registration, or production readiness.
- [x] Existing umbrella Windows signed-release and macOS signed/notarized-release tickets remain the authoritative production proof; this focused ticket does not replace or close them.

## Comments

- Both existing native smoke gates now collect the clean-success observations and reject missing confirmations. Existing lifecycle, continuity, signed-installer, and umbrella release requirements are retained.
- Evidence records supporting scope explicitly and always sets `authoritativeProductionProof` to false. The protected signed/notarized workflows remain authoritative.
- Windows removes the headless test override during native Explorer observations and restores the caller's original value only during final cleanup. Both local endpoints leave time to observe progress/cancel.
- Supporting checks exercised macOS script syntax/help, Windows PowerShell parsing, and an isolated mock-operator/install gate harness: all required observations passed; missing silent-notification confirmation failed and was recorded false. This harness is not Finder/Explorer or release proof.
- Remaining human execution: run the updated gates from real Finder and File Explorer on their supported installed packages, including visible sequential notifications, selection, and a non-clean outcome. No native release evidence was fabricated; observation acceptance checkboxes remain open.

- Final available checks passed: `go test ./...` (five tested packages; one package has no tests), `go vet ./...`, `go test -race ./cmd/upit-file-manager`, desktop `vue-tsc --noEmit`, and Windows/Linux helper cross-builds. macOS linker emitted duplicate-Objective-C-library and deployment-target warnings; no check failed.
- Two-axis review found no documented standards violations or additional code/spec defects. Two minor standards observations were retained deliberately: the required shared presentation coordinator and one-line stderr formatting repeated for unsupported/headless adapters. Native acceptance proof remains blocked as stated above.
- Real local Finder smoke evidence is preserved at `../macos-local-smoke-evidence.json`. It verifies native menu invocation, quiet completion, exact content, auto-dismissal, two distinct events, selection with no product action or clipboard side effect, notifications-disabled silent fallback, and visible failure/Retry. It leaves selection validation, cancellation, warning/Copy recovery, Retry completion presentation, protected macOS release proof, and all Windows native proof open.
- Smoke exited nonzero by design because incomplete observations remained false. Cleanup then restored Configuration Set paths/bytes/modes, original `/Applications/Upit.app` binary hash and preferred LaunchServices mappings; the copied test app was removed. Local evidence remains supporting only, never production proof.
- Final cleanup verification found no test processes, mounts, copied app, temporary smoke directories, or LaunchServices entries for the deleted test path. The Configuration Set still matched its pretest backup by paths, bytes, and modes; `/Applications/Upit.app` retained its pretest Desktop binary hash and preferred Desktop/Finder Service mappings.
