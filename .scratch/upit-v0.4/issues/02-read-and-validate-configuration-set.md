# 02: Read and validate the Configuration Set from `upit config`

**What to build:** Add the non-interactive `upit config` command tree and complete the read-only configuration workflow: locate the Configuration Set, show user-facing Global Configuration values, list available Uploaders and Shorteners, and validate the complete Configuration Set on demand. Each command must load only the documents its purpose requires.

**Blocked by:** 01: Strict shared configuration diagnostics

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Use the production CLI runner with temporary homes and real configuration documents. Assert command output, stdout/stderr separation, exit codes, document dependency boundaries, and no partial output on failure.

**Demo path:** With a temporary Configuration Set, run `config path`, `show`, both list commands, and `validate`; then break one document at a time and observe that unrelated read commands still work while full validation reports the correct deterministic error.

- [ ] Global help lists `upload` and `config`; `upload --help` and `config --help` show command-specific help without adding a separate `help <command>` grammar.
- [ ] Unknown config subcommands, unsupported flags, missing operands, and extra operands write config-specific usage to stderr and exit 2.
- [ ] Config command success uses stdout; configuration and validation failure uses stderr and exit 1; failed commands emit no partial stdout; config commands do not support `--json`.
- [ ] `config path` prints only the absolute fixed `~/.config/upit/` directory plus newline and succeeds even when the directory does not exist.
- [ ] `config show` strict-decodes only the Global Configuration, requires the supported version, does not resolve Uploader or Shortener references, and prints exactly three newline-terminated labels: `Default Uploader: "<name>"`, `Default Shortener: none` or `Default Shortener: "<name>"`, and `Copy to Clipboard: enabled` or `Copy to Clipboard: disabled`; names use JSON string escaping.
- [ ] `config show` remains usable when definition documents or references are invalid, and a Shortener literally named `none` is quoted so it is distinct from no selection.
- [ ] `list-uploaders` validates only the complete Uploader document and prints `Uploaders:` followed by exact case-sensitive names in lexicographic order as `- "<name>"` lines using JSON string escaping.
- [ ] `list-shorteners` validates only the complete Shortener document when present and uses the same `Shorteners:` plus quoted-bullet format; an absent optional document prints exactly `No Shorteners configured.` plus newline.
- [ ] `config validate` requires the Global Configuration and Uploader document. An absent Shortener document is valid only when no default Shortener is selected; a present Shortener document is always validated in full; a selected default requires a matching Shortener.
- [ ] Validation success prints exactly `Configuration is valid.` plus newline and exits 0.
- [ ] Upload and `config validate` use the same decoder, per-document validators, diagnostic paths, ordering, and redaction when they load the same documents.
- [ ] User-facing configuration documentation describes the read-only command workflow and preserves the fixed cross-platform Configuration Set location.
