# Upit — Stable Windows CLI Installation and Transactional Version Layout

Status: ready-for-agent

## Problem Statement

A Windows installation currently stores the Active Payload under an opaque path such as `%LOCALAPPDATA%\Programs\Upit\versions\nsj3AB3.tmp`. Although the directory name is generated as temporary staging, it becomes the installed path used by Upit Desktop, File Manager Integration, and the packaged CLI until the next update. The name does not identify the installed product version and changes on every update.

A user who wants to invoke the installed CLI from a terminal therefore cannot add one durable directory to `PATH`: the user must discover and add a new random payload path after each update. The current installer intentionally keeps the packaged CLI private and does not modify `PATH`, but that contract no longer matches the desired Windows product experience.

Putting every executable directly in the installation root and overwriting it during updates would create a different failure mode. Windows can lock running executables, file-by-file replacement can leave a mixed payload, and an overwritten payload cannot provide intact recovery material. Upit needs a stable public CLI command without sacrificing the existing staged, verified, fail-closed installation and File Manager Integration contracts.

## Solution

On Windows, install a stable public CLI launcher at `%LOCALAPPDATA%\Programs\Upit\upit.exe` and idempotently add `%LOCALAPPDATA%\Programs\Upit` to the current user's `PATH`. The launcher resolves the Active Payload from the existing current-user product metadata, validates it, invokes the real payload CLI with unchanged command semantics, waits for completion, and returns the payload CLI's exit status.

Store the real CLI, Upit Desktop, and private File Manager helper together in one version-aligned Active Payload at `versions\X.Y.Z`. The Start Menu shortcut and File Manager Integration continue to point directly to the selected payload. Temporary staging and recovery directories remain internal installation state and never become public command paths.

Installation and update remain transactional. A new payload is fully staged and deterministically verified before commit. Different-version updates use side-by-side payloads and may defer deletion of a locked inactive payload after the new route has committed. Same-version reinstalls replace the single `versions\X.Y.Z` directory through a staged recovery swap after the user closes processes executing from the Active Payload. Any failure before commit restores the previously observed callable state.

Existing installations whose recorded Active Payload uses `versions\<random>.tmp` receive a one-time transactional cutover. A payload-specific `PATH` entry added manually by the user remains user-owned and is not removed automatically; the user must remove it because it can shadow the new stable-root entry.

## User Stories

