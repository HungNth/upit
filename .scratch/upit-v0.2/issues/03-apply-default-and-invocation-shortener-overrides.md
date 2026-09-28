# 03: Apply the default and per-invocation Shortener overrides

**Parent specification:** [Upit v0.2 — URL Shortener](../spec.md)

**What to build:** Make Shortener selection predictable across global configuration and per-invocation flags. Users can rely on a default, override it by name, disable it for one upload, or omit shortening entirely without making the new credential file mandatory.

**Blocked by:** 02: Shorten explicitly with a named Shortener

Status: resolved

- [x] A non-empty `defaultShortener` selects that named Shortener when no shortening flag is supplied.
- [x] `--shortener <name>` overrides the configured default for one invocation.
- [x] `--no-shorten` disables the configured default and leaves Original URL equal to Final URL.
- [x] Supplying `--shortener` and `--no-shorten` together is a usage error with empty stdout and exit code 2, independent of flag order.
- [x] With no default and no flag, Upit does not open, permission-check, decode, or validate the Shortener file.
- [x] With `--no-shorten`, Upit does not open the Shortener file even when a default is configured.
- [x] Whenever a Shortener is selected, its complete configuration is validated before any upload request.
- [x] Existing Uploader and clipboard precedence remain unchanged.
- [x] CLI-seam tests cover default selection, explicit override, disable override, contradictory flags, absent selection, and interspersed flag positions.
- [x] Demo commands prove default shortening, explicit override, and one-shot disable behavior.
