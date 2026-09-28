# 05: Interrupt shortening and copy the resolved Final URL

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Complete the post-processing lifecycle: explicit interruption during shortening must stop promptly and remain an interruption, while clipboard processing must run afterward with the same Final URL that stdout reports on success or fallback.

**Blocked by:** 04: Fall back safely when runtime shortening fails

Status: resolved

- [x] Ctrl-C or parent-context cancellation during an in-flight Shortener request closes response/request resources promptly and stops the request.
- [x] Interruption during shortening returns stage `shortener`, exit code 130, and empty stdout in plain and JSON modes.
- [x] Interruption is not converted into fallback success, while a timeout deadline remains the fallback behavior defined by ticket 04.
- [x] One timeout context continues to cover upload, shortening, and clipboard work.
- [x] Clipboard copying runs after Shortener resolution and receives the shortened Final URL on success.
- [x] After Shortener fallback, clipboard copying receives the Original URL now used as Final URL.
- [x] Clipboard failure remains nonfatal, preserves normal success output, and appends its existing warning without hiding a preceding Shortener warning.
- [x] Deterministic adapters and local HTTP endpoints prove cancellation, resource release, clipboard ordering, and warning order without requiring an actual desktop clipboard.
- [x] A bounded demo/test proves the command exits promptly after Shortener cancellation and leaves no resident Upit process.
