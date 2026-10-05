# Upit — Windows Classic File Manager Integration

Status: ready-for-agent

## Problem Statement

Upit's Windows File Manager Integration currently depends on a packaged `IExplorerCommand`, sparse MSIX package identity, a native Explorer COM adapter, and trusted signing. An ordinary unsigned installer can be built, but it cannot establish the production registration required by the Windows 11 primary File Explorer menu, so installation fails when it attempts to register the unsigned package.

The user intentionally uses the classic Windows context menu and does not require Upit to appear in the Windows 11 primary menu. Requiring package identity and trusted signing therefore adds build, installation, Repair, and release complexity without serving the desired workflow. The product should expose File Manager Upload under **Show more options**, support ordinary unsigned installation, and preserve the existing one-file upload, privacy, progress, cancellation, recovery, Desktop management, update safety, and one-shot process contracts.

## Solution

Replace the Windows packaged `IExplorerCommand` route with one current-user Classic Verb named `Upload with Upit`. On Windows 11 x64, the command appears under **Show more options** and invokes the private one-shot helper directly with exactly one quoted selected-file path. The verb applies to all file extensions, excludes folders, and is hidden for multi-selection.

Upit Desktop continues to inspect and explicitly Repair File Manager Integration. `Registered` means the exact expected per-user verb values point to the active installed helper, that helper exists as a regular file, and the obsolete package route is absent. Intact payload with absent or stale Classic Verb registration or a remaining obsolete package route is `Needs Repair`; missing or damaged required payload is `Reinstall Upit`. Repair rewrites the exact Classic Verb registration, removes the obsolete package route when present, and then re-inspects without elevation.

The Windows consumer remains one NSIS installer with versioned payload staging. Ordinary unsigned installers are fully installable and functional. Protected Authenticode signing remains optional, but it is no longer a prerequisite for File Manager Integration. Installation and update stage and verify the Classic Verb before removing the obsolete package identity and `IExplorerCommand` route. A brief overlap is allowed only inside the uncommitted transaction; successful completion exposes only the Classic Verb. Uninstall removes registration before deleting payload and fails closed if cleanup cannot be proven.

Windows clean success closes progress, leaves the Final URL in the clipboard, and exits silently without attempting Windows Toast delivery. Outcomes requiring attention retain their existing native interactive feedback and recovery actions. macOS File Manager Integration and clean-success notifications remain unchanged.

## User Stories

