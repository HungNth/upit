# Upit — Unified File Manager Integration

Status: ready-for-agent

## Problem Statement

A user can configure Uploaders and optional Shorteners in Upit Desktop, but File Manager Integration is still presented and delivered as if it were a separate product. Consumer installation remains script-oriented, Desktop cannot show whether the operating-system registration is intact or help repair it, and helper names can appear as separate applications even though they are implementation components of Upit.

File Manager Upload also follows the Global Configuration clipboard preference today. That behavior conflicts with the confirmed direct-upload workflow: a file-manager invocation has no result view, so its Final URL must always be delivered through the clipboard without opening Manual Upload. The existing Windows and macOS upload behavior, privacy protections, progress, cancellation, recovery actions, and one-shot process boundary should remain authoritative rather than being rebuilt.

## Solution

Release one versioned Upit product on Windows and macOS. Each platform package contains Upit Desktop, its native file-manager adapter, and the private one-shot File Manager Upload worker. Installation registers `Upload with Upit`; updates refresh the registration; uninstallation removes it. The helpers remain private package components and are not presented as separate user-facing applications, installers, or versions.

Add a dedicated File Manager Integration area to Upit Desktop. It reports only package, payload, and registration state that Upit can verify, provides explicit user-triggered Repair when the installed repair inputs are intact, directs the user to reinstall when they are not, and never claims that Finder or Explorer is currently displaying the action. File Manager Upload continues to use the default Uploader and optional default Shortener, but always attempts to copy the Final URL regardless of the Global Configuration clipboard preference.

## User Stories

