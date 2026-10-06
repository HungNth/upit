# 04: Integrate Product Version into cross-platform build and packaging targets

**What to build:** Synchronize build and packaging pipelines across platforms by having build targets automatically default to reading the repository root Product Version file, injecting the version and build metadata into compiled binaries, Windows PE resources, and macOS application bundles.

**Blocked by:** 01: Establish authoritative Product Version source and runtime provider

**Status:** resolved

- [x] Top-level build target reads the default Product Version from the root version file when not explicitly overridden.
- [x] Windows packaging pipeline reads the default version from the root version file and embeds it into PE version resources and compiled binaries.
- [x] macOS packaging pipeline reads the default version from the root version file and embeds it into bundle metadata and compiled binaries.
- [x] Packaging smoke verification confirms that built artifacts across platforms match the authoritative Product Version.

