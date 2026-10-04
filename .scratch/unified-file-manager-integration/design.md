# Unified File Manager Integration — Confirmed Design

Status: confirmed-for-specification

This document records approved future behavior. The repository does not yet satisfy the clipboard, Desktop control-surface, consumer installation, repair, naming, or release-proof changes below.

## Problem

Upit already has shared File Manager Upload behavior and native Windows/macOS adapters, but the repository and installation flows expose Desktop and the helper as if they were separate products. Desktop has no control surface for integration status or repair, consumer installation remains script-oriented, and the current File Manager Upload clipboard behavior conflicts with the newly confirmed direct-upload contract.

## Product Boundary

- Upit is one user-visible product with one product version.
- macOS and Windows retain platform-specific packages, native adapters, private one-shot helpers, signing, and native smoke gates.
- Finder/Explorer adapters and the upload helper are private implementation components. They have no independent app entry, installer, product version, or user-facing component name.
- All user-visible package, notification, and service-provider surfaces use `Upit`; the file-manager action is `Upload with Upit`.
- The CLI remains a separate first-class interface and is not merged into the Wails runtime.

## Supported Platforms

- macOS 14 or newer on Apple Silicon.
- Windows 11 x64.
- Linux file-manager integrations, Intel Macs, older macOS, Windows 10, and Windows ARM64 remain outside this feature.

Upit Desktop remains supported on Linux. Its fifth sidebar area is visible but presents `Not supported on Linux`, explains that no Linux File Manager Integration is included, and exposes no registration, Repair, or file-manager verification action.

## File Manager Upload Contract

1. Installation registers File Manager Integration before Configuration Set readiness is known.
2. An invocation accepts exactly one selected regular file.
3. It reloads and strictly validates the current Configuration Set before network work.
4. It uses the default Uploader and optional default Shortener from Global Configuration. Without a Shortener, the Original URL is the Final URL.
5. It always attempts to copy the Final URL. The `copyToClipboard` setting remains the default only for CLI and Manual Upload.
6. Clipboard failure preserves the completed upload as success with a privacy-safe warning and offers `Copy Final URL` against the existing result. It never repeats the upload merely to retry copying.
7. It does not open or focus Upit Desktop during ordinary execution.
8. Existing privacy-safe progress, Cancel, single-flight rejection, explicit Retry, Shortener fallback, opaque action tokens, and short-lived private action state remain unchanged.
9. A clean success—upload completed without warnings and the Final URL was copied—closes active progress feedback and emits one distinct silent native notification per invocation through `UNUserNotificationCenter` on macOS and Windows Toast on Windows 11. Its title is `Upload complete`, its body is `Final URL copied to clipboard.`, selecting it performs no Upit action, Upit does not replace or aggregate it, and the operating system controls display duration, history, and grouping.
10. If clean-success notification delivery is unavailable, the helper exits silently because the Final URL is already in the clipboard. Rejection, cancellation, warnings, failures, and recovery actions retain their existing interactive feedback and native alert fallbacks.
11. Configuration Set failure may offer `Open Upit` so Setup or Repair can be completed deliberately.

## Desktop Control Surface

Add `File Manager Integration` as a dedicated fifth sidebar area beside Manual Upload, Global Configuration, Uploaders, and Shorteners.

The area remains accessible when the Configuration Set is in Setup or Repair mode. File Manager Integration failure never blocks Configuration Set management or Manual Upload.

On Linux, the same area is informational only: status `Not supported on Linux`, no Repair action, and no implication that a file-manager adapter is installed.

### Separate state dimensions

Desktop presents Configuration Set readiness and File Manager Integration status independently. It must not collapse them into one generic readiness state.

### Truthful integration states

| Verifiable condition | Presented state | Available action |
| --- | --- | --- |
| Required package/bundles and private payload are present; OS registration is discoverable | `Registered` | Platform guidance and open-file-manager instructions |
| Private payload is intact but OS registration is absent or stale | `Needs Repair` | `Repair Integration` |
| Required adapter/helper payload is missing or damaged | `Reinstall Upit` | Open reinstall guidance; registration repair cannot recreate binaries |
| Linux Desktop, where this feature provides no file-manager adapter | `Not supported on Linux` | No Repair or registration action |

`Registered` means only that verifiable package and registration state is intact. It never means that Finder or Explorer is currently rendering the action.

### Repair policy

- Repair is explicit and user-triggered. Desktop does not silently mutate OS registration during ordinary startup.
- Repair re-registers only the current installed product and then refreshes the displayed state.
- Windows installation must retain the signed registration material required for current-user repair; the existing external `PackagePath` assumption is insufficient for self-contained Desktop repair.
- Missing or damaged payload requires reinstalling Upit rather than attempting a partial download or hidden repair service.
- Windows Repair is available only when the installed product can locate an intact signed registration package, the matching external payload, and a trusted signing identity. Missing, unsigned, untrusted, or mismatched repair inputs produce actionable reinstall guidance rather than a false successful state.
- Unsigned verification builds may exercise detection and failure behavior, but they must not present production registration or Repair as available.

### Verification guidance

Desktop does not provide a fake `Test File Manager Upload` button that invokes the helper directly. Such a call would test upload behavior but not Finder/Explorer registration.

Instead, the area provides bounded instructions and an action to open Finder or Explorer so the user can right-click one regular file and invoke `Upload with Upit` through the real OS surface.

## macOS Delivery