1. As an Upit user, I want to install one product, so that Desktop and File Manager Integration do not appear to be separate applications.
2. As an Upit user, I want one product version, so that Desktop, native adapters, and private helpers are always compatible.
3. As an Upit user, I want one installer or application bundle per platform, so that I do not need to install File Manager Integration separately.
4. As an Upit user, I want the product to be named `Upit` in operating-system surfaces, so that internal component names do not look like additional applications.
5. As an Upit user, I want the file-manager action named `Upload with Upit`, so that its purpose and product ownership are clear.
6. As an Upit user, I want installation to register File Manager Integration, so that no terminal command or manual registry edit is required.
7. As an Upit user, I want updates to refresh File Manager Integration without duplicate commands, so that the action remains usable after upgrading.
8. As an Upit user, I want uninstallation to remove File Manager Integration, so that no stale action remains after Upit is removed.
9. As a user, I want File Manager Upload to accept exactly one selected regular file, so that each invocation has one deterministic result.
10. As a user, I want empty, multiple, directory, and non-regular selections rejected before network work, so that unsupported input cannot create partial work.
11. As a user, I want the selected path or file URL treated as untrusted, so that a native handoff cannot bypass regular-file validation.
12. As a user, I want File Manager Upload to reload the current Configuration Set before each operation, so that it never uses stale Desktop state.
13. As a user, I want File Manager Upload to use the default Uploader, so that the direct action follows my configured destination.
14. As a user, I want File Manager Upload to use the optional default Shortener, so that its Final URL follows my configured post-processing choice.
15. As a user without a default Shortener, I want the Original URL to become the Final URL, so that shortening remains optional.
16. As a user, I want File Manager Upload to always attempt to copy the Final URL, so that the direct action produces a paste-ready result without a result window.
17. As a user whose Global Configuration clipboard preference is disabled, I still want File Manager Upload to copy its Final URL, so that this direct surface keeps its fixed delivery contract.
18. As a CLI or Manual Upload user, I want the Global Configuration clipboard preference to remain available for those surfaces, so that the File Manager Upload exception does not change unrelated workflows.
19. As a user whose automatic clipboard copy fails, I want the completed upload preserved as success with a warning, so that Upit does not invalidate irreversible remote work.
20. As a user whose automatic clipboard copy fails, I want `Copy Final URL` to retry copying the existing result, so that Upit does not upload the file again.
21. As a user, I want ordinary File Manager Upload to avoid opening or focusing Upit Desktop, so that the action remains direct and unobtrusive.
22. As a user, I want native preparation, phase, and measurable byte progress feedback, so that I can observe long uploads without opening Desktop.
23. As a user, I want to cancel an active File Manager Upload from native feedback, so that unwanted work can stop without a process manager.
24. As a user, I want at most one active File Manager Upload per user, so that progress, clipboard writes, and native feedback cannot race.
25. As a user starting a second File Manager Upload, I want it rejected without queueing or an endpoint request, so that hidden work does not accumulate.
26. As a user whose Shortener fails after upload, I want the Original URL retained as the Final URL with a warning, so that completed upload work is preserved.
27. As a user whose runtime upload fails, I want an explicit Retry action, so that no potentially non-idempotent upload is repeated automatically.
28. As a user choosing Retry, I want Upit to revalidate the file and current Configuration Set before one new request, so that stale inputs are not reused.
29. As a user whose Configuration Set is absent or invalid, I want a sanitized failure with `Open Upit`, so that I can enter Setup or Repair deliberately.
30. As a privacy-conscious user, I want native feedback to omit file names, paths, endpoints, request values, response bodies, Original URLs, Final URLs, and credentials, so that lock-screen and notification content remains private.
31. As a privacy-conscious user, I want Copy, Retry, and recovery actions to use opaque tokens and short-lived private state, so that private values are not exposed as action arguments.
32. As a privacy-conscious user, I want action state consumed on use or removed after its existing expiry, so that recovery does not become upload history.
33. As a user denied notification permission, I want the existing native alert fallback for otherwise invisible terminal outcomes, so that File Manager Upload never fails silently.
34. As a Desktop user, I want a dedicated File Manager Integration sidebar area, so that configuration and integration management are available in one control surface.
35. As a Desktop user, I want the File Manager Integration area available during Configuration Set Setup or Repair, so that the two concerns remain independent.
36. As a Desktop user, I want Configuration Set readiness and File Manager Integration status shown separately, so that I know which concern needs attention.
37. As a Desktop user with intact payload and discoverable registration, I want the status `Registered`, so that Upit reports the state it can verify.
38. As a Desktop user, I want `Registered` to avoid claiming that Finder or Explorer currently displays the action, so that the status remains truthful.
39. As a Desktop user with intact payload but missing or stale registration, I want `Needs Repair`, so that the recoverable problem is explicit.
40. As a Desktop user, I want Repair to run only after I request it, so that Upit does not silently mutate operating-system registration during startup.
41. As a Desktop user, I want Repair to refresh the displayed state after completion, so that I can see whether registration was restored.
42. As a Desktop user with missing or damaged adapter/helper payload, I want `Reinstall Upit`, so that Repair does not pretend it can recreate binaries.
43. As a Desktop user, I want File Manager Integration failure not to block Manual Upload or Configuration Set management, so that a damaged adapter does not disable working product surfaces.
44. As a Desktop user, I want bounded instructions and an action to open Finder or Explorer, so that I can verify the real operating-system invocation.
45. As a Desktop user, I do not want a helper-direct `Test File Manager Upload` control presented as an end-to-end test, so that upload success is not confused with menu registration success.
46. As a macOS user, I want to install Upit by dragging one notarized `Upit.app` from its DMG into Applications, so that installation follows the normal macOS product model.
47. As a macOS user, I do not want to run an installation shell script, so that File Manager Integration remains part of the desktop product.
48. As a macOS user, I want first launch to inspect bundle and Launch Services registration without changing it, so that startup remains non-mutating.
49. As a macOS user whose Service registration is absent or stale, I want Desktop to show `Needs Repair`, so that I can choose re-registration explicitly.
50. As a macOS user, I want `Upload with Upit` under Finder Services or Quick Actions, so that I can invoke File Manager Upload through supported Finder surfaces.
51. As a macOS user, I want guidance when the Service may need enabling in System Settings, so that I can find it despite macOS user preferences.
52. As a macOS user, I want an `Open Keyboard Settings` action with the exact navigation path, so that unsupported deep-link assumptions do not leave me stranded.
53. As a macOS user, I do not want Desktop to read private Services preferences or claim live Finder visibility, so that Upit depends only on supported observable state.
54. As a macOS user removing Upit, I want an explicit `Prepare to Remove Upit` flow, so that the nested Service is unregistered before the application bundle is removed.
55. As a macOS user, I want removal to refresh dynamic services and leave no usable stale action, so that Finder cleanup is deterministic rather than eventual.
56. As a Windows user, I want one signed Upit installer/package, so that Desktop and the Explorer command are installed together.
57. As a Windows user, I do not want to run PowerShell to install File Manager Integration, so that the consumer flow is a normal product installation.
58. As a Windows user, I want `Upload with Upit` in the Windows 11 primary File Explorer context menu, so that the direct action is easy to reach.
59. As a Windows user, I want Desktop to verify current-user package registration and required payload presence, so that status is based on concrete evidence.
60. As a Windows user with intact trusted repair inputs, I want user-triggered Repair without elevation, so that registration can be restored without reinstalling.
61. As a Windows user whose signed registration package is missing, unsigned, untrusted, or mismatched, I want Repair to fail closed with reinstall guidance, so that Upit never reports a registration it could not establish safely.
62. As a user of an unsigned verification build, I want production registration and Repair presented as unavailable, so that development artifacts are not misrepresented as releasable packages.
63. As a Windows user uninstalling Upit, I want package identity and Explorer registration removed together, so that no orphaned command remains.
64. As a Linux Desktop user, I want the File Manager Integration area to show `Not supported on Linux`, so that the absence of an adapter is explicit.
65. As a Linux Desktop user, I want no registration, Repair, or file-manager verification action, so that Desktop does not imply unsupported integration exists.
66. As a maintainer, I want File Manager Upload behavior to remain in the shared Wails-free application layer, so that native adapters do not duplicate upload policy.
67. As a maintainer, I want File Manager Integration inspection and Repair behind one platform-neutral application seam, so that Desktop does not contain Windows or macOS registration policy.
68. As a maintainer, I want native adapters limited to platform inspection, registration, settings navigation, and removal operations, so that the product boundary stays testable.
69. As a maintainer, I want Desktop, adapter, and helper components to update atomically under one Upit version, so that compatibility matrices are unnecessary.
70. As a release maintainer, I want Windows signing and native smoke to gate Windows production readiness, so that package and Explorer behavior are proven on the target platform.
71. As a release maintainer, I want Developer ID signing, Hardened Runtime, notarization, stapling, and native Finder smoke to gate macOS production readiness, so that Gatekeeper and Finder behavior are proven on the target platform.
72. As a maintainer, I want historical v0.6 and v0.7 specifications retained as evidence of their released contracts, so that this new clipboard and product-integration decision is recorded as a deliberate superseding change rather than a rewritten history.

