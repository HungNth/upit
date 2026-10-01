# 02: Release notarized macOS File Manager Upload

**What to build:** Make the macOS File Manager Upload production-releasable. The macOS 14+ Apple Silicon desktop package must include the background NSServices helper, use Developer ID signing with Hardened Runtime, be notarized and stapled, register its Finder Service during installation, remove it during uninstall, and be produced and verified through protected-tag release automation.

**Blocked by:** 01: Expose File Manager Upload through Finder Services

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Package and release checks prove signing, notarization, stapling, registration, and cleanup. The signed native smoke proves the Finder user contract. Upload semantics remain covered by the shared File Manager Upload application seam.

**Demo path:** A protected SemVer tag produces a signed, notarized, stapled macOS package; install it on a clean macOS 14+ Apple Silicon host, enable the Service if System Settings requires it, invoke Finder’s action against a local endpoint, verify all recovery paths and no window, then uninstall and confirm the Service disappears.

- [x] Only protected SemVer tags access Developer ID, notarization, and publication credentials; ordinary CI builds unsigned verification artifacts without release secrets.
- [x] The production package signs every executable and nested helper with Hardened Runtime, completes notarization and stapling, and publishes verifiable checksums and version metadata.
- [x] Installation registers the Finder Service automatically and documents any user-controlled System Settings enablement; uninstall unregisters the Service and leaves no stale action.
- [ ] A signed macOS 14+ Apple Silicon smoke proves package trust, Finder Services or Quick Actions discovery, direct lifecycle, privacy-minimized feedback, cancellation, recovery actions, no-window/helper-exit behavior, CLI regression, and uninstall cleanup.
- [x] Release documentation identifies the Developer ID non-sandbox distribution model and excludes Mac App Store, Intel, older-macOS, and extension-based routes.

## Comments

- Packaging is implemented in `packaging/macos/package.sh`, `validate.sh`, `install.sh`, `uninstall.sh`, and `native-smoke.sh`; the bundle contains Desktop, the NSServices provider, and the nested Wails-free helper.
- `.github/workflows/macos-file-manager.yml` separates unsigned PR/branch verification from protected SemVer signing, notarization, stapling, checksum, smoke, and publish jobs.
- The signed native smoke remains pending until a protected macOS 14+ Apple Silicon runner executes it; no signing or notarization claim is made from this Windows host.