1. As a Windows user, I want the installed `upit` command to remain at one stable location, so that product updates do not require PATH changes.
2. As a Windows user, I want the installer to make `upit` available in newly started terminals, so that I do not need to configure the stable installation root manually.
3. As a CLI user, I want `upit --help` to work without an absolute path, so that the installed product behaves like a normal command-line application.
4. As a CLI user, I want the stable command to keep working after a product-version update, so that scripts do not depend on an internal payload directory.
5. As a CLI user, I want the stable command to keep working after a same-version reinstall, so that Repair does not change my automation entry point.
6. As a Windows user, I want the installation path to remain current-user, so that installing the CLI does not require administrator elevation.
7. As a Windows user, I want the installed product version visible in its payload path as `X.Y.Z`, so that support and inspection do not mistake a random token for the version.
8. As a Windows user, I do not want an active installed payload to end in `.tmp`, so that temporary state is distinguishable from committed product state.
9. As a CLI user, I want every argument passed to the stable launcher unchanged, so that all existing CLI commands and flags retain their meaning.
10. As a CLI user, I want standard input forwarded unchanged, so that interactive and piped CLI workflows continue to work.
11. As a CLI user, I want standard output forwarded unchanged, so that scripts continue to consume the CLI result.
12. As a CLI user, I want standard error forwarded unchanged, so that diagnostics preserve the current contract.
13. As a CLI user, I want the launcher to return the real CLI exit status, so that automation detects success, warnings, cancellation, and failure correctly.
14. As a CLI user, I want cancellation and console-control behavior preserved, so that cancelling the stable command cancels the real CLI rather than leaving an orphan process.
15. As a CLI user, I want the real CLI to inherit my current working directory and environment, so that relative paths and environment-based behavior remain unchanged.
16. As a CLI user, I want paths containing spaces or Unicode characters to pass through the launcher unchanged, so that Windows path syntax does not alter the requested operation.
17. As a CLI user, I do not want the launcher to parse or reinterpret CLI output, so that the real CLI remains authoritative.
18. As a CLI user, I want an actionable nonzero failure when Active Payload metadata is absent or invalid, so that a damaged installation never launches an arbitrary executable.
19. As a security-conscious user, I want the launcher to reject an Active Payload outside the Upit installation root, so that current-user metadata cannot redirect the public command to an unrelated program.
20. As a Windows user, I do not want the launcher to guess the newest directory by scanning `versions`, so that failed staging and recovery material cannot become active accidentally.
21. As a user, I want the CLI, Upit Desktop, and File Manager helper in one version-aligned payload, so that a committed installation never mixes product versions.
22. As a user, I want one public Upit product version, so that launcher, Desktop, CLI, and File Manager Integration do not expose independent user-facing versions.
23. As a user, I want Upit Desktop to continue opening from the Start Menu, so that adding a public CLI does not change the Desktop entry point.
24. As a File Manager Upload user, I want `Upload with Upit` to continue invoking the private one-shot helper directly, so that File Manager Upload does not route through the public CLI.
25. As a File Manager Upload user, I want the Classic Verb to point to the Active Payload selected by the installer, so that new invocations use the committed product version.
26. As a Desktop user, I want a Desktop process already running from an old payload to remain distinct from the Active Payload, so that its executable location does not retarget current registration.
27. As a Windows user, I want temporary staging under an internal directory, so that incomplete extraction is never presented as an installed version.
28. As a Windows user, I want recovery directories to remain internal, so that rollback material is not mistaken for an alternate active installation.
29. As a first-time installer user, I want all required payload files staged before any public route changes, so that a partial extraction cannot become callable.
30. As a first-time installer user, I want the stable launcher, Active Payload metadata, PATH entry, Start Menu shortcut, and File Manager Integration verified before commit, so that installation success is evidence-based.
31. As a first-time installer user, I want a failed install to leave no broken public CLI route, so that `upit` does not point to missing files.
32. As a first-time installer user, I want a failed install to leave no installer-owned PATH entry, so that my environment does not retain an unusable command location.
33. As a Windows user, I want the installer to add the stable installation root to User PATH exactly once, so that repeated updates do not create duplicate entries.
34. As a Windows user, I want PATH comparison to recognize equivalent Windows paths, so that case, slash direction, or a trailing separator does not create duplicates.
35. As a Windows user, I want the installer to preserve all unrelated PATH entries and their order, so that installing Upit does not modify other tools.
36. As a Windows user with the stable root already in PATH, I want the installer to preserve that user-owned entry, so that uninstall does not remove configuration I created.
37. As a Windows user, I want new terminals notified of a PATH change through the standard Windows environment-change broadcast, so that ordinary shells observe installation and removal without a sign-out.
38. As a Windows user, I accept that an already-running terminal may retain its existing environment, so that the installer does not attempt to mutate other processes.
39. As a Windows user upgrading from one product version to another, I want the new version staged side-by-side with the Active Payload, so that the old application can remain callable until cutover is proven.
40. As a Windows user with Upit Desktop open during a different-version update, I want the update to proceed without force-closing Desktop, so that side-by-side installation avoids unnecessary interruption.
41. As a CLI user with a command already running during a different-version update, I want that process to finish from its original payload, so that an update does not replace files underneath it.
42. As a Windows user, I want new invocations to switch to the new Active Payload only after route verification, so that cutover never intentionally targets incomplete files.
43. As a Windows user, I want a committed different-version update to remain committed if the previous payload is still locked, so that cleanup does not undo a verified active route.
44. As a Windows user, I want a locked inactive payload reported as deferred cleanup rather than a clean deletion, so that retained disk state is explicit.
45. As a Windows user, I want deferred cleanup retried by a later install or uninstall, so that inactive payloads do not accumulate permanently.
46. As a Windows user, I do not want an inactive retained payload treated as an automatic runtime fallback, so that one authoritative Active Payload serves new invocations.
47. As a Windows user, I want the previous payload removed after commit when it is not locked, so that successful updates ordinarily leave only current product files.
48. As an installer user, I want unresolved pre-commit state to abort rather than report success, so that retained recovery material is not confused with a committed update.
49. As a Windows user reinstalling the same product version, I want the installer to detect processes executing from the Active Payload, so that it can explain why replacement cannot proceed.
50. As a Windows user reinstalling the same product version interactively, I want Retry and Cancel choices instead of forced termination, so that I control when active work closes.
51. As a Windows user, I do not want the installer to force-kill the CLI, Desktop, or File Manager helper, so that an installation does not destroy active work.
52. As an automation user running a silent same-version reinstall, I want a nonzero abort when required files are in use, so that automation never hangs on an invisible prompt.
53. As an installer user, I want process detection based on executable image paths rather than names alone, so that unrelated processes with similar names are not blocked.
54. As an installer user, I want filesystem rename and replacement results to remain authoritative even after process detection, so that antivirus or other file locks cannot cause a partial update.
55. As a Windows user reinstalling the same version, I want the complete replacement payload verified before the current version directory changes, so that replacement does not begin from incomplete files.
56. As a Windows user reinstalling the same version, I want the existing payload moved to recovery storage before the staged payload takes its final path, so that rollback has an intact directory.
57. As a Windows user reinstalling the same version, I want any pre-commit failure to restore the prior version directory and routes, so that the previous installation remains callable.
58. As a Windows user reinstalling the same version, I do not want file-by-file overwrite, so that failure cannot leave a mixed set of binaries.
59. As a Windows user, I do not want the installer to claim an atomic Windows directory swap, so that lock and rollback behavior is specified honestly as an ordered transaction.
60. As a Windows user, I want a stable launcher update only when its internal format actually changes, so that ordinary product updates do not replace the public entry point unnecessarily.
61. As a Windows user, I want an update that must replace the stable launcher to request closure of a running launcher, so that Windows file locks cannot corrupt the root command.
62. As a Windows user, I do not want the launcher's internal format version presented as a second product version, so that Upit remains one versioned product.
63. As an installer user, I want deterministic verification without opening Upit Desktop, so that installation does not create an unexpected application window.
64. As an installer user, I do not want verification to perform an upload, so that installation has no remote side effects.
65. As an installer user, I want all three payload binaries verified as the requested product version, so that version-directory naming matches executable metadata.
66. As an installer user, I want the real CLI exercised through the stable launcher before commit, so that the public command chain is proven rather than inferred.
67. As an installer user, I want exact Classic Verb, Active Payload, Start Menu, and PATH state inspected before commit, so that every public route resolves to the intended installation.
68. As a Windows user, I want automatic rollback limited to the installer transaction, so that later application failures do not silently switch installed versions.
69. As a Windows user, I do not want first-launch health state, background update state, or delayed runtime rollback, so that Upit remains an on-demand product without a resident updater.
70. As a Windows user, I want rollback failure reported explicitly and all available recovery material retained, so that repair does not delete the only callable payload.
71. As an existing Windows user, I want the installer to recognize the exact random `.tmp` Active Payload previously recorded by Upit, so that I can upgrade without uninstalling first.
72. As an existing Windows user, I want legacy layout cutover to use the same staged and verified transaction, so that migration does not weaken update safety.
73. As an existing Windows user, I do not want the installer to scan for arbitrary `.tmp` directories, so that abandoned staging cannot be mistaken for the installed product.
74. As an existing Windows user, I want the random `.tmp` layout removed or deferred for cleanup after a successful cutover, so that it is not retained as a supported alternate layout.
75. As an existing Windows user who manually added the old Active Payload to PATH, I want the installer to leave that entry untouched, so that it does not claim ownership of my environment edits.
76. As an existing Windows user with an old payload-specific PATH entry, I want clear guidance to remove it manually, so that it cannot shadow the new stable command or point to deleted files.
77. As a Windows user uninstalling Upit, I want File Manager Integration removed before product files, so that no context-menu action points to deleted code.
78. As a Windows user uninstalling Upit, I want the installer-owned stable-root PATH entry removed, so that `upit` does not resolve to an uninstalled product.
79. As a Windows user uninstalling Upit, I want a pre-existing stable-root PATH entry preserved, so that uninstall does not remove user-owned configuration.
80. As a Windows user uninstalling Upit, I want active, inactive, staging, and recovery payload state removed when safe, so that the product does not leave owned installation files behind.
81. As a Windows user, I want uninstall to fail closed when File Manager Integration cleanup cannot be proven, so that a surviving command retains callable payload.
82. As a Windows user, I want Manual Upload, File Manager Upload, Configuration Set behavior, clipboard behavior, and upload output unchanged, so that installation-layout work does not alter product workflows.
83. As a release maintainer, I want unsigned Windows installers to remain fully functional, so that stable CLI installation does not introduce a signing dependency.
84. As a release maintainer, I want optional Authenticode signing to cover the stable launcher and payload executables, so that protected releases preserve the existing trust model.
85. As a release maintainer, I want one Windows installer and one public `X.Y.Z` version, so that the stable CLI is not distributed as a separate product.
86. As a release maintainer, I want a native Windows 11 x64 smoke to prove the complete install, update, migration, rollback, PATH, and uninstall lifecycle, so that source inspection is not accepted as product evidence.
87. As a maintainer, I want current documentation to describe the installed CLI as public on Windows, so that it no longer claims the packaged CLI is private or absent from PATH.
88. As a maintainer, I want native smoke expectations updated from random same-version payload directories to the `versions\X.Y.Z` recovery-swap contract, so that tests enforce the accepted architecture.
89. As a maintainer, I want legacy user-owned PATH behavior documented, so that support does not promise automatic removal of manually added random payload paths.
90. As a maintainer, I want the stable launcher and installer to fail closed on invalid metadata, so that recovery logic never selects an uncommitted payload by convenience.

