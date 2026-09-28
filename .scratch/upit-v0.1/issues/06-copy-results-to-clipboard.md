# 06: Copy successful results to the clipboard optionally

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Add clipboard copying as an opt-in post-action after a successful upload. Desktop users can enable it globally or per invocation, while headless users retain a dependency-free upload path and successful uploads remain successful when clipboard access is unavailable.

**Blocked by:** 04 — Stabilize the automation output contract

**Status:** ready-for-agent

- [ ] Clipboard copying is disabled by default in version-1 global configuration.
- [ ] `--clipboard` and `--no-clipboard` override the configured value in either flag position, with CLI taking precedence over configuration.
- [ ] Clipboard copying receives the Final URL only after upload and response extraction succeed.
- [ ] The production adapter uses available operating-system clipboard commands and does not link Wails, WebView, GTK, or another GUI toolkit into the CLI.
- [ ] When clipboard access succeeds, the normal plain or JSON success output and exit code remain unchanged.
- [ ] When clipboard access is unavailable or fails, the command still exits 0, preserves the normal success output, and writes one actionable warning to stderr.
- [ ] Upload, parse, and configuration failures never attempt clipboard copying.
- [ ] A deterministic test adapter proves enabled/disabled behavior, CLI-over-config precedence, copied value, no-copy-on-failure, and nonfatal warning behavior through the CLI seam.
- [ ] A headless smoke demonstrates that absence of clipboard tooling cannot invalidate an otherwise successful upload.
