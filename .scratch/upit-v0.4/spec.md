# Upit v0.4 — Configuration Management

Status: ready-for-human

## Problem Statement

Upit requires users to create and edit a strict, multi-document Configuration Set by hand. The CLI cannot show the active Global Configuration, list available Uploaders or Shorteners, validate the complete Configuration Set on demand, or safely change defaults. Configuration failures are strict but not consistently actionable: errors may lack a precise document path, validation order can depend on Go map iteration, duplicate JSON keys are accepted implicitly, and upload-time diagnostics are not designed as a configuration-management workflow.

Upit also publishes no machine-readable schemas. Editors therefore cannot provide reliable autocomplete or preflight validation for the Global Configuration, Uploaders, or Shorteners, and users cannot tell which rules are representable in JSON Schema versus enforced only by Upit.

The next release must add non-interactive configuration management without creating an editor, wizard, compatibility layer, resident process, or second validation implementation. Writes must be safe across Windows, macOS, and Linux while stating the operating-system limits honestly.

## Solution

Add a non-interactive `upit config` command tree that resolves the fixed per-user Configuration Set, shows the Global Configuration, lists named Uploaders and Shorteners, validates all present configuration documents, changes the default Uploader or Shortener, clears the default Shortener, and enables or disables default clipboard copying.

Use one shared strict decoder and semantic validation path for upload and configuration commands. Diagnostics report one deterministic, actionable error with an absolute file path and precise configuration path. Validation rejects duplicate JSON keys, invalid definition names, incomplete documents, and case-insensitive duplicate Shortener headers while preserving the existing Global Configuration version 2, Uploader document version 2, and Shortener document version 1.

Publish hand-written Draft 2020-12 schemas for all three documents. Schemas strictly describe every intra-document rule JSON Schema can express; Go remains authoritative for cross-document references, filesystem permissions, protocol syntax, and other semantic rules.

Only the Global Configuration is writable. Mutations use canonical JSON and a staged replacement flow rather than in-place writes. POSIX systems use atomic rename semantics. Windows uses the native replacement primitive without remove-then-rename, with the documented limitation that Windows does not provide the same failure guarantee as POSIX without a backup. Upit does not claim crash durability, adversarial path-race protection, or full metadata preservation.

## User Stories

