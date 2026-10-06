# 02: Deliver CLI Product Version commands, flags, and JSON output

**What to build:** Enable CLI users and automated scripts to inspect the installed Product Version via top-level commands and flags (`upit version`, `upit --version`, `upit -v`, and `--json`), rejecting invalid arguments with exit code 2.

**Blocked by:** 01: Establish authoritative Product Version source and runtime provider

**Status:** resolved

- [x] Top-level CLI invocations `upit version`, `upit --version`, and `upit -v` emit human-readable version text (`upit <version> (commit: <sha>, built: <date>)`) to standard output and exit with code 0.
- [x] Passing the `--json` flag to version commands or flags outputs structured JSON metadata containing version, commit, date, goVersion, os, and arch to standard output and exits with code 0.
- [x] Version commands and flags take precedence over upload argument parsing and do not fail with upload usage errors.
- [x] Extraneous or invalid arguments passed to version commands or flags emit a usage diagnostic to standard error and exit with code 2.
- [x] CLI runner tests verify all text, JSON, and error exit code behaviors.