1. As a Windows 11 x64 user, I want `Upload with Upit` in the classic File Explorer context menu, so that it matches the menu workflow I use.
2. As a Windows 11 x64 user, I accept opening **Show more options** or pressing `Shift+F10`, so that Upit does not require the primary-menu package route.
3. As a Windows user, I want an unsigned Upit installer to install File Manager Integration successfully, so that a signing certificate is not required for ordinary use.
4. As a Windows user, I want installation to remain current-user, so that it does not require elevation.
5. As a Windows user, I want one `Upload with Upit` command, so that obsolete modern and new classic registrations do not appear together.
6. As a Windows user upgrading from the packaged route, I want the obsolete package identity and COM command removed automatically, so that no stale primary-menu action remains.
7. As a Windows user, I want the Classic Verb to apply to every file extension, so that supported regular files are not limited by association.
8. As a Windows user, I do not want `Upload with Upit` on folders or folder backgrounds, so that the command always has one selected file.
9. As a Windows user selecting exactly one file, I want `Upload with Upit` to be visible in the classic menu, so that I can start File Manager Upload directly.
10. As a Windows user selecting multiple files, I want the command hidden, so that Upit does not start competing uploads or clipboard writes.
11. As a Windows user, I want the command to retain the Upit icon and the exact label `Upload with Upit`, so that the action is recognizable.
12. As a Windows user, I want the command to launch the private one-shot helper directly, so that Upit Desktop does not open for File Manager Upload.
13. As a user selecting a path containing spaces or Unicode characters, I want the complete path passed unchanged, so that the intended file is uploaded.
14. As a user, I want the selected path treated as untrusted after Explorer handoff, so that registry invocation cannot bypass regular-file validation.
15. As a user selecting a non-regular or changed file, I want rejection before network work, so that unsupported input cannot create partial remote work.
16. As a user, I want File Manager Upload to continue accepting exactly one regular file, so that each invocation has one deterministic result.
17. As a user, I want File Manager Upload to use the current Configuration Set, default Uploader, and optional default Shortener, so that registration changes do not alter upload behavior.
18. As a user, I want File Manager Upload to continue attempting to copy the Final URL regardless of the Global Configuration clipboard preference, so that the direct workflow remains paste-ready.
19. As a user, I want progress and cancellation to remain native and interactive while work is active, so that long uploads remain observable and stoppable.
20. As a user starting a second File Manager Upload, I want the existing single-flight rejection, so that hidden work is not queued or run concurrently.
21. As a user whose upload succeeds cleanly, I want progress to close after the Final URL is copied, so that completed work leaves no active surface.
22. As a Windows user whose upload succeeds cleanly, I want the helper to exit silently without a Toast or acknowledgment dialog, so that unsigned installation needs no notification identity.
23. As a user whose clean-success Toast was previously available, I accept losing that Windows notification, so that package identity can be removed.
24. As a user whose Shortener, clipboard, configuration, cancellation, or upload outcome needs attention, I want the existing native feedback and recovery actions preserved, so that actionable outcomes never become silent.
25. As a privacy-conscious user, I want feedback to continue omitting file names, paths, endpoints, request values, response content, Original URLs, Final URLs, credentials, and action tokens, so that the cutover does not expose private data.
26. As a Desktop user, I want the File Manager Integration area retained, so that registration health remains visible inside Upit.
27. As a Desktop user, I want `Registered` only when the exact Classic Verb contract points to the active helper and the helper exists, so that status cannot be satisfied by a stale key.
28. As a Desktop user with intact payload but missing or mismatched registry values, I want `Needs Repair`, so that the recoverable condition is explicit.
29. As a Desktop user with missing required payload, I want `Reinstall Upit`, so that Repair does not claim it can recreate binaries.
30. As a Desktop user, I want Repair to run only after my explicit action, so that passive inspection never mutates the registry.
31. As a Desktop user choosing Repair, I want Upit to rewrite the exact current-user Classic Verb and re-inspect it, so that success is evidence-based.
32. As a Desktop user, I want guidance to name **Show more options**, so that verification matches the actual Windows 11 surface.
33. As a Desktop user, I want an action to open File Explorer for manual verification, so that `Registered` is not confused with live menu visibility.
34. As a Desktop or Manual Upload user, I want File Manager Integration failure isolated from Configuration Set management and Manual Upload, so that an integration fault does not disable working features.
35. As an installer user, I want one NSIS setup executable containing Desktop, CLI, and the private helper, so that Upit remains one product.
36. As an installer user, I do not want a separate MSIX, Explorer DLL, or helper application entry, so that private implementation details remain private.
37. As an installer user, I want installation to stage a complete new payload before changing registration, so that a partial copy cannot become active.
38. As an installer user, I want update to switch the Classic Verb to the new helper only after the new payload is ready, so that the command never intentionally targets incomplete files.
39. As an installer user, I want the new registration inspected before the previous payload is removed, so that update success is proven before cleanup.
40. As an installer user, I want failed registration to restore the previous working route when possible, so that a failed update does not strand File Manager Integration.
41. As an installer user, I want a failure to remove the obsolete packaged route to abort and roll back, so that Upit never commits duplicate classic and modern commands.
42. As an installer user, I want failed rollback reported explicitly with required payload retained, so that recovery material is not deleted.
43. As a user uninstalling Upit, I want the Classic Verb and obsolete package registration removed before files are deleted, so that no stale action points to missing code.
44. As a user uninstalling Upit, I want cleanup failure to abort removal and retain payload, so that a surviving command remains callable rather than broken.
45. As a release maintainer, I want ordinary `make package` output to be fully installable without signing inputs, so that unsigned functional smoke is authoritative.
46. As a release maintainer, I want protected Authenticode signing to remain available for binaries and the installer, so that trust and reputation can be added without changing integration behavior.
47. As a release maintainer, I want signing secrets to remain restricted to protected SemVer-tag workflows, so that making signing optional does not weaken secret boundaries.
48. As a release maintainer, I want the Windows package build to stop requiring the C++ Explorer adapter toolchain and MSIX creation tools, so that packaging reflects the Classic Verb architecture.
49. As a release maintainer, I want package metadata and checksums to remain published, so that unsigned artifacts are still versioned and verifiable.
50. As a release maintainer, I want native Windows 11 x64 smoke to install the real unsigned consumer setup and exercise File Explorer, so that registry tests are not mistaken for user-surface proof.
51. As a release maintainer, I want native smoke to observe one-file visibility, multi-selection suppression, direct invocation, silent clean success, non-clean feedback, Desktop Repair, update continuity, rollback, and uninstall cleanup, so that the full lifecycle is proven.
52. As a release maintainer, I want migration smoke to start from a real old package identity and prove its removal, so that source inspection or mocked package queries do not stand in for cutover evidence.
53. As a CLI user, I want CLI behavior and output unchanged, so that File Manager Integration registration does not alter automation.
54. As a Manual Upload user, I want Desktop upload behavior unchanged, so that the Windows shell cutover remains isolated from interactive uploads.
55. As a macOS user, I want Finder Services, packaging, and native clean-success notifications unchanged, so that the Windows-only decision does not regress macOS.
56. As a maintainer, I want the shared File Manager Upload implementation to remain authoritative, so that Classic Verb registration does not duplicate upload policy.
57. As a maintainer, I want the platform-neutral File Manager Integration seam retained, so that Desktop does not learn registry details.
58. As a maintainer, I want Windows registry behavior concentrated behind one adapter, so that install, Desktop Repair, update, and uninstall use one authoritative contract.
59. As a maintainer, I want no package-identity compatibility layer or dual registration mode, so that the codebase has one current Windows route after migration.
60. As a maintainer, I want historical v0.6 records preserved, so that the move away from the primary menu is documented as a deliberate superseding decision rather than rewritten history.

