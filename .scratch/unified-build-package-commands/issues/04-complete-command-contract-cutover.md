# 04: Complete the unified command-contract cutover

**Parent specification:** `../spec.md`

**What to build:** Finish the clean cutover after both native package paths work: reject unsupported Desktop packaging, remove historical build entry points, migrate remaining callsites and general documentation, and verify the final command surface as one coherent contract.

**Blocked by:** 02: Package the complete macOS product through `make package`; 03: Package the complete Windows product through `make package`.

**Status:** ready-for-human

## Acceptance criteria

- [x] `make build` is the sole public binary-build command and `make package` is the sole public Desktop-package command.
- [ ] On Linux and other unsupported hosts, `make package` exits nonzero with a clear Desktop-packaging-not-supported diagnostic and does not fall back to a CLI build.
- [x] Obsolete granular Desktop/helper/frontend build targets, generic cross-platform CLI targets, platform build aggregators, platform-specific public package wrappers, and redundant standalone validator wrappers are removed.
- [x] Removed targets have no aliases, compatibility wrappers, deprecation paths, or stale repository callsites.
- [x] CLI run, tests, race tests, vet, formatting, explicit bindings generation, cleanup, and help remain available.
- [x] General contributor documentation describes only the unified `make build`, `make package`, and `make clean` contract and distinguishes machine prerequisites from repository dependency setup.
- [ ] macOS and Windows package callsites continue to pass their platform-specific signing and protected-release inputs through the unified command.
- [ ] The final command surface preserves the existing DMG and Windows setup consumer formats, checksum/metadata outputs, and native-smoke release gates.
- [x] A repository-wide reference check finds no active use of removed targets or old package invocation sequences.
- [ ] Final smoke evidence covers clean CLI build output, macOS package shape, Windows package shape, unsupported Linux packaging, and idempotent cleanup without source-text Makefile assertions.

## Testing seam

Run the approved CLI-build and cleanup smoke. Run native `make package VERSION=0.0.0` package checks on macOS and Windows. On Linux, invoke `make package` and require a clear nonzero rejection. Search active repository callsites for removed command names only as a migration check; consumer behavior remains proven by command execution and artifact inspection.

## Demo path

1. Show the final help output with the unified build/package contract and preserved developer utilities.
2. Demonstrate `make build` and `make clean` on a native host.
3. Present successful macOS and Windows package evidence from tickets 02 and 03.
4. Demonstrate Linux's explicit unsupported-package error.
5. Show that repository workflows and documentation contain no active legacy command callsites.

## Comments

- 2026-10-04: Implementation complete. Active Make/workflow/platform/general/frontend documentation migrated without aliases; historical issue evidence and the spec's Problem Statement deliberately retain historical command names. Final public help and CLI runtime exercised; macOS DMG artifact/failure smoke, focused CLI upload test, vet, and full race suite passed.
- 2026-10-04: `make package DETECTED_OS=Linux` exercised explicit nonzero unsupported dispatch from macOS, not native Linux. Docker CLI exists but its daemon is unavailable; no SSH hosts are configured. `.github/workflows/build-commands.yml` provides native Ubuntu CLI/unsupported-package/cleanup verification. Windows shape and native Linux evidence are pending those hosted jobs; protected signing/native interaction remain separate release gates.