1. As a user, I want one `config` command namespace, so that configuration work is discoverable without mixing it into upload flags.
2. As a user, I want configuration commands to remain non-interactive, so that they work in terminals, SSH sessions, scripts, and CI.
3. As a headless user, I want configuration management to require no GUI, WebView, tray process, or daemon, so that server use remains unchanged.
4. As a user, I want `upit config path` to print the absolute Configuration Set directory, so that I can open or script against the fixed location.
5. As a user, I want `config path` to work before the directory exists, so that it can guide first-time manual setup.
6. As a cross-platform user, I want `config path` to use the same `~/.config/upit/` location on Windows, macOS, and Linux, so that existing documentation and automation remain valid.
7. As a user, I want `upit config show` to display the selected default Uploader, optional default Shortener, and clipboard-copying default, so that I can inspect active global choices without reading JSON.
8. As a user repairing configuration, I want `config show` to read only the Global Configuration and not require Uploader or Shortener documents to be valid, so that an unrelated document error does not hide my current defaults.
9. As a user repairing a dangling reference, I want `config show` to display the configured name even when that name does not exist, so that I can see what must be changed.
10. As a user, I want displayed Uploader and Shortener names quoted and escaped, so that names containing internal spaces remain unambiguous.
11. As a user, I want an absent default Shortener displayed as `none`, so that it is distinct from a Shortener literally named `none`.
12. As a user, I want clipboard state displayed as `enabled` or `disabled`, so that the setting is readable without understanding JSON booleans.
13. As a user, I want `upit config list-uploaders` to show every configured Uploader name, so that I can choose a valid default without opening the document.
14. As a user, I want `upit config list-shorteners` to show every configured Shortener name, so that I can choose a valid default without opening the document.
15. As a user, I want listed names sorted deterministically, so that repeated output is stable across runs and platforms.
16. As a user, I want listed names rendered as JSON-quoted bullet items under a clear heading, so that internal spaces and Unicode remain readable.
17. As a user, I want `list-uploaders` to validate the complete Uploader document before printing, so that it never lists names from a document Upit would reject.
18. As a user, I want `list-shorteners` to validate the complete Shortener document when it exists, so that it never lists names from a document Upit would reject.
19. As a user who does not use shortening, I want `list-shorteners` to succeed with `No Shorteners configured.` when the optional Shortener document is absent, so that absence is not treated as failure.
20. As a user repairing another document, I want each list command to depend only on its own definition document, so that a broken Global Configuration does not prevent discovery.
21. As a user, I want `upit config validate` to validate the complete Configuration Set without uploading a file or making a network request, so that I can detect errors safely.
22. As a user, I want validation to require the Global Configuration and Uploader document, so that a Configuration Set capable of upload has both required documents.
23. As a user who does not use shortening, I want the Shortener document to remain optional when no default Shortener is selected, so that current valid setups remain valid.
24. As a user, I want an existing Shortener document validated in full even when no Shortener is selected, so that dormant errors are found before later use.
25. As a user, I want a selected default Shortener to require an existing Shortener document and matching named Shortener, so that validation catches dangling selections.
26. As a user, I want validation success to print `Configuration is valid.`, so that interactive use receives explicit confirmation.
27. As an automation author, I want validation success to exit 0 and invalid configuration to exit 1, so that scripts can use the command without parsing prose.
28. As an automation author, I want CLI syntax errors to remain exit 2, so that malformed commands are distinct from invalid configuration.
29. As an automation author, I want successful config command output only on stdout and all failures only on stderr, so that stream handling remains predictable.
30. As an automation author, I want a failed command to produce no partial stdout, so that consumers never receive incomplete state as success data.
31. As a user, I want `upit config set-default-uploader <name>` to select an existing Uploader, so that I can change the upload default without editing JSON.
32. As a user, I want the selected Uploader name matched case-sensitively, so that configuration map semantics remain exact.
33. As a user, I want setting a default Uploader to validate every Uploader in that document, so that one mutation cannot bless a partially invalid definition document.
34. As a user repairing an invalid default Uploader, I want the requested change applied before the resulting Global Configuration is semantically validated, so that the command can repair the field it owns.
35. As a user, I want setting an unknown Uploader to fail without changing the Global Configuration, so that the command cannot create a dangling default.
36. As a user, I want `upit config set-default-shortener <name>` to select an existing Shortener, so that I can enable a configured shortening destination without editing JSON.
37. As a user, I want setting a default Shortener to validate every Shortener in that document, so that one mutation cannot bless a partially invalid definition document.
38. As a user, I want setting an unknown Shortener or using a missing Shortener document to fail without changing the Global Configuration, so that the command cannot create a dangling default.
39. As a user, I want `upit config clear-default-shortener` to disable default shortening, so that future uploads return the Original URL unless a Shortener is explicitly selected.
40. As a user repairing a malformed selected Shortener document, I want clearing the default Shortener to require only a decodable, supported Global Configuration, so that the broken Shortener does not block disabling it.
41. As a user, I accept that an existing malformed optional Shortener document remains invalid under `config validate` after clearing the default, so that full validation still reports dormant errors.
42. As a user, I want `upit config enable-clipboard` to enable default clipboard copying, so that I can manage the Global Configuration without editing JSON.
43. As a user, I want `upit config disable-clipboard` to disable default clipboard copying, so that headless or automated invocations do not attempt clipboard work by default.
44. As a headless user, I want clipboard commands not to probe for `pbcopy`, `clip`, `wl-copy`, `xclip`, or `xsel`, so that configuration can be provisioned independently of the current machine state.
45. As a user, I want clipboard mutations to preserve the existing Uploader and Shortener defaults, so that changing one global setting does not alter another.
46. As a user, I want default-selection mutations to preserve the clipboard setting and unrelated default, so that writes are field-specific.
47. As an automation author, I want setting an already-selected value to succeed without rewriting the file, so that configuration commands are idempotent.
48. As a user, I want no-op Uploader and Shortener mutations to perform the same relevant-document validation as real mutations, so that a no-op cannot hide a broken definition document.
49. As a user, I want no-op clipboard mutations to validate the resulting Global Configuration, so that they cannot hide unrelated invalid global fields.
50. As a user, I want no-op commands to state that the value is already set or already clear, so that I can distinguish idempotence from an unreported write.
51. As a user, I want successful mutations to confirm the resulting setting briefly, so that interactive use has clear feedback.
52. As a user, I want Uploader and Shortener names to be non-empty printable Unicode with no leading or trailing Unicode whitespace, so that names remain expressive without producing invisible or multiline command output.
53. As a user, I want internal spaces and international characters allowed in names, so that naming is not limited to ASCII slugs.
54. As a user, I want invalid names rejected rather than silently trimmed or normalized, so that references remain exact.
55. As a user, I want the same name rules applied to definition keys and Global Configuration references, so that documents cannot disagree about name validity.
56. As a user, I want duplicate JSON object keys rejected, so that the value Upit uses cannot differ from the value I think I configured.
57. As a user, I want duplicate-key rejection applied at every nesting level in all three documents, so that ambiguity cannot hide in headers, request data, or nested objects.
58. As a user, I want unknown fields rejected in all fixed-shape objects, so that misspellings and stale options never pass silently.
59. As a user, I want an existing Uploader document to contain at least one Uploader, so that a required document cannot be structurally useless.
60. As a user, I want an existing Shortener document to contain at least one Shortener, so that an optional document cannot be structurally useless.
61. As a user, I want each Uploader and Shortener definition to contain its required request, response, and URL-extraction structure, so that incomplete definitions fail at the relevant path.
62. As a user, I want Global Configuration version 2, Uploader document version 2, and Shortener document version 1 retained, so that v0.4 does not force a format-only migration.
63. As a user, I accept that empty names, empty definition maps, ambiguous duplicate keys, and case-variant duplicate Shortener headers are rejected as validation bug fixes under the existing versions, so that malformed configurations fail earlier without migrating valid configurations.
64. As a user, I want unsupported versions rejected with the file, `$.version`, received version, expected version, a statement that automatic migration is unavailable, and the matching schema filename, so that manual remediation is precise.
65. As a user, I want Shortener header names treated case-insensitively for duplicate detection, so that Shortener validation matches HTTP semantics and Uploader behavior.
66. As a user, I want Shortener Response Extractors to remain JSON-only in v0.4, so that configuration management does not silently expand Shortener protocol behavior.
67. As a user, I want all existing Uploader Request Body Mode and Response Extractor rules preserved, so that schema publication does not change upload protocols.
68. As a user, I want all existing Shortener request-template and JSONPath rules preserved, so that schema publication does not change shortening behavior.
69. As a user, I want configuration validation to remain local and make no endpoint requests, so that validation cannot leak credentials or trigger side effects.
70. As a user, I want one deterministic validation error at a time, so that each run presents one clear repair step.
71. As a user, I want document validation ordered as Global Configuration, Uploaders, then Shorteners, so that the first error is stable.
72. As a user, I want named definitions validated in exact lexicographic order, so that Go map iteration cannot change the first error.
73. As a user, I want field checks ordered deterministically inside each definition, so that repeated runs on the same files produce the same diagnostic.
74. As a user, I want semantic diagnostics to include the absolute file path, an RFC 9535-style configuration path, the violated rule, and the corrective action, so that I can find and repair the exact value.
75. As a user, I want map keys bracket-quoted and escaped in diagnostic paths, so that dots, spaces, quotes, and Unicode names remain unambiguous.
76. As a user, I want malformed JSON diagnostics to use the root path plus line and column when an exact field path is unavailable, so that syntax errors remain actionable without guessed paths.
77. As a security-conscious user, I want diagnostics never to echo credential values, request values, clipboard content, or secret-bearing URL values, so that better errors do not weaken redaction.
78. As a user, I want definition names and safe field names quoted or escaped in diagnostics, so that control sequences cannot alter terminal output.
79. As an existing upload user, I want upload-time configuration errors to use the same decoder, validation rules, deterministic ordering, and actionable paths as `config validate`, so that the two entry points cannot drift.
80. As a maintainer, I want one shared validation implementation rather than a CLI validator and an upload validator, so that fixes apply once.
81. As an editor user, I want a Draft 2020-12 schema for the Global Configuration, so that supported fields and defaults are discoverable before runtime.
82. As an editor user, I want a Draft 2020-12 schema for Uploaders, so that Request Body Modes and Response Extractors receive autocomplete and validation.
83. As an editor user, I want a Draft 2020-12 schema for Shorteners, so that the optional document has equal tooling support.
84. As a user, I want schemas committed as stable repository artifacts under a `schemas` directory, so that they can be inspected and referenced without running Upit.
85. As a user, I want schema files to declare the Draft 2020-12 meta-schema, so that validators select the correct dialect.
86. As a maintainer, I want schema files to omit `$id` until Upit has a stable schema-hosting URL, so that the project does not publish a misleading identifier.
87. As a user, I want configuration documents to remain free of a `$schema` field, so that the strict runtime format and document versions do not change for editor integration.
88. As an editor user, I want documentation showing filename-based schema association, so that I can enable schemas without modifying configuration documents.
89. As a user, I want fixed-shape schema objects to reject additional properties, so that editor validation matches strict runtime decoding.
90. As a user, I want schemas to express required fields, supported versions, enums, conditional body fields, conditional extractor fields, and non-empty definition maps wherever Draft 2020-12 can do so, so that editor feedback is as strict as the format permits.
91. As a user, I want schema descriptions to document rules Draft 2020-12 cannot express exactly, so that a schema-valid document does not create unexplained runtime failures.
92. As a maintainer, I want Go to remain authoritative for cross-document references, filesystem permissions, HTTP method and URL semantics, header rules, JSONPath and regular-expression compilation, media types, and recursive placeholder counts, so that Upit does not add a runtime schema-validator dependency.
93. As a maintainer, I want hand-written schemas instead of generated schemas, so that conditional and protocol-specific rules remain explicit without code-generation infrastructure.
94. As a maintainer, I want shared parity fixtures to prove every schema-expressible rule agrees with runtime validation, so that public schema artifacts cannot drift silently.
95. As a user, I want Global Configuration mutations to serialize all four global fields in a canonical order with four-space indentation and a trailing newline, so that resulting files are stable and readable.
96. As a user, I accept that a mutation canonicalizes whitespace and key order rather than preserving original formatting, so that the writer remains simple and deterministic.
97. As a user, I want mutation commands to require an existing Global Configuration and directory, so that v0.4 never invents missing values or bootstraps files implicitly.
98. As a user, I want a symlinked Global Configuration refused for mutation with an actionable error, so that an atomic replace does not silently break a dotfiles link.
99. As a user, I want read and validation commands to continue following ordinary filesystem reads, so that a symlinked configuration can still be inspected when no write is requested.
100. As a user, I want staged content written in the target directory, flushed, and closed before publication, so that readers never observe a partially written configuration.
101. As a user, I want pre-publication failures to leave the original file unchanged and clean up the staged file, so that failed writes do not create clutter or data loss.
102. As a user, I want Upit never to implement replacement as remove-old-then-rename-new, so that there is no intentional interval where the Global Configuration is missing.
103. As a Unix user, I want replacement to preserve the existing permission bits, so that a mutation does not broaden or unexpectedly narrow ordinary file access.
104. As a Unix user, I want Uploader and Shortener documents to retain the existing group/other permission rejection, so that credential-bearing documents still require private permissions.
105. As a Windows user, I want credential permission-bit checks skipped as they are today, so that Unix mode rules are not simulated incorrectly.
106. As a user, I want the Global Configuration exempt from the credential-file `0600` requirement, so that v0.4 does not add an unrelated permission migration.
107. As a Windows user, I want native replacement used instead of a delete-and-rename fallback, so that writes use the strongest ordinary Windows primitive available.
108. As a Windows user, I want an ambiguous publication failure to retain the staged file and report its recovery path, so that the replacement content is not destroyed when the target state cannot be guaranteed.
109. As a cross-platform user, I accept that Windows replacement does not promise the POSIX old-target-on-error guarantee or full ACL/stream preservation without a backup, so that v0.4 can remain backup-free and honest about platform limits.
110. As a user, I accept that v0.4 guarantees atomic visibility rather than immediate power-loss durability, so that the implementation does not claim a portable guarantee filesystems cannot provide.
111. As a user, I want no lock file and last completed writer wins, so that short configuration mutations do not introduce stale-lock lifecycle or background coordination.
112. As a user, I want no persistent backup file, so that configuration management does not retain stale copies or secrets by default.
113. As a user, I accept a point-in-time symlink preflight under the assumption that my Configuration Set directory is user-owned and not being adversarially raced by another same-user process, so that v0.4 avoids platform-specific handle-relative security machinery.
114. As an existing CLI user, I want `upit --help` and `upit help` to list both `upload` and `config`, so that the new command is discoverable.
115. As a user, I want `upit upload --help` and `upit config --help` to describe their own command surfaces, so that help remains bounded and relevant.
116. As a user, I want unknown config subcommands, missing operands, and extra operands to produce config-specific usage on stderr and exit 2, so that mistakes are distinct from configuration failures.
117. As an existing upload user, I want upload success output, upload JSON output, Shortener fallback, clipboard warning behavior, cancellation, and upload exit codes unchanged, so that configuration management does not break automation.
118. As a maintainer, I want the CLI to remain the primary external seam and the application module to own configuration loading, validation, and safe persistence, so that validation is not duplicated in command parsing.
119. As a maintainer, I want no public strategy interfaces, factories, schema-generation framework, or provider-specific adapters, so that v0.4 adds only the seams required by behavior.
120. As a maintainer, I want the v0.4 roadmap documentation updated without rewriting historical v0.1 planning artifacts, so that current status is accurate without mixing feature work with archival cleanup.

