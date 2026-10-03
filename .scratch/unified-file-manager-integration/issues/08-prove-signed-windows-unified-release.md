# 08: Prove the signed Windows unified release

**What to build:** Execute and retain production evidence for the signed Windows 11 x64 Upit package. The protected run must prove publisher trust, the primary Explorer action, File Manager Upload behavior, Desktop status and Repair, atomic update, CLI and Manual Upload continuity, and complete uninstall cleanup.

**Blocked by:** 07: Unify Upit product version and release metadata.

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Use the protected Windows release workflow and a real Windows 11 x64 host. Unit, package-shape, direct-helper, and unsigned checks are prerequisites but not substitutes.

**Demo path:** Install the signed package on a clean target, exercise `Upload with Upit` against a local endpoint, damage and Repair registration, update the product, verify CLI and Manual Upload, then uninstall and confirm the Explorer action and package identity are gone.

- [ ] A protected SemVer release signs and timestamps the installer/package, Desktop, Explorer adapter, private worker, and retained repair material with the intended publisher identity.
- [ ] Windows establishes publisher trust and installs the consumer package without manual PowerShell or untrusted-certificate workarounds.
- [ ] `Upload with Upit` appears in the Windows 11 primary File Explorer context menu for exactly one regular file and is unavailable for unsupported selections.
- [ ] A real File Manager Upload proves default Uploader, optional Shortener, always-copy behavior, privacy-safe progress, cancellation, warning, Retry, and configuration recovery without opening Desktop ordinarily.
- [ ] Clipboard failure preserves upload success and `Copy Final URL` copies the existing result without a second upload request.
- [ ] Desktop reports truthful registration state, repairs deliberately damaged registration without elevation, and fails closed when trusted repair inputs are made unavailable.
- [ ] Updating the signed product preserves one versioned identity and does not duplicate package or Explorer registration.
- [ ] CLI and Manual Upload remain operational against the same Configuration Set.
- [ ] Uninstall removes package identity, COM/Explorer registration, payload, and the visible action.
- [ ] The workflow retains signatures, checksums, version metadata, smoke evidence, and cleanup evidence; no production-ready claim is made without all results.