## Implementation Decisions

- ADR 0021 is authoritative. It supersedes ADR 0015 and only the Windows Toast portion of ADR 0020; macOS native clean-success notification behavior remains authoritative.
- Windows support remains Windows 11 x64. Windows 10, ARM64, and modern primary-menu placement are not added by this cutover.
- Windows uses one per-user static verb under `HKCU\Software\Classes\*\shell\Upit.Upload`; no machine-wide registration or elevation is introduced.
- The verb display name is exactly `Upload with Upit`. Its icon resolves from the installed Upit Desktop executable. `MultiSelectModel` is exactly `Single`.
- The verb command contains the fully qualified private helper path and one quoted `"%1"` argument. Both the executable path and selected path are quoted.
- The verb is registered under the all-files `*` association only. No directory, folder, folder-background, or all-filesystem-objects registration is created.
- The private one-shot helper remains the selected-path entry point. It continues to revalidate exactly one regular file before network work; the registry is not a trust boundary.
- The existing File Manager Upload module remains authoritative for Configuration Set loading, default Uploader and optional Shortener selection, always-copy behavior, progress, cancellation, single-flight, warnings, privacy, actions, and process exit.
- Windows clean success closes progress after clipboard delivery and exits without attempting Toast delivery or presenting a Task Dialog, Message Box, Desktop window, or other acknowledgment. Windows notification identity and Toast code become obsolete.
- Rejection, cancellation, warning, failure, `Copy Final URL`, `Retry`, and `Open Upit` preserve their existing native interactive feedback and opaque-action contracts.
- The existing platform-neutral File Manager Integration module remains the Desktop seam. The Windows adapter changes from package inspection to Classic Verb inspection and lifecycle operations.
- `Registered` requires the exact expected verb key, display name, icon, `MultiSelectModel`, command string, active payload path, an existing regular helper file, and absence of the obsolete Upit package registration.
- `Needs Repair` requires an intact required payload with absent or stale verb registration or a remaining obsolete package route. `Reinstall Upit` requires missing or damaged required payload. Signing status is not part of state classification.
- Passive inspection never mutates registration. Explicit Repair rewrites the exact current-user verb, removes the obsolete package route when present, notifies Explorer that associations changed, re-inspects, and succeeds only when state becomes `Registered`; a failed legacy cleanup remains visible and is never reported as success.
- The current installer-only lifecycle commands may remain private implementation entry points, but no user-facing self-registration or portable distribution command is added.
- The consumer deliverable remains one versioned NSIS installer. Desktop, CLI, and the one-shot helper remain private compatible payload members under one Upit version.
- The Explorer COM adapter, C++ build, sparse MSIX, Appx manifest, package identity, retained MSIX repair material, MSIX payload trust checks, and MakeAppx dependency are removed completely.
- Ordinary package builds are unsigned and fully functional. Existing protected-tag certificate inputs may optionally Authenticode-sign binaries and the installer; signing remains restricted to protected SemVer release context and does not change runtime behavior.
- Installation and update keep versioned staging and use this transaction: stage the new payload; write the Classic Verb pointing to the new helper; inspect it as `Registered`; switch active product metadata while retaining the previous payload; request removal of the obsolete package registration; inspect whether that package identity is absent; commit only when the Classic Verb is registered and the obsolete package is absent; then remove the previous payload.
- A brief classic-and-packaged overlap is permitted only between Classic Verb verification and legacy-package inspection. If Classic Verb creation or inspection fails, the installer removes any partial Classic Verb and leaves the previous route and metadata unchanged. If legacy removal fails and inspection still finds the old package, the installer removes the Classic Verb, restores previous product metadata, retains both payloads needed for recovery, and aborts. If the old package is observed absent, the cutover commits even when the removal command reported an error. If package state cannot be inspected, the installer retains both payloads, reports an unresolved cutover, and does not claim success.
- Uninstall removes the exact Classic Verb and any remaining obsolete Upit package registration before deleting shortcuts, metadata, or payload. Cleanup failure aborts and retains payload.
- Registry creation/removal notifies Explorer of association changes so users do not need to sign out or restart Explorer under ordinary conditions.
- Historical v0.6 specifications and research remain unchanged as evidence. Current product, architecture, packaging, and release documentation is updated to ADR 0021.
- No Configuration Set schema, upload protocol, history, queue, resident process, or background daemon is introduced.

