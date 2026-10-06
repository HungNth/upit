# 01: Establish authoritative Product Version source and runtime provider

**What to build:** Establish the repository's authoritative Product Version plain-text source file initialized to `0.9.0`, along with an internal Go runtime provider package that encapsulates version metadata (version string, short commit SHA with `-dirty` suffix, build date, Go version, OS, arch) supporting both compile-time `-ldflags` overrides and automatic runtime fallback via build info.

**Blocked by:** None (can start immediately)

**Status:** resolved

- [x] A root version file defines the canonical SemVer string initialized to `0.9.0`.
- [x] An internal Go runtime package exposes structured version metadata and formatted text/JSON representations.
- [x] Runtime provider reads build info when ldflags are omitted, extracting commit SHA and dirty state.
- [x] Runtime provider falls back gracefully to `"unknown"` when commit SHA or build date are missing.
- [x] Tests verify that the runtime provider correctly exposes all metadata fields and handles clean and missing build info states.