## Implementation Decisions

- This specification defines the post-v0.7 unified product contract. It supersedes only the v0.6/v0.7 clauses that made File Manager Upload honor the Global Configuration clipboard preference and the script-oriented/separate-product presentation. Existing one-file selection, upload, privacy, native feedback, cancellation, single-flight, recovery-action, helper-lifecycle, and platform-adapter contracts remain authoritative.
- Upit uses one user-visible product identity and one product version. Platform packages may contain multiple executables, but adapters and the one-shot worker are private package components without independent user-facing application entries, installers, or versions.
- The existing Wails-free `FileManagerUploadService` remains the authoritative behavior seam for File Manager Upload. Native Finder and Explorer adapters continue to hand one untrusted selection to that service through the private one-shot worker.
- File Manager Upload forces clipboard copying instead of selecting clipboard behavior from Global Configuration. No Configuration Set schema field is added, removed, or migrated.
- The existing `copyToClipboard` field remains the default clipboard policy for CLI and Manual Upload. Desktop copy must label that scope explicitly so users are not led to expect it to control File Manager Upload.
- Automatic clipboard failure remains nonfatal. The result retains `Copy Final URL` recovery backed by the existing Final URL and action-token lifetime; Retry remains reserved for a fresh upload attempt after runtime failure.
- Add one Wails-free `FileManagerIntegrationService` as the highest product seam for integration management. It owns platform selection, state classification, user-triggered Repair orchestration, settings/file-manager navigation outcomes, and removal preparation while depending on a narrow platform adapter for operating-system operations.
- The integration state model contains exactly four user-visible states: `Registered`, `Needs Repair`, `Reinstall Upit`, and `Not supported on Linux`.
- `Registered` requires intact expected payload and discoverable operating-system registration. It does not assert user enablement, context-menu rendering, or successful end-to-end invocation.
- `Needs Repair` requires intact expected payload plus absent or stale registration and available trustworthy repair inputs.
- `Reinstall Upit` covers missing or damaged payload and any platform state where safe registration repair inputs are unavailable.
- `Not supported on Linux` is returned by the Linux adapter without registration or Repair capabilities.
- Desktop adds File Manager Integration as a dedicated fifth sidebar area. The area is not disabled by Configuration Set Setup or Repair and does not block other Desktop functions.
- Desktop reads integration state on area entry, when the window regains focus, and after an explicit action. It does not poll Finder or Explorer.
- Desktop offers only actions supported by the current state and platform: Repair, open operating-system settings, open the file manager for user verification, reinstall guidance, or prepare removal. It exposes no synthetic enable/disable toggle and no helper-direct test presented as end-to-end proof.
- macOS remains an un-sandboxed Developer ID distribution outside the Mac App Store. The fixed Configuration Set location remains unchanged.
- The macOS package contains one outer Upit application, a background-only `NSServices` provider, and the private one-shot worker. User-visible names resolve to `Upit` or `Upload with Upit`.
- macOS drag installation relies on Launch Services discovery but does not assume immediate registration. First launch performs a non-mutating inspection and offers Repair when appropriate.
- macOS Repair re-registers the installed outer application and nested Service, refreshes dynamic services, and then re-inspects state. It never writes private Services preference data.
- macOS cannot publicly observe actual user enablement or Finder menu visibility. Desktop provides non-blocking instructions and opens the Keyboard settings pane on a best-effort basis with explicit manual navigation guidance.
- macOS provides an explicit confirmed `Prepare to Remove Upit` flow that unregisters the nested Service and outer application, refreshes dynamic services, exits Desktop, and directs the user to remove the application bundle.
- Windows uses one signed installer/package that installs Desktop, the private worker, native `IExplorerCommand` adapter, package identity, and the signed registration artifact required for repair.
- Windows consumer installation and update do not require PowerShell. Existing packaging scripts may remain internal build and verification tools.
- Windows Repair is available only when the signed registration artifact, matching external payload, and trusted signing identity can be verified. It runs in the current-user context and fails closed to reinstall guidance for missing, unsigned, untrusted, or mismatched inputs.
- Unsigned verification artifacts expose detection and negative-path verification only; they do not claim production registration or Repair availability.
- Updates replace compatible components atomically, retain one product identity, and refresh registration without duplicate commands.
- Uninstall removes package identity, native registration, and user-visible integration. macOS deterministic removal uses the explicit prepare-removal flow rather than treating bundle deletion as immediate Launch Services proof.
- No background daemon, system tray process, resident worker, upload queue, or persistent history is introduced.