## Implementation Decisions

- This specification is Windows-only and implements ADR 0023. macOS and other platforms retain their current installation and CLI distribution contracts.
- The current-user installation root remains `%LOCALAPPDATA%\Programs\Upit` and requires no elevation.
- `%LOCALAPPDATA%\Programs\Upit\upit.exe` is the only stable executable entry point added by this feature. It is a launcher/forwarder, not a copied real CLI payload binary.
- The real `upit.exe`, `upit-desktop.exe`, and `upit-file-manager.exe` remain one compatible, version-aligned Active Payload under `versions\X.Y.Z`.
- The public product version remains numeric `X.Y.Z`. The Windows technical file version may remain `X.Y.Z.0`; no launcher or component version becomes user-facing.
- The final Active Payload path contains only the product version. Same-version replacement is handled transactionally rather than by adding a deployment-instance token to the final path.
- Temporary extraction uses an internal staging directory outside final version paths. Recovery uses an internal recovery directory. Neither path is added to PATH, registered as active, or exposed as an application entry point.
- Current-user `PayloadPath` product metadata remains the sole authority for the Active Payload. The launcher, Desktop inspection, Repair, installer, and uninstaller must agree on that authority.
- The stable launcher canonicalizes and validates `PayloadPath` before starting a child. The path must be inside the expected installation root, match the committed version-directory shape, contain all required regular payload files, and identify a coherent product version. Invalid state fails closed with a nonzero diagnostic; the launcher never scans for a highest or newest version.
- The launcher passes arguments without reinterpretation, inherits stdin/stdout/stderr, current working directory, environment, and console behavior, waits for the real CLI, and returns its exit code. It does not load Configuration Set data or implement CLI commands itself.
- The launcher stays attached to the same console process tree so normal Windows console cancellation reaches the real CLI and the launcher does not leave an orphan child.
- The Start Menu shortcut continues to target the real `upit-desktop.exe` in the Active Payload. File Manager Integration continues to target the real private `upit-file-manager.exe` in the Active Payload. No stable Desktop or helper launcher is introduced.
- The launcher has an internal technical format identifier used only to decide whether the root launcher must be replaced. An ordinary product update preserves an adequate existing launcher. A format-changing update stages and transactionally replaces it after any running launcher closes.
- Installer process detection uses executable image paths. A same-version payload replacement blocks on processes executing any required binary from the Active Payload. A launcher-format replacement additionally blocks on the root launcher. Detection is advisory UX; filesystem mutation results remain the correctness authority.
- Interactive blocking presents Retry and Cancel and never force-terminates a process. Silent installation returns failure without changing active state when closure is required.
- Fresh installation stages and validates the payload before tentatively publishing the stable launcher, Active Payload metadata, PATH entry, Start Menu shortcut, File Manager Integration, and uninstall metadata. It verifies the complete public state before commit. Failure removes all newly published routes and installer-owned state.
- Different-version update stages the new payload side-by-side, validates it, switches direct routes and Active Payload metadata, verifies all public state, then commits. Existing processes may continue running from the former payload while new invocations use only the new Active Payload.
- After a committed different-version update, deletion of the inactive former payload is a post-commit cleanup action. An unlocked payload is deleted immediately. A locked payload is retained as inactive cleanup material, the active route remains committed, and the installer reports a successful update with deferred cleanup rather than rolling back.
- Deferred cleanup is retried on a later install and on uninstall. Retained inactive payloads are never candidates for launcher selection, Repair, or automatic runtime fallback.
- Same-version reinstall first stages and validates a complete replacement, then requires Active Payload processes to close. The existing `versions\X.Y.Z` directory is renamed into recovery storage before staging is moved to the final path. Routes and metadata are then verified against the replacement. Any pre-commit failure reverses the ordered mutations and restores the previously observed state.
- The same-version flow must not be described or implemented as a single guaranteed atomic directory swap on Windows. Each rename, publication, verification, rollback, and cleanup result is checked explicitly.
- Pre-commit verification is deterministic and side-effect-free: required regular files exist; all payload binaries report the requested product version and architecture; the stable launcher successfully invokes CLI help; Active Payload metadata is exact; the Classic Verb targets the Active Payload helper; the Start Menu shortcut targets the Active Payload Desktop; the stable PATH entry is present according to ownership rules; and obsolete active routes are absent. Verification does not open Desktop or perform an upload.
- Automatic rollback ends at installer commit. Upit does not add first-launch pending state, post-launch health rollback, a resident update service, or a runtime version selector.
- A pre-commit failure restores the previous launcher, Active Payload path, Classic Verb, shortcut, uninstall metadata, and PATH state when each was changed by the transaction. A rollback that cannot be proven retains all required recovery material, returns failure, and does not claim a single clean active state.
- User PATH mutation operates on the current-user PATH only. Windows path comparison is case-insensitive and normalizes slash direction and trailing separators for equivalence while preserving unrelated entry text and order.
- If no equivalent stable-root entry exists, the installer inserts exactly one canonical entry and persists separate product metadata stating that it owns one such entry. It broadcasts the standard Windows environment-change notification after a successful PATH mutation.
- If an equivalent stable-root entry existed before installation, the installer records that it does not own that entry and preserves it on uninstall. Repeated install and update operations do not create duplicates.
- If ownership metadata says Upit inserted the stable-root entry, uninstall removes one normalized-equivalent occurrence and leaves every unrelated or additional user-created entry untouched. If the owned entry no longer exists, uninstall does not remove a substitute path.
- Existing payload-specific PATH entries are user-owned. The one-time legacy cutover does not remove, rewrite, or claim ownership of a manually added `versions\<random>.tmp` entry. Installation guidance warns that the obsolete entry may shadow the stable root until the user removes it manually.
- Existing random-layout installation is accepted only through the exact recorded prior `PayloadPath`, after canonical validation that it belongs to the requested Upit installation root and contains the required payload. The installer does not discover legacy state by enumerating arbitrary temporary directories.
- Legacy cutover stages the new `versions\X.Y.Z` payload and publishes the stable launcher, PATH entry, direct Desktop/helper routes, and new Active Payload metadata within the same transaction. After commit, random `.tmp` payload layout is obsolete and receives no runtime compatibility fallback.
- File Manager Integration compensation semantics from ADR 0021 remain authoritative for pre-commit registration failure and unknown registration state. This specification extends the transaction to launcher, version-directory, PATH, shortcut, and legacy-layout state without weakening those guarantees.
- Uninstall continues to remove and verify File Manager Integration before deleting callable payload. It removes the stable launcher and installer-owned PATH entry, retries inactive/staging/recovery cleanup, and preserves user-owned PATH entries. Registration cleanup failure remains fail-closed and retains callable payload.
- Build and package staging remain private and separate from consumer installation staging. The standalone CLI build output remains a developer/release artifact; Windows consumer installation now also exposes the packaged CLI through the stable launcher.
- Optional Authenticode signing includes the stable launcher, real payload executables, and setup installer under the existing protected-release rules. Unsigned ordinary installation remains fully functional.
- Documentation must replace the current Windows claims that the packaged CLI is private, that installation never modifies PATH, and that a random temporary payload directory is the installed version identity. Documentation must retain the distinction between the public CLI and the private File Manager helper.
- Current notification behavior, Toast authority, upload semantics, Configuration Set schema, and File Manager Integration menu placement are not changed by this specification.

