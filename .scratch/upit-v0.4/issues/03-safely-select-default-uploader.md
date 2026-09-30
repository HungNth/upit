# 03: Safely select the default Uploader

**What to build:** Add safe Global Configuration persistence and the first complete mutation workflow through `upit config set-default-uploader <name>`. The command must validate the entire Uploader document, repair the owned Global Configuration field when possible, and publish canonical content without in-place writes or delete-before-rename behavior.

**Blocked by:** 02: Read and validate the Configuration Set from `upit config`

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Prove user behavior through the production CLI runner and real temporary files. Add focused real-filesystem, platform-gated tests only for publication invariants that cannot be induced safely through the CLI: symlink refusal, POSIX mode preservation, staged-file cleanup or retention, and native replacement behavior.

**Demo path:** Run the built CLI against an existing Configuration Set, select another valid Uploader, inspect canonical persisted bytes, rerun the same command to observe an idempotent no-op, and confirm `config show` plus a real upload use the new default.

- [ ] `set-default-uploader` requires an existing Global Configuration and Uploader document; it never creates the Configuration Set directory or bootstraps missing files.
- [ ] The command strict-decodes the existing Global Configuration and supported version, applies the requested default only to an in-memory value, validates the resulting Global Configuration, and publishes nothing on validation failure; an invalid current default can be repaired while unrelated invalid global fields still block the write.
- [ ] The complete Uploader document is validated and the requested case-sensitive Uploader must exist; an invalid sibling Uploader or unknown target prevents any write.
- [ ] A successful change preserves the Shortener and clipboard settings and prints exactly `Default Uploader set to "<name>".` plus newline, with JSON string escaping inside the quotes.
- [ ] Selecting the already-current Uploader performs the same Uploader validation, succeeds without rewriting bytes or modification time, and prints exactly `Default Uploader is already "<name>".` plus newline.
- [ ] Successful writes canonically serialize all four Global Configuration fields in stable order with four-space indentation and a trailing newline.
- [ ] Mutation refuses a symlinked Global Configuration with an actionable error; ordinary reads remain unaffected.
- [ ] The writer stages a uniquely named file matching `.config.json.upit-tmp-*` in the target directory, writes complete canonical content, applies existing POSIX permission bits where meaningful, synchronizes and closes the staged file, and only then publishes it.
- [ ] Pre-publication failures leave the original byte-for-byte unchanged and remove the staged file; the implementation never removes the target before replacement.
- [ ] Linux and macOS use ordinary POSIX rename publication and preserve the existing permission bits, within documented POSIX filesystem limits.
- [ ] Windows uses the native replace-existing move primitive without remove-then-rename. The behavior does not claim POSIX-style failure rollback or full ACL/stream preservation.
- [ ] If publication failure leaves target state ambiguous, the staged file is retained with the intended POSIX permission bits and stderr reports its absolute recovery path without file contents. Upit does not auto-retry, auto-promote, or auto-delete retained recovery files; later invocations ignore them because discovery uses fixed filenames.
- [ ] The writer creates no backup or lock file; concurrent successful writes expose complete files and use last-completed-write-wins semantics.
- [ ] The command promises atomic visibility under ordinary supported-filesystem behavior, not immediate power-loss durability, adversarial same-user path-race protection, or excluded extended-metadata preservation.
- [ ] Native Windows and POSIX tests execute their real publication paths; cross-compilation alone is not accepted as behavioral proof.