## Implementation Decisions

- v0.4 is named **Configuration Management** and remains CLI-only, headless-first, and process-per-invocation.
- The canonical terms are **Global Configuration** for `config.json` and **Configuration Set** for the per-user Global Configuration plus the documents defining Uploaders and optional Shorteners.
- The Configuration Set remains fixed under `~/.config/upit/` on Windows, macOS, and Linux. No native platform config directory, environment override, command flag, or alternate profile directory is added.
- The command surface is `upit config path`, `show`, `validate`, `list-uploaders`, `list-shorteners`, `set-default-uploader <name>`, `set-default-shortener <name>`, `clear-default-shortener`, `enable-clipboard`, and `disable-clipboard`.
- Config commands are non-interactive. They do not prompt, launch an editor, open a browser, or read secrets from stdin.
- Global help lists `upload` and `config`. `upload --help` and `config --help` show bounded command-specific help. A separate `help <command>` grammar is not added.
- Config commands do not support `--json`. Unknown flags, unknown subcommands, missing operands, and extra operands are usage failures.
- Exit code 0 means command success, exit code 1 means configuration, validation, filesystem, or write failure, and exit code 2 means invalid CLI usage. Config commands introduce no exit-130 path because they perform no long-running work.
- Successful command output and help use stdout. Usage, configuration, validation, and write failures use stderr. A failed command emits no partial stdout.
- `config path` resolves and prints only the absolute Configuration Set directory plus a newline. It does not require the directory or any document to exist.
- `config show` strict-decodes the Global Configuration and requires a supported version, but it does not load definition documents or resolve default references. It prints `Default Uploader`, `Default Shortener`, and `Copy to Clipboard`; names are JSON-quoted, no Shortener is `none`, and clipboard state is `enabled` or `disabled`.
- `list-uploaders` loads and validates only the entire Uploader document, then prints exact case-sensitive names in lexicographic order as JSON-quoted bullet items under `Uploaders:`.
- `list-shorteners` loads and validates only the entire Shortener document when present. An absent document succeeds with `No Shorteners configured.`; a present invalid document fails without partial output.
- `config validate` requires the Global Configuration and Uploader document. The Shortener document is optional only when absent and no default Shortener is selected. If the Shortener document exists, `validate` checks every Shortener regardless of selection. A selected default Shortener requires the document and matching name.
- Validation success prints exactly `Configuration is valid.` plus a newline.
- Only the Global Configuration is writable. Uploader and Shortener creation, editing, rename, and deletion remain manual JSON operations.
- A mutation strict-decodes the existing Global Configuration, rejects duplicate or unknown fields, and requires the supported version. It applies the requested change only to an in-memory value, validates the resulting Global Configuration, and publishes nothing unless every required check succeeds. This lets the command repair the field it owns without allowing unrelated global invalidity to persist.
- `set-default-uploader` validates the complete Uploader document and requires the requested Uploader. It does not load the Shortener document.
- `set-default-shortener` validates the complete Shortener document and requires the requested Shortener. It does not load the Uploader document.
- `clear-default-shortener`, `enable-clipboard`, and `disable-clipboard` require only the Global Configuration. Clearing can repair a bad default Shortener without loading a malformed Shortener document.
- No-op mutations return success without rewriting the file. They still perform the same validation as a real mutation for the relevant component.
- Mutation stdout is literal and newline-terminated: `Default Uploader set to "<name>".`, `Default Uploader is already "<name>".`, `Default Shortener set to "<name>".`, `Default Shortener is already "<name>".`, `Default Shortener cleared.`, `Default Shortener is already clear.`, `Clipboard copying enabled.`, `Clipboard copying is already enabled.`, `Clipboard copying disabled.`, or `Clipboard copying is already disabled.` Names use JSON string escaping inside the quotes.
- Global mutations preserve fields they do not own. They canonically reserialize all four global fields in this order: version, default Uploader, default Shortener, and clipboard boolean; formatting is four-space indentation plus a trailing newline.
- The Global Configuration remains version 2, the Uploader document remains version 2, and the Shortener document remains version 1. There is no compatibility reader, automatic migration, migration command, or version bump.
- Unsupported-version diagnostics identify the absolute file, `$.version`, received and expected versions, state that automatic migration is unsupported, and identify the corresponding published schema filename.
- Validation tightening for duplicate keys, usable document structure, valid names, and case-insensitive duplicate Shortener headers is treated as correction of malformed or ambiguous configurations under the existing versions, not as a format migration.
- The shared JSON reader rejects trailing values, unknown fields, and duplicate object keys at every depth. Duplicate-key detection occurs before Go map decoding can discard earlier values.
- Uploader and Shortener definition names are case-sensitive, non-empty printable Unicode strings with no leading or trailing Unicode whitespace. Internal whitespace and international characters are allowed. Invalid names are rejected, never trimmed or normalized.
- The Uploader document must contain at least one Uploader. A present Shortener document must contain at least one Shortener. Every definition requires its request, response, and URL extractor structure.
- Existing v0.3 Uploader Request Body Mode and Response Extractor contracts remain authoritative. v0.4 adds no protocol modes or extractors.
- Shortener Response Extractors remain JSON-only. Shortener header names become case-insensitively unique, matching HTTP semantics and Uploader validation.
- Upload, config validation, list commands, and mutation commands reuse one strict decoder and shared semantic validators. Their orchestration differs only by which documents each command intentionally loads.
- Diagnostics return the first error only. Document order is Global Configuration, Uploader document, then Shortener document. Named definitions use exact lexicographic order; field checks use fixed order.
- Semantic diagnostics use the absolute file path, an RFC 9535-style configuration path, a concise cause, and an actionable correction. Dynamic keys use bracket notation with escaping.
- Malformed JSON diagnostics use the root path plus line and column where an exact semantic path is unavailable. The implementation does not guess a field path.
- Existing sensitive-value redaction applies to all new diagnostics. Credential values, request values, static body values, and secret-bearing URLs are never echoed. Safe names and field identifiers are quoted or escaped.
- Three hand-written Draft 2020-12 schemas are published at `schemas/config.schema.json`, `schemas/custom-uploader.schema.json`, and `schemas/custom-shortener.schema.json`.
- Each schema declares `https://json-schema.org/draft/2020-12/schema` through its own `$schema` keyword and omits `$id` until the project has a stable hosting URL.
- User configuration documents do not gain a `$schema` property. The README documents three filename associations scoped to an opened Configuration Set: `config.json` → `schemas/config.schema.json`, `custom-uploader.json` → `schemas/custom-uploader.schema.json`, and `custom-shortener.json` → `schemas/custom-shortener.schema.json`.
- Fixed-shape schema objects use strict property sets. Schemas express required members, supported-version constants, primitive types, enums, conditional Request Body Mode fields, conditional Response Extractor fields, non-empty definition maps, and every other intra-document rule supported by Draft 2020-12.
- Rules JSON Schema cannot express exactly are documented with ordinary `description` annotations rather than custom keywords. These include Go HTTP method and URL semantics, header validity and case-insensitive uniqueness, Go JSONPath and regular-expression compilation, MIME parsing, recursive exact-one-placeholder rules, filesystem permissions, and cross-document references.
- Go validation remains the execution authority for semantic and cross-document rules. Production does not load or execute the published schema files and does not add a runtime schema-validator dependency.
- Shared valid and invalid fixtures keep schema-expressible rules behaviorally aligned between Draft 2020-12 validation and production config validation.
- Global writes require the config directory and Global Configuration to exist. v0.4 does not create either one implicitly.
- Mutations refuse a Global Configuration that is a symlink or equivalent supported reparse-point link at preflight. Reads may follow ordinary filesystem behavior.
- The writer stages a uniquely named file in the target directory, writes canonical bytes, applies the existing POSIX permission bits where meaningful, synchronizes and closes the staged file, then publishes it with the platform replacement primitive.
- Pre-publication failures leave the original byte-for-byte unchanged and remove the staged file. The implementation never removes the target before publishing the replacement.
- Linux and macOS publish with POSIX rename semantics. The specification accepts the POSIX `EIO` limitation and does not claim guarantees beyond the operating-system contract.
- Windows publishes with the native replace-existing move primitive rather than remove-then-rename. The specification does not claim POSIX-style target preservation on every reported Windows failure, full ACL/stream preservation, or behavior stronger than the documented Windows API.
- When publication failure leaves target state ambiguous, Upit retains the staged file instead of deleting potentially recoverable content. Its basename matches `.config.json.upit-tmp-*`, it has the same intended POSIX permission bits as the replacement, and stderr reports its absolute recovery path without including file contents. Upit does not auto-retry, auto-promote, or auto-delete retained recovery files; later invocations ignore them because configuration discovery uses fixed filenames.
- The write contract provides atomic visibility of complete old or new content under ordinary supported-filesystem behavior. It does not promise immediate power-loss durability after success.
- The writer does not create a backup or lock file. Concurrent successful writers are last-completed-write-wins.
- POSIX permission bits are preserved for the Global Configuration. Extended ACLs, xattrs, ownership, Windows ACLs, alternate streams, and other metadata are not part of the v0.4 preservation contract.
- Existing Unix credential-document permission checks remain: Uploader and present Shortener documents reject group or other permission bits. Windows skips these Unix mode checks. The Global Configuration is not newly required to use mode `0600`.
- The filesystem threat model assumes the fixed Configuration Set directory is user-owned and is not being adversarially raced by another same-user process. Symlink refusal is a point-in-time preflight, not a handle-relative anti-race security boundary.
- Configuration management stays behind the existing application/deep-module boundary. The CLI parses commands and renders outputs; it does not duplicate validation or persistence logic.
- No new public request/response packages, provider adapters, validator interfaces, factories, schema generators, or filesystem abstraction layers are introduced unless a concrete platform test cannot be written without one. Any necessary publication seam remains private and narrow.
- The architecture roadmap is updated for v0.4 when the feature is delivered. Historical v0.1 planning statuses remain untouched.

