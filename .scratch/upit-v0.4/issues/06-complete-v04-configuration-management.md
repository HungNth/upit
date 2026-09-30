# 06: Complete and demonstrate v0.4 Configuration Management

**What to build:** Deliver the complete documented v0.4 user journey across the read, validate, mutate, schema, and upload surfaces. Update current user and architecture documentation, run the built binary through the full Configuration Management workflow, and capture native evidence for every supported operating system rather than treating cross-compilation as behavioral proof.

**Blocked by:** 04: Manage default Shortener and clipboard behavior; 05: Publish strict Draft 2020-12 configuration schemas

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Use the actual built CLI for smoke verification, backed by the production CLI test suite, schema parity fixtures, and platform-specific writer tests completed by prior tickets. This ticket adds no duplicate lower-level test suites.

**Demo path:** Starting from valid example documents, associate the schemas, run every `config` read and mutation command, validate the resulting Configuration Set, perform a real local upload using the selected defaults, and observe that the process exits with no resident Upit process.

- [ ] User documentation explains the canonical Global Configuration and Configuration Set terms, fixed location, all config subcommands, literal output/exit behavior, mutation safety limits, and the three filename mappings to `schemas/config.schema.json`, `schemas/custom-uploader.schema.json`, and `schemas/custom-shortener.schema.json`.
- [ ] Architecture documentation marks v0.4 Configuration Management as delivered and removes or updates stale planned-v0.4 wording without rewriting historical v0.1 ticket statuses.
- [ ] Documentation states the exact optional Shortener rule: absence is valid only when none is selected; existence always triggers full `config validate`; selection requires a matching valid Shortener.
- [ ] Documentation states that only the Global Configuration is writable and that Uploader/Shortener creation, editing, rename, deletion, migration, and interactive management remain out of scope.
- [ ] Documentation states the Windows replacement caveat, POSIX/filesystem limits, point-in-time symlink refusal, no-backup/no-lock policy, last-completed-write-wins behavior, absence of a crash-durability guarantee, and the `.config.json.upit-tmp-*` recovery-artifact contract without overstating safety.
- [ ] An actual built-binary smoke runs `config path`, `show`, both list commands, `validate`, default Uploader and Shortener mutations, clear Shortener, and clipboard enable/disable against a real temporary Configuration Set.
- [ ] The smoke observes canonical persisted Global Configuration bytes, no-op behavior, final `config show` state, and a successful local upload using the selected defaults.
- [ ] Native Windows, macOS, and Linux verification exercises the command suite; native Windows and POSIX runs exercise their respective publication primitives. Cross-compilation alone is not accepted as proof.
- [ ] Existing upload plain/JSON output, failure stages, exit codes, Shortener fallback, clipboard warnings, timeout, cancellation, redaction, and no-resident-process behavior remain passing.
- [ ] All permanent tests are deterministic, isolated, full-suite-safe, and limited to consumer-visible behavior or unavoidable filesystem/schema invariants.
- [ ] No migration shim, compatibility path, generated schema system, interactive editor, unfinished placeholder, or follow-up implementation stub remains.