## Testing Decisions

- Permanent tests assert consumer-visible behavior, boundaries, state transitions, privacy, and failure recovery. They do not pin source layout, exact helper command lines, generated bindings, copied values between adapters, or operating-system implementation details that are already proven at a higher seam.
- Two high behavioral seams are used because upload execution and installed-integration management have distinct lifecycles:
  1. The existing Wails-free `FileManagerUploadService` proves selection, current Configuration Set use, default Uploader/optional Shortener behavior, always-copy policy, warnings, actions, cancellation, single-flight, privacy, and result preservation.
  2. The new Wails-free `FileManagerIntegrationService` proves state classification, action availability, explicit Repair, fail-closed reinstall guidance, Linux behavior, failure isolation, and removal preparation through deterministic platform-adapter doubles.
- File Manager Upload tests continue to use temporary user homes, real Configuration Set documents, real local HTTP endpoints, deterministic clipboard adapters, controllable contexts/readers, and the existing action store. This follows the current File Manager Upload test prior art.
- Add a regression test where Global Configuration has clipboard copying disabled but File Manager Upload still copies the Final URL.
- Add a regression test where automatic copying fails after a successful upload and the result is success-with-warning with one usable `Copy Final URL` action. Dispatching that action copies the existing Final URL without a second endpoint request.
- Preserve Shortener fallback tests: an ordinary Shortener failure retains the Original URL as Final URL, attempts to copy it, and reports a privacy-safe warning.
- Integration-service state tests cover every state and transition: intact/registered, intact/unregistered, repair success, repair failure, missing payload, invalid repair inputs, post-repair reinspection, and Linux unsupported.
- Integration-service tests prove Configuration Set readiness does not alter integration state and integration failure does not block Manual Upload or configuration management.
- Integration-service tests prove Repair never starts from passive inspection or Desktop startup; only an explicit user action may invoke it.
- Windows adapter tests use controlled package-query, payload, signature/trust, and registration boundaries. They prove unsigned, untrusted, missing, and mismatched repair inputs return reinstall guidance and never report `Registered` after failed work.
- macOS adapter tests prove bundle inspection, Launch Services discovery classification, explicit re-registration, settings-navigation fallback, and unregister-before-removal ordering. They do not read private `pbs` data or assert live Finder visibility.
- Linux adapter tests prove the stable `Not supported on Linux` state and absence of Repair, registration, settings, or verification actions.
- Desktop frontend tests cover the fifth navigation area, accessibility names and live status, independent Configuration Set and integration state, action visibility by state/platform, non-blocking Setup/Repair access, and scoped clipboard-setting copy.
- Native Finder and Explorer adapter tests remain narrow: exact-one selection handoff, private component launch, action dispatch, and no Desktop window. They do not duplicate upload or integration-management policy.
- Packaging verification proves one user-visible product name/version, required private payload, absence of separate application entries, retained Windows repair material, macOS nested bundle metadata, and clean install/update/uninstall registration contracts.
- Windows 11 x64 signed native smoke installs the consumer package, verifies the primary Explorer action, performs a real one-file upload against a local endpoint, observes privacy-safe progress/cancellation/recovery, exercises Desktop state and Repair, updates without duplication, and uninstalls without a stale action.
- macOS 14+ Apple Silicon signed/notarized/stapled native smoke drags Upit into Applications, observes first-launch non-mutating status, repairs registration when deliberately removed, verifies Finder Services or Quick Actions, performs real upload and recovery paths, exercises System Settings guidance, prepares removal, and proves the Service disappears.
- Linux Desktop smoke verifies the File Manager Integration area reports `Not supported on Linux`, exposes no unsupported actions, and leaves Manual Upload and Configuration Set management operational.
- Native smoke is required evidence for operating-system menu visibility. A direct helper invocation, package inspection, unit test, or compile check is not accepted as an end-to-end substitute.
- Protected release jobs alone receive signing and notarization credentials. Ordinary CI verifies unsigned package shape and negative Repair behavior without claiming production readiness.

