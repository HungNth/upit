# Use unified Product Version source and runtime exposure

## Context

Upit builds multiple native artifacts: the standalone CLI (`upit`), Desktop app (`upit-desktop`), Windows launcher (`upit-launcher`), installer (`upit-install`), and platform integration helpers (`upit-file-manager`, macOS Finder service adapter).

Previously:
- Version metadata lacked an authoritative in-tree source of truth; build scripts (`Makefile`, `packaging/windows/build.ps1`, `packaging/macos/build.sh`) defaulted independently to values like `0.0.0` or required manual parameter overrides.
- No repository Git tags exist, so tag-derived discovery alone cannot serve as a local source of truth.
- Terminology overloaded the word "version" between JSON configuration schema formats (`version: 2`), Windows technical launcher formats (`launcherFormat: "1"`), and the overall release version.
- The CLI lacked a public command or flag to inspect the installed version, and the Desktop UI did not display the running version.

ADR 0019 established releasing one versioned Upit product across all artifacts, but did not specify the physical storage, build injection mechanics, or runtime contracts for exposing this version.

## Decision

1. **Authoritative Source of Truth**:
   - The root repository file `VERSION` defines the authoritative Product Version as a plain-text SemVer string (`X.Y.Z`).
   - The initial version recorded in `VERSION` is set to `0.9.0`.
   - All build and packaging scripts (`Makefile`, `packaging/windows/build.ps1`, `packaging/macos/build.sh`) read `VERSION` as the default source.

2. **Metadata Injection & Runtime Fallback**:
   - A Go package `internal/version` holds runtime metadata:
     - `Version`: SemVer string (e.g. `0.9.0`), injected via `-ldflags` or falling back to the package default.
     - `Commit`: Short 7-character Git commit hash (appended with `-dirty` if uncommitted modifications exist; `"unknown"` if VCS information is unavailable).
     - `Date`: Build date in UTC (formatted as `YYYY-MM-DD` in human text; ISO 8601 / RFC 3339 in JSON, or `"unknown"` if unavailable).
     - `GoVersion`, `OS`, and `Arch`: Extracted from runtime properties.
   - Values are injected during build via `-ldflags` and complemented by `runtime/debug.ReadBuildInfo()` when built via standard `go build`.

3. **CLI Contract**:
   - The CLI recognizes top-level version inspection commands:
     - `upit version`
     - `upit --version`
     - `upit -v`
   - An optional `--json` flag formats output as JSON:
     ```json
     {
       "version": "0.9.0",
       "commit": "e62acf3",
       "date": "2026-10-06T10:00:00Z",
       "goVersion": "go1.23.0",
       "os": "windows",
       "arch": "amd64"
     }
     ```
   - Text output format:
     `upit 0.9.0 (commit: <sha>, built: <date>)`
   - Extraneous arguments passed to version commands or flags are rejected with an error message and exit code 2.

4. **Desktop UI Contract**:
   - The Desktop application displays the static Product Version string (`Upit v0.9.0`) at the bottom of the navigation sidebar below all navigation tabs.

## Consequences

- All platform binaries, installers, and packages reference a single, clear source of truth.
- Disambiguates Product Version from configuration document versions and launcher internal format identifiers in domain terminology.
- Build scripts and developers have a predictable, zero-dependency mechanism to read or bump versions.
