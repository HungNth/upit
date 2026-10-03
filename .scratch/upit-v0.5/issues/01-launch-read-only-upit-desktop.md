# 01: Launch the read-only Upit Desktop

**What to build:** Add a separately launched `upit-desktop` application that opens one read-only desktop window over an existing valid Configuration Set. It gives users a usable four-area shell while keeping `upit` headless and unchanged. The completed slice demonstrates the selected Wails/Vue stack, system theme, single-instance focus, deterministic current-Configuration Set display, and a clear invalid/missing-state handoff without writing configuration.

**Blocked by:** None (can start immediately)

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Use the Wails-free application module as the primary seam for loading, strict validation, snapshot creation, and startup classification with temporary homes. Preserve existing CLI runner tests. Test the Wails adapter only for framework-specific single-instance/focus behavior and binding input validation; use a running desktop smoke for the visible shell.

**Demo path:** Launch `upit-desktop` against a valid temporary Configuration Set, inspect the four sidebar areas and selected defaults, launch it a second time and observe the first window restored/focused, then run `upit upload` from the same Configuration Set to prove the CLI remains independent.

- [x] The repository uses Go 1.25 in one module and pins Wails `v3.0.0-beta.26`; `upit` keeps no Wails/WebView startup dependency.
- [x] The desktop executable opens a single English-language Vue/TypeScript/Vite/shadcn-vue window with Manual Upload, Global Configuration, Uploaders, and Shorteners navigation and system light/dark theme behavior.
- [x] The application module remains Wails-free and classifies an existing Configuration Set as valid or requiring Setup/Repair without mutating bytes.
- [x] Valid startup reads the current Global Configuration and named definitions through the production strict validator and presents deterministic list/default state.
- [x] A second desktop launch restores and focuses the existing window rather than creating a competing instance.
- [x] The desktop has no system tray or resident background behavior; closing a clean idle window exits the process.
- [x] Existing CLI builds, tests, output, exit codes, cancellation, and Configuration Management commands remain unchanged.
- [x] A native runnable desktop smoke verifies window startup, sidebar navigation, system light/dark theme response where observable, second-launch focus, and clean process exit on an available host platform.

**Earlier verification note:** Native Windows smoke covered startup, primary window health, second-instance exit/focus, assets, and clean exit; sidebar/theme interaction remained unverified in that earlier run. Supplemental Vite evidence was not native proof.

## Comments

- 2026-10-03: actual rebuilt macOS Wails WebView verified window startup and all sidebar areas, native assets, Setup/normal/Repair states, and clean process exit 0. Second launch exited 0 and the original native window was observed visible and focused, with `document.hasFocus()` true.
- Native appearance smoke changed macOS appearance to light and dark, observing the real WebView media query and body background `rgb(250, 250, 250)` then `rgb(9, 9, 11)`. The original system appearance was restored in a finally block. Native focused-window capture provided visual proof; no Vite browser was substituted.
