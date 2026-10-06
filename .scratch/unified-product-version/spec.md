# Spec: Unified Product Version and Runtime Exposure

**Status:** ready-for-agent

## Problem Statement

When building and running Upit, there is no single, persistent in-repo source of truth for the release version. Release packaging defaults to placeholder versions like `0.0.0` or requires ad-hoc command parameters. Because the repository does not have Git tags, builds cannot reliably infer a release identity from Git history. Furthermore, the CLI provides no way to query the installed version (`upit version` or `upit --version` fail with upload argument errors), and the Desktop interface displays no visual indicator of the running version. Users and automated pipelines cannot quickly determine which exact product release and build artifact they are interacting with.

## Solution

Establish an authoritative plain-text version file at the repository root initialized to `0.9.0`. All packaging scripts and build targets use this file as the default Product Version. The build pipeline injects version metadata (version, short Git commit SHA with dirty state, and build timestamp) into native binaries. The CLI exposes standard top-level commands and flags (`upit version`, `upit --version`, `upit -v`, including `--json`), and the Desktop application displays the static Product Version at the base of its navigation sidebar.

## User Stories

1. As a CLI user, I want to run `upit --version` so that I can immediately check the installed Product Version.
2. As a CLI user, I want to run `upit -v` so that I have a convenient short flag to check the version.
3. As a CLI user, I want to run `upit version` as a subcommand so that I can inspect version information without remembering specific flag styles.
4. As a CLI user, I want `upit version` to output a concise, human-readable line containing the Product Version, short commit SHA, and build date so that I have complete context for debugging.
5. As an automated script or CI job, I want to run `upit version --json` so that I can programmatically consume structured metadata (version, commit, date, goVersion, os, arch).
6. As a CLI user, I want `upit --version --json` and `upit -v --json` to emit the exact same JSON schema as `upit version --json` so that flag order and syntax are forgiving.
7. As a CLI user, I want the CLI to reject invalid extra arguments passed to version commands (such as `upit version extra`) with exit code 2 and a usage error message so that unintentional syntax errors are immediately caught.
8. As a Desktop user, I want to see the running Product Version at the bottom of the navigation sidebar so that I always know which version of Upit Desktop is active.
9. As a developer building from a dirty working tree, I want the embedded commit SHA to include a `-dirty` suffix so that development builds are clearly distinguished from clean releases.
10. As a developer building from a source archive without Git metadata, I want the build and CLI to fall back gracefully to `"unknown"` for commit SHA and build date while still reporting the exact Product Version so that compilation and inspection never fail.
11. As a release engineer, I want the build targets across platforms to default to the repository root version file so that I do not need to manually synchronize version parameters across build scripts.
12. As a developer, I want domain documentation to clearly distinguish between the overall Product Version, JSON configuration schema versions, and internal launcher formats so that terminology remains consistent across the codebase.

## Implementation Decisions

- **Source of Truth**: A single file at repository root named `VERSION` contains the authoritative SemVer string (`0.9.0\n`).
- **Domain Language Alignment**: The term `Product Version` is defined in `CONTEXT.md` and governed by ADR 0024, explicitly separating release versions from configuration document schema versions and launcher technical formats.
- **Go Version Runtime Provider**: An internal Go version package encapsulates runtime version metadata:
  - Exposes properties for version, commit SHA, build date, Go runtime version, target OS, and architecture.
  - Supports build-time injection via Go `-ldflags` (`-X`).
  - Reads VCS revision and status via Go runtime build information (`runtime/debug.ReadBuildInfo`) when ldflags are not provided.
  - Falls back gracefully to the canonical default version and `"unknown"` for missing commit/date.
- **CLI Command Interception & Precedence**:
  - The CLI runner checks top-level arguments before upload flag parsing.
  - Matches `version`, `--version`, and `-v`.
  - Parses an optional `--json` flag.
  - Text mode emits: `upit <version> (commit: <sha>, built: <date>)` to stdout, exit code 0.
  - JSON mode emits a JSON object with version, commit, date, goVersion, os, and arch keys to stdout, exit code 0.
  - Any unexpected positional arguments or unknown flags yield exit code 2 and usage instructions written to stderr.
- **Desktop Sidebar Exposure**:
  - The Desktop frontend displays the static Product Version string (e.g. `Upit v0.9.0`) at the base of the navigation sidebar below existing area tabs.
- **Packaging Pipeline Integration**:
  - Build scripts across platforms read the default version from the root version file if not overridden via command arguments or environment variables.
  - Windows packaging embeds the version into the PE resource compilation and passes it to binary compilation.
  - macOS packaging embeds the version into app bundle info property lists and binary compilation.

## Testing Decisions

- **Good Test Definition**: Tests verify public external observable contracts without depending on internal implementation details or private variables.
- **CLI Testing Seam**:
  - Execute the CLI runner through the public runner interface across all supported invocations:
    - `version`, `--version`, `-v` (verifying formatted text on stdout, empty stderr, exit code 0).
    - `version --json`, `--version --json`, `-v --json` (verifying JSON structure, field values, exit code 0).
    - Invalid extra argument invocations (e.g. `version unexpected`, `--version unexpected`) verifying exit code 2, error diagnostic on stderr, empty stdout.
- **Desktop Testing Seam**:
  - Desktop integration test verifying the presence and text content of the version element within the navigation sidebar layout.
- **Packaging & Build Seams**:
  - Script smoke checks verifying that build targets correctly read the root version file when no explicit override is provided.
- **Prior Art**:
  - Existing CLI test suite and schema validation test suites.

## Out of Scope

- Dynamic in-app update checks, remote release downloads, or auto-update mechanisms.
- Interactive About dialogs, release notes popups, or expandable commit inspection modals in the Desktop UI.
- Re-architecting or migrating JSON configuration document schema versions (`version: 2`) or launcher format identifiers (`launcherFormat: "1"`).
- Automatically creating Git tags or pushing commits to remote repositories.

## Further Notes

- Fully adheres to ADR 0019 ("Release one versioned Upit product") and ADR 0024 ("Use unified Product Version source and runtime exposure").
- Initial product release version is set to `0.9.0`.