## Testing Decisions

- The primary permanent seam is the production CLI command-running interface with a temporary home directory, real temporary configuration files, and captured stdout/stderr. This follows the existing CLI tests and proves command parsing, document selection, diagnostics, output, exit codes, and persisted state together.
- CLI tests cover every config subcommand, global and command-specific help, unknown subcommands, missing and extra operands, unsupported flags, stdout/stderr separation, and exit codes 0, 1, and 2.
- `config path` tests prove the fixed absolute directory is returned even when it does not exist and that home-directory resolution failure produces no partial stdout.
- `config show` tests cover valid defaults, no default Shortener, a Shortener literally named `none`, clipboard states, dangling references, invalid unrelated documents, malformed Global Configuration, unsupported Global version, and safe quoting.
- List-command tests cover deterministic exact-case ordering, Unicode and internal-space names, invalid names, invalid unselected definitions, missing optional Shortener document, present empty documents, and no partial output on failure.
- Validation tests cover required versus optional documents, the exact Shortener matrix: absent plus no selected default is valid; present means validate every Shortener; selected plus absent or missing name is invalid.
- Shared CLI validation tests prove upload and `config validate` report the same document-level error paths, deterministic ordering, duplicate-key failures, name failures, and sensitive-value redaction when they load the same documents.
- Mutation tests cover every successful change, canonical serialization, preservation of unrelated fields, no-op output and unchanged bytes/mtime, component-specific validation, repair of the owned field, rejection of unrelated invalid global fields, unknown target names, and missing documents.
- Clipboard mutation tests prove no platform helper is probed and no clipboard operation occurs during configuration management.
- Diagnostic tests cover absolute paths, bracket-quoted dynamic keys, deterministic document/definition/field order, malformed JSON line and column, unknown fields, duplicate keys at multiple depths, unsupported-version remediation, and one-error-only behavior.
- Security tests prove diagnostics never emit configured header values, query values, request body values, secret-bearing endpoint values, or control-character output.
- Name tests cover printable Unicode, internal whitespace, exact case sensitivity, empty names, leading/trailing Unicode whitespace, control characters, format characters, and no implicit trimming or normalization.
- Existing Uploader protocol tests remain the regression proof for body modes and Response Extractors. Existing Shortener tests remain the regression proof that Shortener extractors stay JSON-only and runtime fallback behavior is unchanged.
- New Shortener validation coverage proves case-insensitive duplicate header names are rejected while distinct valid names remain accepted.
- The schema parity seam uses shared fixtures evaluated by both a conforming Draft 2020-12 test validator and production `config validate` behavior. A test-only validator dependency is acceptable; production must not depend on it.
- Schema parity fixtures cover versions, required fields, additional properties, non-empty definition maps, name minimums, body-mode conditionals, extractor conditionals, null-versus-absent fields, and every other rule representable in Draft 2020-12.
- Runtime-only fixture cases cover duplicate JSON keys, printable/trimmed Unicode names, HTTP method and URL semantics, header validity and case-insensitive duplicates, MIME parsing, JSONPath and regular-expression compilation, placeholder counts, permissions, and cross-document references. Schema descriptions are checked by review, not by tautological source-text tests.
- Schema tests validate all repository example documents and representative canonical documents. They do not test that files are merely non-empty or contain expected strings.
- Focused lower-level filesystem tests are limited to invariants unsafe or impractical to induce through the CLI: symlink refusal, POSIX mode preservation, staged-file cleanup before publication, retained recovery content after ambiguous publication failure, and platform publication behavior.
- Platform-specific publication tests use real temporary filesystems and build constraints. They do not mock filesystem calls merely to assert forwarding.
- POSIX tests prove readers never observe an intentional missing-target interval, ordinary publication failure leaves the old target available, and a successful replacement exposes complete canonical content.
- Windows tests exercise the native replace-existing path on Windows itself. Cross-compilation is not accepted as behavioral proof. Tests verify no remove-first fallback, complete-content visibility, ordinary target-open failure handling, and recovery-path reporting when state cannot be established.
- Filesystem tests do not claim to simulate power loss, hardware `EIO`, adversarial same-user path swaps, every filesystem, or metadata guarantees excluded by the specification.
- Race-sensitive concurrent writer tests prove complete-file visibility and last-completed-write-wins without asserting a deterministic winner.
- Focused lower-level tests must remain about observable invariants, not private helper names, call counts, source text, or mock echoes.
- Completion requires an actual built binary smoke on the host platform that runs `config path`, validates a real temporary Configuration Set, lists definitions, changes each managed Global Configuration field, observes canonical persisted bytes, and confirms a subsequent `config show` result.
- Native Windows and POSIX execution are required for their respective publication paths; macOS and Linux retain ordinary POSIX behavior but supported release CI should run the config command suite on all three target operating systems.

