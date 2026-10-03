# 05: Cancel and time-limit uploads safely

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Let users stop an in-flight streaming upload through Ctrl-C or an explicit deadline without leaving files, multipart pipes, request bodies, response bodies, or network work alive after the command exits.

**Blocked by:** 01 — Upload one file with the default Uploader

**Status:** ready-for-human

- [ ] The upload workflow receives and propagates a signal-aware context through file streaming and HTTP execution.
- [ ] `--timeout` accepts Go duration syntax in either supported flag position; zero or omission means no overall deadline.
- [ ] Ctrl-C cancels the request promptly and exits 130 without success output.
- [ ] An expired explicit timeout cancels the request, produces a structured runtime failure, and exits 1 rather than 130.
- [ ] Cancellation and timeout close the input file, multipart pipe endpoints, request body, and any response body without waiting for the remote endpoint to finish.
- [ ] The local test server can observe client cancellation instead of receiving a silently continued upload.
- [ ] CLI-seam tests deterministically cover no-default-timeout behavior, invalid durations, explicit timeout, interrupt handling, stdout/stderr behavior, and exit-code mapping.
- [ ] A smoke scenario starts a deliberately slow local upload, cancels it, and observes prompt process exit.

## Comments

- 2026-10-03 backlog audit: this ticket is dispositioned `wontfix`/`ready-for-human`. Its parent v0.1 document contract conflicts with approved v0.2/v0.3 cutovers and the no-compatibility principle. Historical version-1 behavior is not restored.
