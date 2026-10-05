# 03: Brand macOS Application Bundle

**Parent:** `.scratch/desktop-branding/spec.md`

**What to build:** Integrate native application icon branding into the macOS desktop application bundle so that Finder, Dock, and the application switcher display the official Upit icon.

**Blocked by:** 01: Establish Master Branding Assets and Reproducible Generation Pipeline

**Status:** ready-for-agent

- [ ] macOS bundle metadata configures the application bundle icon resource identifier.
- [ ] macOS packaging script places the canonical Apple Icon Image into the application bundle resources during packaging.
- [ ] macOS packaging validation asserts that the bundle icon resource exists with non-zero size and is referenced in the bundle information property list.