## Testing Decisions

- The single primary acceptance seam is the existing native Windows 11 x64 installer smoke. It must install real setup artifacts and observe actual filesystem layout, current-user registry, User PATH, environment broadcasts as visible to new processes, shortcuts, process locks, command execution, update behavior, rollback, migration, and uninstall. Source inspection, mocked installers, and package-shape checks alone are insufficient.
- Good tests assert consumer-visible state transitions and invariants: which command a new terminal resolves, which payload serves new invocations, whether the old callable state survives failure, whether PATH ownership is preserved, and whether uninstall leaves callable or stale routes. Tests must not pin temporary directory tokens, mutation order that is not contractual, source text, helper forwarding, or mock echoes.
- Fresh-install smoke starts from a clean current-user state, installs the real unsigned setup, proves the final Active Payload is exactly `versions\X.Y.Z`, proves the three required real executables report one product version, and proves no active payload path ends in `.tmp`.
- Fresh-install smoke starts a new process with the post-install user environment and invokes `upit --help` by command name rather than absolute path. It also performs the existing local upload scenario through `upit` to prove arguments, Configuration Set use, stdout, stderr isolation, and exit status through the launcher.
- Launcher smoke exercises a path containing spaces and Unicode characters and a controlled nonzero CLI outcome. It proves that the launcher preserves the working directory, output channels, arguments, exit status, and cancellation behavior rather than merely proving that a child process starts.
- Active Payload corruption smoke changes `PayloadPath` to an out-of-root, missing, staging, recovery, or incoherent payload and proves the stable launcher fails closed without scanning another version. The previous valid metadata is restored by test cleanup.
- PATH smoke covers four externally different states: no stable-root entry, an equivalent user-owned entry already present, an installer-owned entry across repeated updates, and a user-edited or missing owned entry at uninstall. It proves idempotent addition, no duplicate equivalent paths, environment-change visibility in a new process, removal only when owned, and preservation of unrelated order and entries.
- Legacy PATH smoke inserts the exact prior random payload path as a user-owned entry, performs the cutover, and proves the installer leaves that entry unchanged while adding and owning the stable root. The smoke records that command resolution can remain shadowed until the fixture removes the obsolete entry manually.
- Legacy-layout smoke installs a real prior random-layout setup or fixture, confirms its recorded `PayloadPath`, then installs the new setup. It proves transactional cutover to `versions\X.Y.Z`, stable launcher publication, direct Desktop/helper routes, preservation of user-owned legacy PATH, and removal or deferred cleanup of the old random payload. Fabricating only a registry string is not sufficient acceptance evidence.
- Different-version smoke uses two real setup artifacts with different `X.Y.Z` versions. It keeps a process executing from the first Active Payload, installs the second version, proves the process remains alive on its original image, proves new `upit` and File Manager routes use the second Active Payload, and proves the locked first payload is retained only as inactive deferred-cleanup material.
- Deferred-cleanup smoke closes the old process and runs a later install or uninstall, proving the inactive payload is then removed without changing the Active Payload selected for new invocations.
- Same-version smoke first keeps a payload process active and runs the setup silently. It must fail without changing the launcher, Active Payload, direct routes, version directory, PATH, or uninstall metadata. After the process closes, the same setup succeeds while retaining the same final `versions\X.Y.Z` path.
- Same-version failure injection locks or denies a transaction-critical resource after staging and proves the prior payload and public routes are restored. If rollback itself is deliberately made impossible, the smoke proves failure is explicit and recovery directories remain present rather than being reported as clean success.
- Launcher-format update smoke uses a real prior launcher-format fixture. A running old launcher causes silent update to abort without route changes; after it exits, the new launcher is installed and the stable command remains behaviorally equivalent.
- Pre-commit verification smoke injects representative failures in launcher publication, PATH mutation, Classic Verb registration, Start Menu publication, metadata write, and payload replacement. Each scenario asserts the same external rollback invariant rather than testing private callbacks or individual copy operations.
- Post-commit cleanup-lock smoke is distinct from pre-commit failure smoke. It proves a verified new route remains active, the installer reports deferred cleanup rather than rollback, the inactive payload is not selected by the launcher, and later cleanup removes it.
- Uninstall smoke proves successful removal of File Manager Integration, active and inactive owned payload state, stable launcher, uninstall metadata, shortcut, and installer-owned PATH entry. A separate pre-existing equivalent stable-root entry is preserved.
- Existing Windows File Manager Integration registry tests remain regression coverage for `Registered`, `Needs Repair`, `Reinstall Upit`, exact Classic Verb matching, active-payload authority, Repair, and compensation. They are not expanded into a second launcher or installer acceptance seam.
- Existing package and artifact checks continue to prove public `X.Y.Z` metadata, Windows x64 architecture, signing status, checksums, and required installer contents. They support but do not replace native installer smoke.
- Native-smoke evidence must distinguish clean committed update, committed update with deferred cleanup, pre-commit failure with successful rollback, unresolved rollback with retained recovery material, and silent process-block abort. A boolean “installer exited” result is not sufficient.
- Documentation assertions are not tests. README or specification text must not be searched as proof of runtime behavior.