## Out of Scope

- Interactive prompts, questionnaires, terminal forms, editors, or wizards.
- Creating the Configuration Set directory or bootstrapping missing documents.
- Creating, editing, renaming, deleting, importing, or exporting Uploaders or Shorteners.
- A generic `get`, `set`, or `unset` configuration key API.
- A command to edit arbitrary Global Configuration fields beyond default Uploader, default Shortener, and clipboard copying.
- JSON output for config commands or a new machine-readable config-diagnostic API.
- Multiple simultaneous diagnostics or aggregate error reports.
- Automatic migration, a migration command, compatibility readers, version fallbacks, or dual-version support.
- Changing Global Configuration version 2, Uploader document version 2, or Shortener document version 1.
- Adding Request Body Modes or Response Extractors, including non-JSON Shortener extractors.
- Runtime execution of JSON Schema or a production schema-validator dependency.
- Generated schemas, schema code generation, custom schema keywords, embedded schemas, `config schema` commands, separate schema release archives, SchemaStore publication, or stable hosted schema IDs.
- Adding `$schema` to user configuration documents.
- Backups, rollback files, lock files, optimistic conflict detection, or transaction coordination between processes.
- Strict power-loss durability, parent-directory fsync guarantees, hardware-failure guarantees, or stronger claims than the underlying OS/filesystem APIs.
- Full preservation of ownership, ACLs, xattrs, alternate streams, object IDs, compression, encryption metadata, or other extended filesystem metadata.
- Handle-relative hardening against adversarial same-user path races or a general secure-filesystem abstraction.
- Following or replacing a symlinked Global Configuration during mutation.
- A configurable configuration directory, environment-selected profiles, workspace-local configuration, or platform-native config locations.
- Secret stores, operating-system keychains, environment-variable substitution, encrypted configuration, or per-field secret metadata.
- Clipboard-helper installation, probing, provisioning, or validation during config commands.
- Network reachability checks, endpoint authentication checks, test uploads, or provider-specific validation.
- Rewriting historical v0.1 issue statuses or broad architecture-document cleanup.
- Wails, WebView, desktop UI, system tray, notifications, daemon, resident worker, file watcher, or background process.
- Release installers, package-manager manifests, container images, or automated schema hosting.

## Further Notes

- The fixed cross-platform Configuration Set location remains governed by ADR 0001.
- Global Configuration and Configuration Set are canonical domain terms in `CONTEXT.md`; existing Uploader, Request Body Mode, Response Extractor, Shortener, Original URL, and Final URL terminology remains authoritative.
- The schema/runtime authority boundary is recorded in ADR 0009: schemas are strict public artifacts, while Go remains authoritative for semantic, cross-document, and filesystem rules.
- Primary-source research on atomic replacement and its Windows/POSIX limits is recorded under the v0.4 research notes. The specification deliberately weakens the impossible cross-platform “old target intact on every reported failure without backup” claim rather than hiding the Windows and POSIX limitations.
- The Shortener document rule is precise: absence is allowed when no default Shortener is selected; existence always causes full validation under `config validate`; selection requires existence and a matching valid Shortener.
- v0.1, v0.2, and v0.3 specifications remain authoritative for upload, Shortener fallback, clipboard runtime behavior, output contracts, cancellation, security, and protocol behavior not explicitly changed here.
- Implementation must follow approved v0.4 tickets produced after this specification. Repository changes remain uncommitted until the user explicitly authorizes a commit.
