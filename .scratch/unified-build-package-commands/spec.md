# Upit — Unified CLI Build and Desktop Package Commands

Status: ready-for-human

## Problem Statement

Upit's Makefile exposes overlapping build targets with inconsistent meanings. The default build creates only the CLI, Desktop and File Manager helper binaries are produced through separate targets, generic cross-platform targets create only CLI binaries despite broad names such as `build-all`, macOS packaging has a Make target, and Windows packaging is invoked directly through platform scripts. The resulting command surface does not communicate which artifacts are public deliverables and which executables are private package inputs.

The generated files are similarly ambiguous. Desktop and File Manager helper executables are left beside the standalone CLI under `bin`, even though users should receive one complete Desktop installer rather than download those private components independently. Platform-specific build commands, package commands, version inputs, staging behavior, and cleanup behavior also differ.

Maintainers need one small, predictable command contract: build a standalone native CLI, or package the complete native Desktop product. The contract must preserve the existing macOS and Windows package guarantees without introducing Linux Desktop packaging, cross-compilation, compatibility aliases, or new CLI release publication work.

## Solution

Expose two authoritative build commands. `make build` builds only the standalone CLI for the current host and places that executable under `bin`. `make package` creates the complete native Desktop installer on supported macOS and Windows hosts. Linux continues to support the CLI build but rejects Desktop packaging with a clear nonzero error.

Desktop packaging runs from a clean source checkout once the host's required development toolchain is installed. It installs locked frontend dependencies, builds the frontend, CLI, Desktop executable, File Manager helper, and native platform adapter, then assembles the existing platform-native consumer package. Private package inputs use temporary staging rather than `bin`, and staging is removed after both successful and failed packaging.

The macOS consumer artifact remains a DMG. Its private CLI lives at `Upit.app/Contents/Helpers/upit`, is signed as nested app code, and is the CLI copy used by package validation and native smoke. The Windows consumer artifact remains the setup executable; private registration material remains internal to package assembly. Both installers contain Desktop, the standalone CLI executable, the private File Manager helper, and the native File Manager Integration adapter. The packaged CLI is a private payload: installation does not add it to `PATH`, create a CLI shortcut, or expose a separate application entry point.

Local packaging defaults to version `0.0.0` and creates an unsigned verification package. Protected CI invokes the same `make package` contract with a product SemVer and the existing signing/notarization inputs. Windows derives its required four-component technical package version from the shared three-component product SemVer.

## User Stories