## Testing Decisions

- Permanent tests assert consumer-visible state, transitions, registry contract, lifecycle safety, and installed behavior. They do not assert source text, implementation call ordering, generated bindings, or mock echoes.
- The highest deterministic behavior seam is the existing File Manager Integration module. Its tests prove state classification, action availability, explicit-only Repair, post-action reinspection, failure isolation, and fail-closed behavior.
- The Windows adapter has one internal registry seam with an isolated current-user test root. Adapter tests exercise real registry reads and writes against that root and prove exact display name, icon, `MultiSelectModel`, quoted command, active-helper validation, Repair, removal, and rollback states.
- Registry tests use temporary payload directories and actual filesystem entries. They distinguish intact helper, missing helper, stale command path, wrong values, absent key, and failed mutation.
- Existing File Manager Upload tests remain authoritative for regular-file validation, changed-file rejection, Configuration Set loading, upload execution, Shortener fallback, clipboard behavior, cancellation, single-flight, privacy, recovery actions, and process exit.
- Helper terminal-feedback tests prove Windows clean success closes progress and exits silently without notification delivery or modal fallback, while every actionable non-clean outcome remains interactive. macOS notification tests remain unchanged.
- Package verification runs the public package command and proves the consumer output contains Desktop, CLI, and helper; omits the Explorer DLL, MSIX repair material, and separate helper application entries; emits public version metadata and checksum sidecars; and cleans temporary staging.
- Package verification proves ordinary unsigned output is installable and does not invoke or require CMake, a C++ compiler, MakeAppx, package publisher identity, or signing credentials. Optional protected signing remains separately verified when credentials are present.
- Native Windows 11 x64 smoke installs the actual unsigned NSIS consumer artifact into a current-user location and inspects the actual canonical registry key.
- Native smoke opens File Explorer and requires operator-observed **Show more options** placement, visibility for exactly one file, suppression for multiple files and folders, and invocation through the registered verb rather than direct helper execution.
- Native smoke exercises a real local upload and observes privacy-safe progress, cancellation, silent clean success after clipboard delivery, warning/failure/recovery feedback, helper exit, no Desktop window, and unchanged CLI and Manual Upload behavior.
- Native smoke deliberately damages registration, proves Desktop reports `Needs Repair`, runs explicit Repair, and proves state becomes `Registered` and the Explorer command works again.
- Native update smoke proves the new payload is staged before switch, the Classic Verb is verified before legacy cleanup, any dual-route overlap exists only during the transaction, successful completion leaves only the new helper registration, and the prior payload is removed only after commit.
- Failure-path smoke proves failed Classic Verb registration leaves the previous route unchanged; failed legacy cleanup with the old package still present removes the Classic Verb and restores previous metadata; an observed-absent old package commits the cutover despite a removal-command error; unknown package state retains both payloads and reports failure; failed uninstall cleanup retains callable payload rather than leaving a stale action.
- Legacy migration smoke begins with an actual registered old Upit package identity and primary-menu route, installs the Classic Verb build, observes the bounded transactional overlap, and proves successful completion leaves only the Classic Verb. A mocked package query is not sufficient.
- The native unsigned functional smoke is authoritative for Windows integration behavior. Optional signed release checks prove Authenticode trust only; signing is not required to claim functional registration.
- Full Go tests, Go vet, frontend typecheck/build, race-sensitive helper tests, package smoke, and unchanged-platform checks remain required. Cross-compilation alone is not native Windows proof.