- Distribute one Developer ID-signed, Hardened Runtime, notarized, and stapled `Upit.app` in a DMG.
- The user installs by dragging `Upit.app` into `/Applications`; running `install.sh` is not part of the consumer flow.
- `Upit.app` contains the background-only `NSServices` provider and private one-shot helper.
- Launch Services may discover the nested Service after the app is copied. On first Desktop launch, Upit performs a non-mutating bundle and registration check; if registration is absent or stale while the bundles are intact, Desktop reports `Needs Repair` and waits for the user to invoke re-registration.
- App Sandbox, Finder Sync, Share Extensions, Action Extensions, Automator workflows, and Mac App Store distribution remain rejected because they conflict with the fixed Configuration Set access model or product runtime model.
- macOS exposes `Upload with Upit` through Finder Services or Quick Actions; a first-level Finder context-menu position is not required.
- Public APIs do not reveal actual Services enablement or Finder menu visibility. Desktop must not read private `pbs` preferences or claim an `Active in Finder` state.
- The Integration area provides non-blocking guidance and `Open Keyboard Settings`, instructing the user to navigate to Keyboard Shortcuts → Services → Files and Folders when the action is not visible.
- The product provides an explicit `Prepare to Remove Upit` action in Desktop. After confirmation it unregisters the nested Service with Launch Services, refreshes dynamic services, exits Desktop, and directs the user to remove `Upit.app`; deleting the bundle without this path is not treated as proven immediate cleanup.

## Windows Delivery

- Distribute one signed Upit installer/package containing Desktop, the private File Manager Upload helper, and the native `IExplorerCommand` adapter.
- Installation and update register or refresh `Upload with Upit` in the Windows 11 primary File Explorer context menu without requiring a consumer PowerShell flow.
- Uninstallation removes package identity and Explorer registration with the product.
- Desktop may verify current-user package registration and required payload presence, but it must not claim that Explorer is currently painting the menu item.
- With intact payload and retained signed registration material, `Repair Integration` performs current-user re-registration without elevation. Missing payload requires reinstalling Upit.
- Repair must fail closed with reinstall guidance when the signed registration package is unavailable, its signature is untrusted, or its external payload does not match the installed product.

## Installation and Update Invariants

- One install or update delivers mutually compatible Desktop, adapter, and helper versions.
- Platform components share the Upit product version and update atomically.
- Installation registers File Manager Integration; Configuration Set Setup is not a registration prerequisite.
- Updates refresh registration without creating duplicate commands or separate app identities.
- Uninstall removes File Manager Integration and leaves no usable stale action.

## Out of Scope

- One executable or one process for Desktop and File Manager Upload.
- Opening Wails or a WebView for a file-manager invocation.
- User-visible helper applications or separate integration installers.
- Enable/disable toggles that pretend to control unobservable macOS Services state.
- Polling Finder or Explorer for menu visibility.
- Direct-helper test buttons presented as end-to-end integration tests.
- Multi-file or directory upload, batching, queues, concurrency, history, automatic retry, scheduling, or new upload protocol behavior.
- A separate File Manager clipboard setting.

## Repository Delta

The next specification must cover these concrete changes:

1. Change File Manager Upload from `ClipboardFromConfig` to an always-copy policy while retaining nonfatal copy failure and `Copy Final URL` recovery.
2. Clarify the Global Configuration clipboard editor text so it applies to CLI and Manual Upload, not File Manager Upload.
3. Replace blocking clean-success completion with the ADR 0020 contract: close progress feedback, use silent `UNUserNotificationCenter` and Windows Toast notifications, and remove the clean-success modal fallback while preserving actionable outcomes.
4. Add platform-neutral integration state and repair boundaries to the Wails-free application layer, with thin Wails/native adapters.
5. Add the dedicated Desktop sidebar area, independent status presentation, repair actions, OS guidance, and accessibility behavior.
6. Make Windows repair self-contained by retaining signed registration material in the installed product.
7. Replace consumer-facing macOS and Windows script installation with one-product installation flows while retaining scripts as development/package internals where useful.
8. Remove user-visible `Upit File Manager Upload` / `Upit Finder Service` naming from system surfaces without hiding component names from technical documentation.
9. Preserve existing native behavior tests and add consumer-visible tests for the new clipboard, clean-success notification, state, repair, naming, and failure-isolation contracts.
10. Complete signed Windows native smoke and signed/notarized/stapled macOS native smoke before claiming production readiness.
11. Add explicit Linux Desktop behavior: visible informational area, `Not supported on Linux`, and no Repair or registration actions.
12. Add the macOS user-facing unregister-before-removal flow and prove that the Service disappears without relying on eventual Launch Services cleanup.
13. Prove Windows Repair rejects missing, unsigned, untrusted, or mismatched registration inputs and directs the user to reinstall.

## Decision Records and Domain Language

- `CONTEXT.md` defines File Manager Upload, File Manager Integration, and the narrowed Global Configuration clipboard scope.
- ADR 0014 preserves the one-shot Wails-free helper boundary.
- ADRs 0015 and 0016 preserve the Windows `IExplorerCommand` and macOS `NSServices` routes.
- ADR 0017 records the always-copy File Manager Upload contract.
- ADR 0018 records one visible Upit product with private helpers and truthful Desktop controls.
- ADR 0019 records one atomic product version and installer identity.
- ADR 0020 records native, silent, non-activating clean-success notifications and the absence of a modal fallback after successful clipboard delivery.

The next `to-spec` workflow should use this document as its confirmed design input and incorporate ADR 0020. It must explicitly supersede the v0.6/v0.7 clauses that made File Manager Upload honor Global Configuration clipboard behavior and narrow native alert fallback so a clean copied success never requires acknowledgment, while preserving the remaining completed upload, privacy, actionable feedback, helper-lifecycle, and platform-adapter contracts.
