# 02: Package the complete macOS product through `make package`

**Parent specification:** `../spec.md`

**What to build:** On supported macOS Apple Silicon hosts, make one `make package` command build and validate the complete DMG from a clean checkout while keeping every private executable in package-owned staging rather than `bin`.

**Blocked by:** 01: Make CLI build and cleanup authoritative.

**Status:** ready-for-human

## Acceptance criteria

- [x] On macOS 14+ Apple Silicon, `make package` defaults to product version `0.0.0`; `VERSION=X.Y.Z` overrides it with validated numeric SemVer.
- [x] The command installs frontend dependencies from the lockfile, builds current frontend assets, and builds the CLI, Desktop executable, File Manager helper, and Finder adapter required by the package.
- [x] The packaging handoff passes explicit staged CLI, Desktop, and helper locations; the package script no longer discovers private executables in `bin`.
- [x] `bin` contains only the standalone CLI. Desktop, File Manager helper, Finder adapter, and packaged CLI copies remain private staging inputs.
- [ ] The DMG contains one `Upit.app`; its private CLI is installed at `Upit.app/Contents/Helpers/upit`, signed as nested code when signing is enabled, and is not added to `PATH` or exposed through a shortcut.
- [x] The bundle validator proves the private CLI, Desktop, Finder service, nested File Manager helper, shared version metadata, Apple Silicon architecture, and macOS 14 minimum target.
- [ ] The native-smoke path invokes the packaged CLI copy instead of a separate CLI from `bin`.
- [ ] Local packaging remains unsigned; protected CI uses the same command with the existing signing, notarization, protected-tag, and native-smoke gates.
- [x] The final DMG, checksum, and metadata use the existing macOS consumer naming convention.
- [x] Temporary package staging is removed on success and deterministic failure paths.
- [x] macOS CI and macOS packaging documentation use the unified command.
- [x] No standalone CLI GitHub Release publication is added.

## Testing seam

Run `make package VERSION=0.0.0` on a supported macOS host. Recompute the DMG checksum, inspect its metadata, mount or extract the DMG, and run the existing bundle validator against `Upit.app`. Verify the private CLI location and execute that packaged copy during native smoke. Confirm `bin` contains no private package component and no temporary staging remains.

## Demo path

1. Start from a clean checkout with the documented macOS toolchain installed.
2. Run `make package VERSION=0.0.0`.
3. Show the DMG and sidecars under the macOS distribution output.
4. Mount the DMG, validate `Upit.app`, and show the private CLI under `Contents/Helpers`.
5. Show that `bin` contains only the standalone CLI and package staging has been removed.

## Comments

- 2026-10-04: Implementation complete. Native macOS arm64 artifact smoke passed for versions `0.0.0` and `0.7.0`: locked npm install, Vue typecheck/Vite build, staged native compilation, DMG checksum/metadata, mounted bundle validation, and real `Contents/Helpers/upit --help`. Invalid Go flags caused a build failure and left no per-run staging; a subsequent package succeeded. Invalid four-component and `v`-prefixed product versions were rejected before building.
- 2026-10-04: Private CLI is explicitly signed before the outer app; native smoke now uses the installed CLI payload, not `bin`. Signed/notarized execution and interactive Finder installation/upload remain unexecuted without protected credentials/runner. Remaining checkboxes require that gate; unsigned shape is not production proof.
