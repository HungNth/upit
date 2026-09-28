# 04: Stabilize the automation output contract

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Provide the stable plain and machine-readable command contract used by shells and CI. Successful uploads preserve Original URL and Final URL, errors stay off stdout, and command outcomes use the agreed exit codes.

**Blocked by:** 03 — Enforce endpoint and extraction failure semantics

**Status:** ready-for-agent

- [ ] A successful result contains Original URL, Final URL, and HTTP status; Original URL equals Final URL in v0.1.
- [ ] Plain success writes exactly the Final URL plus a newline to stdout and writes no status prose.
- [ ] `--json` success writes one JSON object containing `success: true`, `originalUrl`, and `finalUrl` to stdout.
- [ ] Plain failures write an actionable human-readable diagnostic to stderr and leave stdout empty.
- [ ] With `--json`, failures write one JSON object to stderr containing `success: false`, `stage`, `message`, and `statusCode` only when available; stdout remains empty.
- [ ] Success exits 0, runtime/configuration/upload failure exits 1, and invalid command usage exits 2.
- [ ] Invalid flag combinations and missing or extra file operands are usage errors rather than upload attempts.
- [ ] Output is deterministic and contains no logs, progress rendering, headers, request fields, stack traces, or secrets.
- [ ] CLI-seam tests cover every success/error output mode, stdout/stderr separation, URL preservation, optional status-code omission, usage failures, and exact exit codes.
- [ ] Shell demos prove command substitution for plain output and structured extraction from JSON output.
