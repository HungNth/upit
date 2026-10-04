# 03: Package the complete Windows product through `make package`

**Parent specification:** `../spec.md`

**What to build:** On supported Windows 11 x64 hosts, extend the unified package contract to create the complete consumer setup executable through the existing PowerShell, CMake, Windows SDK, and NSIS flow while keeping private payloads out of `bin`.

**Blocked by:** 01: Make CLI build and cleanup authoritative.

**Status:** ready-for-human

## Acceptance criteria

- [ ] On Windows 11 x64, `make package` works under GNU Make's `cmd.exe` shell and invokes the PowerShell/CMake/NSIS packaging flow correctly.
- [ ] The command defaults to product version `0.0.0`; `VERSION=X.Y.Z` is validated and converted internally to technical Windows version `X.Y.Z.0` where required.
- [ ] The public product version and consumer artifact naming remain `X.Y.Z`; the four-component value remains a Windows package-identity detail.
- [ ] The command installs frontend dependencies from the lockfile and builds current frontend assets, CLI, Desktop executable, File Manager helper, and Explorer adapter.
- [ ] The packaging handoff passes explicit staged input and adapter-output locations; the package script no longer discovers private executables in `bin` or writes the generated Explorer DLL there.
- [ ] `bin` contains only `upit.exe`. Desktop, File Manager helper, Explorer adapter, packaged CLI copy, MSIX, and installer payload files remain package-private.
- [ ] The consumer output remains one setup executable with checksum and metadata sidecars; private MSIX registration/repair material is embedded for installer use rather than presented as a separate consumer download.
- [ ] The setup payload contains Desktop, private CLI, File Manager helper, Explorer adapter, and repair material with matching payload hashes.
- [ ] Local packaging remains unsigned; protected CI uses the same command with the existing publisher, certificate, protected-tag, install/uninstall, signature, package-identity, and native-smoke gates.
- [ ] Temporary CMake output, package staging, and installer payload staging are removed on success and deterministic failure paths.
- [x] Windows CI and Windows packaging documentation use the unified command.
- [x] No standalone CLI GitHub Release publication is added.

## Testing seam

Run `make package VERSION=0.0.0` on native Windows and prove the package flow receives technical version `0.0.0.0`. Recompute setup checksums, inspect metadata and package identity, inspect the staged/embedded payload contract, and verify no generated DLL or private executable remains in `bin` or temporary staging. Protected CI remains authoritative for trusted installation, Explorer integration, repair/update, CLI continuity, and uninstall.

## Demo path

1. Start from a clean checkout on Windows 11 x64 with the documented toolchain installed.
2. Run `make package VERSION=0.0.0`.
3. Show the consumer setup executable and sidecars.
4. Show that the installer contains the complete private payload and uses technical version `0.0.0.0` internally.
5. Show that `bin` contains only `upit.exe` and temporary staging has been removed.

## Comments

- 2026-10-04: Source implementation complete: native `cmd.exe` Make calls PowerShell orchestration, explicit staged binary/adapter/workspace inputs replace all private `bin` discovery, product `X.Y.Z` derives technical `X.Y.Z.0` for MSIX and NSIS fixed version resources, and registration material is embedded privately in setup. Obsolete release wrapper removed.
- 2026-10-04: Hosted unsigned CI now checks setup checksum, parsed metadata, exact public artifact set, staging/output boundaries, and a real invalid-Go-flag failure followed by cleanup. Existing signed installation/uninstallation and interactive Explorer release gates are retained.
- 2026-10-04: Native Windows evidence is unavailable on the current macOS host: no PowerShell/Windows toolchain or configured SSH host. Do not mark runtime acceptance complete until `.github/workflows/windows-file-manager.yml` runs on Windows. Cross-compilation or macOS checks are not substitutes.