## Out of Scope

- Windows 11 primary-menu placement, packaged `IExplorerCommand`, sparse MSIX, package identity, or dual classic/modern registration.
- Windows 10 support, Windows ARM64, machine-wide installation, elevation, or all-user registry keys.
- Folder upload, folder-background commands, directory archiving, multi-file upload, batch queues, or one process per selected file.
- A portable archive, user-facing self-registration command, second installer, or separate File Manager Integration product.
- Mandatory code signing, weakening protected release-secret boundaries, SmartScreen reputation guarantees, WDAC policy bypass, or AppLocker policy bypass.
- Windows clean-success Toast, a custom notification surface, a blocking clean-success acknowledgment, notification settings, or notification history.
- Changes to macOS Finder Services or macOS clean-success notifications.
- Changes to Configuration Set schemas, Uploader or Shortener behavior, CLI output, Manual Upload behavior, upload history, or recovery-token lifetime.
- Rewriting historical v0.6/v0.7 specifications to pretend the original packaged route was never shipped.

## Further Notes

- Primary-source Windows research is recorded in `research/windows-classic-verbs.md` beside this specification.
- The Windows 11 secondary-menu placement is an explicit product choice, not a temporary fallback.
- Classic Verb registration itself does not require signing, but enterprise WDAC/AppLocker policies may still block unsigned executables and SmartScreen may still warn about downloaded unsigned installers. Those operating-system policies are not bypassed by this design.
- `Registered` proves the structural state Upit can inspect. It does not prove that File Explorer is currently rendering the menu item; the native smoke and user guidance cover the real surface.
