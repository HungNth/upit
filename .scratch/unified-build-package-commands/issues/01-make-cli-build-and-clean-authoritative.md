# 01: Make CLI build and cleanup authoritative

**Parent specification:** `../spec.md`

**What to build:** Make the native standalone CLI the only output of the default build command, and make cleanup return all owned generated-output locations to a pre-build state. This establishes the output and staging boundary that both platform package slices will consume.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

## Acceptance criteria

- [x] `make build` compiles only the native CLI for the current host.
- [ ] The local output is `bin/upit` on macOS/Linux and `bin/upit.exe` on Windows.
- [x] Starting from clean generated output, `bin` contains exactly that CLI after the build; Desktop, File Manager helper, native adapter, and installer artifacts are absent.
- [x] The generated CLI runs successfully with `--help`.
- [x] The build contract defines a package-owned temporary staging boundary outside `bin` for later platform package slices.
- [x] `make clean` removes `bin`, package-owned staging, and `dist` without deleting source or configuration.
- [x] Running `make clean` when those outputs are already absent succeeds.
- [x] CLI run, tests, race tests, vet, formatting, bindings generation, cleanup, and help commands remain available.
- [x] No source-text test pins Makefile recipes or target wording.

## Testing seam

From clean generated output, run `make build`, execute the produced CLI with `--help`, and enumerate the direct children of `bin`. Create sentinel files under `bin`, package staging, and `dist`; run `make clean` twice and verify all owned output paths are absent after both runs.

## Demo path

1. Run `make clean`.
2. Run `make build`.
3. Run the generated CLI with `--help`.
4. Show that `bin` contains only the host-native CLI.
5. Create representative generated-output sentinels, run `make clean` twice, and show that cleanup is complete and idempotent.

## Comments

- 2026-10-04: Implementation complete. `bash packaging/command-smoke.sh` reproduced leftover `.build` under the original cleanup, then passed after the fix on native macOS arm64: executable CLI, exact `bin` contents, sentinel cleanup, and second cleanup. Staging is repo-owned `.build/package/`, not an untracked OS temporary directory.
- 2026-10-04: Final `make build` and `bin/upit --help`, focused CLI upload test, `go vet ./...`, and full `go test -race ./...` passed. Native Windows/Linux execution is pending the hosted command-smoke jobs; the remaining checkbox is not claimed verified on macOS.