1. As a CLI developer, I want `make build` to produce the native Upit CLI, so that the default build command has one unambiguous result.
2. As a CLI developer on macOS or Linux, I want the local executable named `upit`, so that I can invoke the conventional host-native name.
3. As a CLI developer on Windows, I want the local executable named `upit.exe`, so that it follows Windows executable conventions.
4. As a maintainer, I want `bin` to contain only the standalone CLI after a build, so that public and private artifacts are not mixed.
5. As a maintainer, I want the generated CLI to be executable after `make build`, so that a successful compile is not mistaken for a usable artifact.
6. As a Desktop maintainer, I want `make package` to build the complete native Desktop product, so that packaging has one consistent entry point on every supported Desktop platform.
7. As a macOS user, I want one DMG containing the complete Upit Desktop product, so that I can download and install one artifact.
8. As a Windows user, I want one setup executable containing the complete Upit Desktop product, so that I can install it without manual package registration or scripting.
9. As a Desktop user, I want the installer to contain Upit Desktop, so that the installed product has one visible control surface.
10. As a File Manager Upload user, I want the installer to contain the required private helper, so that Finder or File Explorer integration does not require a second download.
11. As a File Manager Upload user, I want the installer to contain the native platform adapter, so that File Manager Integration is delivered with the product.
12. As a Desktop installer user, I want the CLI payload included in the installation, so that package smoke, recovery, and product continuity use the same product version.
13. As a Windows terminal user, I want the installer to expose `upit.exe` as a stable command on User PATH, while keeping the real CLI, Desktop, and File Manager helper together under `versions\X.Y.Z`.
14. As a macOS user, I want the packaged CLI to remain a private helper inside the application bundle, so that Finder Services and Desktop remain self-contained.
15. As a maintainer, I want Desktop, helper, adapter, and packaged CLI inputs built in temporary staging, so that `bin` remains the standalone CLI output boundary.
16. As a maintainer, I want package staging removed after successful packaging, so that generated private inputs do not accumulate.
17. As a maintainer, I want package staging removed after failed packaging, so that retries start from a known state.
18. As a maintainer working from a clean checkout, I want `make package` to install frontend dependencies from the lockfile, so that one command creates reproducible frontend assets.
19. As a maintainer, I want `make package` to build the embedded frontend before compiling Desktop, so that the packaged application contains current assets.
20. As a maintainer, I want bindings regeneration to remain an explicit maintenance command, so that routine packaging does not rewrite committed generated interfaces.
21. As a local package verifier, I want `make package` to work without signing credentials, so that I can validate package structure before protected release work.
22. As a release maintainer, I want protected CI to use the same `make package` entry point, so that local verification and production packaging do not drift.
23. As a release maintainer, I want signing, notarization, and protected-tag enforcement to remain controlled by existing CI inputs, so that package-command simplification does not weaken release gates.
24. As a local package verifier, I want a default product version of `0.0.0`, so that `make package` remains a one-command local operation.
25. As a release maintainer, I want to pass one `X.Y.Z` product version, so that macOS and Windows release under the same Upit version.
26. As a Windows release maintainer, I want the required `X.Y.Z.0` package identity derived internally, so that Windows technical constraints do not leak into the cross-platform command contract.
27. As a user downloading a package, I want checksum and metadata sidecars generated with the installer, so that existing verification and publication flows remain available.
28. As a Linux CLI developer, I want `make build` to remain supported, so that Linux CLI work is unaffected by the absence of Desktop packaging.
29. As a Linux maintainer, I want `make package` to fail clearly and nonzero, so that unsupported Desktop packaging is never mistaken for success.
30. As a maintainer, I want `make clean` to remove CLI output, package staging, and distribution artifacts, so that the repository returns to a pre-build state.
31. As a maintainer, I want `make clean` to be idempotent, so that cleanup is safe when some or all outputs are already absent.
32. As a contributor, I want obsolete granular and misleading cross-platform build targets removed, so that the Makefile presents one current convention rather than several historical ones.
33. As a contributor, I want CLI run, tests, race tests, vet, formatting, bindings generation, cleanup, and help commands preserved, so that unrelated development workflows remain available.
34. As a CI maintainer, I want macOS and Windows workflows migrated to the unified commands, so that repository automation exercises the same public contract documented for contributors.
35. As a contributor, I want build and packaging documentation to describe only the unified command contract, so that examples cannot direct me to removed targets.
36. As a release maintainer, I want existing native package validation and protected native smoke gates preserved, so that a Makefile refactor is not treated as proof of real installation or File Manager Integration.
37. As a project maintainer, I do not want this work to publish new standalone CLI release assets, so that release-workflow expansion remains a separate decision.

## Implementation Decisions

