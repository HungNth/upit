# 03: Package the unsigned Classic Verb product transactionally

**What to build:** Deliver one Windows 11 x64 NSIS installer whose unsigned output installs a fully functional Classic Verb product. Remove the packaged Explorer command and MSIX machinery, retain optional protected Authenticode signing, and preserve versioned staging, exact compensation, atomic update visibility, and fail-closed uninstall cleanup.

**Blocked by:** 01: Replace Windows File Manager Integration with a per-user Classic Verb; 02: Make Windows clean success clipboard-only and silent.

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Run the public package command on native Windows and inspect the actual consumer installer, metadata, checksum, staging cleanup, installed payload, product metadata, registry state, update transitions, compensation, and uninstall result. Do not accept source inspection or cross-compilation as package proof.

**Demo path:** Build an unsigned product version, install it without elevation or certificate trust, invoke `Upload with Upit`, update from a prior payload, exercise registration and legacy-cleanup failure paths, uninstall, and show that no usable stale action remains.

- [x] One consumer setup executable contains Upit Desktop, the standalone CLI, and the private one-shot helper under one public product version.
- [x] The consumer payload contains no Explorer COM DLL, sparse MSIX, Appx manifest, retained MSIX repair material, or separate helper application entry.
- [x] The Windows package command no longer builds or requires the Explorer C++ adapter, CMake, a C++ compiler, MakeAppx, package publisher identity, or MSIX signing.
- [x] Ordinary unsigned package output installs and registers File Manager Integration successfully on Windows 11 x64 without elevation.
- [x] Protected SemVer releases may still Authenticode-sign payload executables and the setup executable when certificate inputs are supplied; ordinary builds receive no signing secrets and runtime behavior does not depend on signature status.
- [x] Public artifact naming uses product version `X.Y.Z`, metadata reports Windows x64 and the signing state truthfully, and SHA-256 sidecars are generated from the final setup executable.
- [x] Installation stages the complete new payload before creating the Classic Verb and does not overwrite the active payload in place.
- [x] The Classic Verb is written for the new helper and inspected as `Registered` before legacy package cleanup begins.
- [x] A classic-and-packaged overlap is allowed only inside the uncommitted transaction; successful completion leaves only the Classic Verb.
- [x] If Classic Verb creation or inspection fails, any partial Classic Verb is removed and the previous route, active metadata, and payload remain unchanged.
- [x] If legacy removal fails and inspection still finds the old package, the installer removes the Classic Verb, restores previous product metadata, retains both recovery payloads, and aborts.
- [x] If the old package is observed absent, the cutover commits even when the removal command reported an error; if package state cannot be inspected, the installer retains both payloads, reports failure, and does not claim completion.
- [x] Active product metadata switches only within the transaction, and the previous payload is deleted only after the Classic Verb is registered and the obsolete package is observed absent.
- [x] Uninstall removes the Classic Verb and any remaining obsolete package registration before deleting shortcuts, metadata, or payload.
- [x] Uninstall cleanup failure aborts and retains callable payload instead of leaving a menu command that targets deleted files.
- [x] Installer and package verification cover first install, same-version replacement policy, update, compensation, interrupted cleanup, repeated cleanup, staging removal, and exact final product contents.
- [x] Standalone CLI and Desktop builds retain their existing public output boundaries; temporary package payloads do not leak into `bin` or repository-root tracked files.

## Comments