## Out of Scope

- Merging Desktop, native adapters, and File Manager Upload into one executable or process.
- Starting Wails, a WebView, a Desktop window, Dock icon, or tray process for ordinary File Manager Upload.
- Presenting native adapters or the one-shot worker as separately installable or user-launchable applications.
- Linux file-manager adapters for Nautilus, Dolphin, or other Linux file managers.
- Intel macOS, macOS versions before 14, Mac App Store distribution, App Sandbox, Finder Sync, Share Extensions, Action Extensions, or Automator workflows.
- Windows 10, Windows ARM64, classic registry shell verbs, portable shell registration, or a separate File Manager Integration installer.
- A first-level Finder context-menu requirement; Finder Services or Quick Actions are sufficient.
- Reading or writing private macOS Services preference data, polling Finder/Explorer, or claiming actual menu visibility from Desktop.
- A File Manager Integration enable/disable toggle.
- A helper-direct test button presented as end-to-end integration verification.
- Multi-file or directory upload, batching, queues, concurrent File Manager Uploads, upload history, scheduling, or automatic upload retry.
- A separate File Manager clipboard preference or Configuration Set schema migration.
- New Uploader Request Body Modes, Response Extractors, Shortener protocols, credential stores, configuration locations, or timeout policy.
- Rewriting historical v0.6 or v0.7 specifications and tickets to describe the new contract.

## Further Notes

- The confirmed design source is `design.md` in this feature directory.
- ADR 0017 records the always-copy File Manager Upload contract and nonfatal Copy recovery.
- ADR 0018 records one user-visible Upit product with private helpers and truthful Desktop controls.
- ADR 0019 records one atomic product version and installer identity.
- ADRs 0014–0016 remain authoritative for the one-shot worker, Windows `IExplorerCommand`, and macOS `NSServices` architecture.
- The current repository implements most upload and native feedback behavior but does not yet implement this specification's clipboard override, Desktop File Manager Integration area, productized consumer installation, self-contained trusted Repair, unified visible naming, deterministic macOS removal flow, or signed native release proof.
- Release credentials, trusted Windows signing identity, Apple Developer ID identity, notarization credentials, and real target-platform native smoke environments remain external prerequisites for production evidence; they are not replaced by mocks or unsigned builds.