- `make build` is the authoritative native standalone CLI build for the current host.
- The local CLI output is `bin/upit` on macOS and Linux and `bin/upit.exe` on Windows.
- `make build` does not build Desktop, the File Manager helper, native adapters, installers, or cross-platform variants.
- `make package` is the authoritative native Desktop packaging command on macOS and Windows.
- `make package` rejects Linux and any other unsupported host with a clear nonzero error. It does not silently skip work or fall back to a CLI build.
- Cross-compilation is removed from the public Make contract. Each supported artifact is built on its native target platform.
- Desktop packaging starts from the current source checkout and runs locked frontend dependency installation, frontend compilation, CLI compilation, Desktop compilation, File Manager helper compilation, native adapter compilation, package validation, and final package assembly.
- Host development tools remain prerequisites. Make does not install Go, Node.js, npm, Xcode Command Line Tools, Windows SDK, CMake, NSIS, signing certificates, or notarization credentials.
- Generated frontend bindings remain committed inputs and are regenerated only through the explicit bindings maintenance target after exported Go services or models change.
- `bin` is reserved for the standalone CLI. Desktop, File Manager helper, native adapter, registration material, and the CLI copy used by an installer are private staging inputs.
- Private staging is temporary and is cleaned on success and failure.
- Package orchestration passes explicit staged CLI, Desktop, helper, adapter, and registration-material locations into the platform package scripts. Those scripts no longer discover private inputs in `bin` or write generated private outputs there.
- Native adapter compilation writes into package-owned temporary staging rather than `bin`.
- The macOS consumer artifact remains one DMG containing one `Upit.app` with Desktop and its private Finder integration components. The private CLI is installed at `Upit.app/Contents/Helpers/upit`, signed with the nested code, validated as part of the bundle, and used by native smoke instead of a separate CLI from `bin`.
- The Windows consumer artifact remains one setup executable containing Desktop, the private CLI payload, File Manager helper, Explorer adapter, and private registration/repair material.
- Private Windows registration material is an installer input, not a separate consumer download.
- On Windows, desktop installation publishes a stable launcher at `%LOCALAPPDATA%\Programs\Upit\upit.exe` and idempotently adds that root to User PATH, while Desktop and the private helper target the versioned Active Payload directly. On macOS, the packaged CLI remains private within the app bundle and does not modify PATH.
- The package output retains the existing platform naming convention, checksum sidecar, and structured metadata sidecar.
- `VERSION` is the single product-version input. It defaults to `0.0.0` for local packaging and accepts a numeric `X.Y.Z` SemVer for CI and releases.
- macOS uses `X.Y.Z` directly. Windows derives `X.Y.Z.0` only where its package identity or file metadata requires four numeric components; the public product version remains `X.Y.Z`.
- On native Windows, `make package` invokes the PowerShell/CMake/NSIS packaging flow correctly even though GNU Make uses `cmd.exe`; it passes the derived four-component technical version without exposing that version through the public Make contract.
- Local `make package` creates unsigned verification artifacts. Protected CI supplies existing signing, notarization, publisher, certificate, and protected-tag inputs to the same command.
- Existing production signature, notarization, install/uninstall, package-identity, and native smoke gates remain authoritative.
- `make clean` removes `bin`, temporary package staging owned by the build, and `dist`; repeated cleanup succeeds.
- The obsolete granular build targets, generic cross-platform build targets, platform build aggregators, platform-specific public package wrapper, and redundant standalone validator wrapper are removed rather than retained as compatibility aliases.
- Platform-specific package scripts remain implementation details behind `make package`.
- CLI run, test, race-test, vet, format, bindings-generation, cleanup, and help commands remain available.
- Every repository callsite and document moves to the unified command contract in the same cutover.
- Existing ADRs defining one user-visible Upit product, private File Manager helpers, and one product version remain authoritative; no new domain term or ADR is introduced.

## Testing Decisions

- Tests and smoke checks exercise commands and inspect real outputs. They do not assert Makefile source text, recipe strings, target declarations, or help wording as a substitute for behavior.
- The primary CLI-build seam starts from clean generated output, runs `make build`, executes the produced CLI with `--help`, and inspects the direct contents of `bin`.
- The CLI-build seam requires exactly the native CLI executable in `bin` and rejects Desktop, File Manager helper, adapter, or installer artifacts.
- Existing CLI behavior tests remain authoritative for upload, configuration, errors, and exit semantics. This work does not duplicate those behaviors in Makefile-specific tests.
- The primary package seam runs `make package VERSION=0.0.0` on each supported native host.
- macOS package verification reuses the existing bundle validator to prove application layout, nested Finder components, the private CLI at `Upit.app/Contents/Helpers/upit`, shared version metadata, architecture, minimum operating system, and optional signatures. Native smoke exercises that packaged CLI copy rather than a separate CLI from `bin`.
- Windows package verification invokes `make package VERSION=0.0.0` from native Windows, proves the PowerShell/CMake/NSIS flow receives technical version `0.0.0.0`, and reuses the existing manifest/input and package-builder checks to prove package identity, payload composition, Explorer registration metadata, checksums, and installer metadata. No generated adapter DLL, Desktop executable, helper executable, or private CLI copy may remain in `bin`.
- Automated package checks recompute installer checksums and validate structured metadata rather than checking only that a file exists.
- Automated package checks prove private Desktop, helper, adapter, registration material, and packaged CLI inputs do not remain in `bin` or temporary staging after success.
- Failure-path package checks prove temporary staging cleanup where the platform scripts provide a deterministic failure seam.
- Linux verification invokes `make package`, requires a nonzero exit status, and requires a clear unsupported-platform diagnostic. It does not inspect Makefile text.
- Cleanup verification creates sentinel output under `bin`, package staging, and `dist`, runs `make clean`, and requires all owned generated output to be absent.
- Cleanup verification runs `make clean` again and requires success to prove idempotence.
- Hosted unsigned package checks prove package shape only. They are not accepted as evidence of signing, notarization, installation, Desktop launch, Finder or Explorer visibility, File Manager Upload, repair, update, or uninstall behavior.
- Existing protected macOS native smoke remains authoritative for signed/notarized DMG installation, Desktop launch, Finder Service discovery, File Manager Upload, CLI continuity, repair/removal, and package cleanup.
- Existing protected Windows native smoke remains authoritative for trusted setup installation, package identity, Desktop launch, Explorer command behavior, File Manager Upload, CLI continuity, repair/update/recovery, and uninstall cleanup.
- Windows consumer and native smoke verification remains on a supported Windows 11 x64 runner. It is not replaced by checks run on macOS.