## Out of Scope

- Changing macOS or Linux installation, PATH, CLI packaging, or application-bundle behavior.
- Adding a stable launcher for Upit Desktop or the private File Manager helper.
- Routing File Manager Upload through the public CLI.
- Replacing the Active Payload pointer with a symlink, junction, `current` directory, version scan, or root JSON pointer.
- Adding a deployment-instance token below or beside the final `versions\X.Y.Z` path.
- Automatic first-launch health checks or rollback after installer commit.
- A resident updater, background daemon, scheduled cleanup task, or application-start cleanup worker.
- Force-terminating running Upit processes during install, update, Repair, or uninstall.
- Automatically removing or rewriting user-owned payload-specific PATH entries.
- Machine-wide installation or machine-wide PATH modification.
- Shell completion, command aliases, PowerShell profiles, Windows App Execution Aliases, or package-manager integration.
- A user-facing manual version rollback control or retaining one previous version intentionally after clean post-commit cleanup.
- Changes to CLI commands, output formats, upload protocol, Configuration Set schema, Manual Upload, File Manager Upload, clipboard behavior, or timeout behavior.
- Changes to Windows File Explorer menu placement, Classic Verb label, multi-selection behavior, or private helper lifecycle.
- Resolving the separate Windows Toast documentation/authority conflict; notification behavior is unchanged by this specification.
- Maintaining runtime compatibility with random `.tmp` payload paths after the one-time committed cutover.

