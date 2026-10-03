# 03: Release signed Windows File Manager Upload

**What to build:** Make Windows File Manager Upload production-releasable. The Windows 11 x64 desktop package must carry trusted package identity, a signed `IExplorerCommand` route, automatic installation registration, complete uninstall cleanup, protected-tag release automation, checksums, and signed native smoke evidence.

**Blocked by:** 02: Expose File Manager Upload in Windows 11 File Explorer

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Package tests verify install/uninstall and signature-visible behavior. The release pipeline verifies ordinary builds without release secrets; native signed-package smoke verifies the user contract. The application module remains the upload test seam.

**Demo path:** A protected SemVer tag produces a signed Windows 11 x64 package; install it on a clean Windows host, use the primary context-menu action against a local endpoint, validate notification recovery and CLI continuity, uninstall it, and verify the command no longer appears.

- [x] Protected SemVer tags, and only those tags, can access Windows signing/timestamp/publishing credentials; pull requests and ordinary branch builds perform unsigned verification only.
- [x] The production package includes the desktop executable, helper, COM command, package identity, version metadata, and checksums required for a verifiable Windows 11 x64 release.
- [x] Installation registers `Upload with Upit` automatically in the primary File Explorer menu; uninstall removes package identity and all command registration without stale entries.
- [ ] A signed Windows 11 x64 smoke proves install, primary-menu discovery, direct upload lifecycle, privacy-minimized feedback, cancellation, recovery actions, complete helper exit, CLI regression, and uninstall cleanup.
- [x] Release documentation distinguishes supported Windows 11 x64 integration from unsupported Windows 10, ARM64, and classic-verb routes.

## Comments

- GitHub Actions workflow (`.github/workflows/windows-file-manager.yml`) and packaging scripts (`packaging/windows/`) configured.
- Final release verification with actual signing and signed smoke proof will be executed upon triggering a release tag.
- 2026-10-03 audit: blocked by ticket 02 native evidence and a trusted signed Windows package/native runner. No release tag or commit was created. Run the protected release workflow and retain signed native smoke evidence before resolving this ticket.