## Out of Scope

- Publishing standalone CLI binaries as new GitHub Release assets.
- Adding or changing standalone CLI checksums, release metadata, release naming, or release workflows.
- Linux Desktop packaging, AppImage, DEB, RPM, Flatpak, Snap, or Linux File Manager Integration.
- Windows ARM64, Windows 10 shell integration, Intel macOS Desktop packaging, macOS versions before 14, or additional operating systems.
- Cross-compiling CLI, Desktop, helpers, adapters, or installers from one host to another.
- Adding the packaged CLI to `PATH`, installing shell shims, creating CLI shortcuts, or changing shell profiles.
- Replacing the existing DMG or Windows setup formats.
- Changing Desktop, CLI, Manual Upload, File Manager Upload, Configuration Set, Uploader, Shortener, clipboard, notification, repair, or recovery behavior.
- Automatically regenerating frontend bindings during every package build.
- Installing host development toolchains or provisioning signing/notarization credentials.
- Introducing a version file, deriving local versions from Git state, or synchronizing unrelated frontend package metadata.
- Preserving removed Make targets through aliases, deprecation periods, or compatibility wrappers.
- Treating unsigned local packages as production-ready artifacts.

## Further Notes

- The current macOS packaging route already provides temporary staging cleanup, bundle validation, checksums, metadata, signing/notarization inputs, and protected native smoke. The unified command should reuse those capabilities rather than replace them.
- The current Windows packaging route already provides native adapter compilation, package-input validation, temporary staging cleanup, checksums, metadata, signing inputs, consumer installation checks, and protected native smoke. The unified command should reuse those capabilities while keeping private registration material out of the public artifact surface.
- The package command is self-contained with respect to repository dependencies, not machine provisioning. Missing platform toolchains remain actionable prerequisite errors.
- The confirmed design deliberately favors a small public command surface over preserving historical build conveniences.

## Implementation and verification evidence

- 2026-10-04: Source implementation for all four tickets delivered. Public commands are `make build`, `make package VERSION=X.Y.Z`, and idempotent `make clean`; package-owned per-run staging lives under `.build/package/`. CI, platform scripts, validators, native CLI handoff, and active contributor documentation were cut over without compatibility aliases.
- Native macOS arm64: CLI command/output/cleanup smoke passed; real unsigned DMG smoke passed at `0.0.0` and `0.7.0`, including Vue typecheck, bundle layout, minimum platform/architecture/version metadata, checksum, mounted private CLI execution, and staging cleanup after success and deliberate Go compiler failure. Invalid product-version formats and unsupported Linux dispatch were rejected; Linux dispatch was exercised from macOS, not on Linux.
- Focused CLI upload test, `go vet ./...`, shell syntax checks, and one final full `go test -race ./...` passed. The Go test linker emitted duplicate Objective-C-library and deployment-target warnings; no warning suppression or runtime-code change was made.
- Independent post-implementation static reviews reported zero Standards findings and zero Spec findings. This is source-review evidence, not native Windows execution or production signing proof.
- Outstanding acceptance: native Windows package/output/failure smoke, native Ubuntu CLI/unsupported-package/cleanup smoke, and protected signed/notarized installation/Finder/Explorer interaction. No configured SSH hosts, Windows toolchain, or running Docker daemon were available. The Windows, macOS, and new Linux workflows contain the required gates; tickets retain unchecked acceptance criteria and `ready-for-human` status until that evidence exists.