## Further Notes

- ADR 0023 is the authoritative installation-layout decision for this specification. ADR 0019 remains authoritative for one product version, and ADR 0021 remains authoritative for transactional File Manager Integration registration and compensation.
- This specification intentionally supersedes the current Windows statements that the packaged CLI is private, installation does not modify PATH, and same-version update must activate a newly generated random payload directory.
- Follow-up documentation must update the project README, Windows packaging guide, unified build/package specification, current Windows Classic Verb specification, and native-smoke contract to distinguish the public Windows CLI launcher from the private File Manager helper.
- The current native smoke assertion that every same-version update changes `PayloadPath` is obsolete under this design. Same-version replacement retains the final `versions\X.Y.Z` path and proves replacement through version-aligned contents, recovery behavior, and transaction evidence instead.
- The current native smoke assertion that the previous payload must always be absent immediately after a successful update is narrowed: immediate absence is required only when cleanup is not locked. A committed update with a locked inactive payload is a distinct successful-with-deferred-cleanup outcome and must be evidenced separately.
- Existing user-owned random payload PATH entries can resolve before the installer-added stable root. Release notes and installation guidance must tell affected users to remove those entries manually.

## Delivery Status

- **Implementation:** Hoàn thành toàn bộ mã nguồn cho stable CLI launcher, transactional version layout, deferred cleanup, same-version swap, legacy cutover và fail-closed uninstall.
- **Automated Native Smoke:** Đã kiểm chứng 22/22 kịch bản tự động cốt lõi trên Windows 11 x64 (fresh install, command forwarding, out-of-root failure, cancellation, update side-by-side, deferred cleanup exit code 10, same-version process abort, swap retaining path, format upgrade, legacy cutover, ACL fail-closed uninstall, clean uninstall).
- **Known Acceptance Gaps:** Chưa đưa vào smoke tự động các kịch bản tiêm lỗi chi tiết từng bước mutation nhỏ (lỗi ghi registry Classic Verb riêng lẻ, lỗi ghi shortcut riêng lẻ) và kịch bản cố tình phá hỏng rollback để kiểm tra việc lưu giữ recovery directory.
