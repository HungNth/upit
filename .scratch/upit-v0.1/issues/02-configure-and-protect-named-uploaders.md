# 02: Configure and protect practical named Uploaders

**Parent specification:** [Upit v0.1 — Headless Upload CLI](../spec.md)

**What to build:** Let users define multiple practical Uploaders and select one safely. Requests can carry configured headers, query parameters, static multipart fields, and a named file part. Configuration failures must stop before network access, and credential-bearing values must not leak through diagnostics.

**Blocked by:** 01 — Upload one file with the default Uploader

**Status:** ready-for-human

- [ ] Global configuration version 1 accepts only the default Uploader and clipboard-default fields; unsupported future settings and unknown fields fail validation.
- [ ] Uploader configuration version 1 accepts only named Uploaders; unsupported shortener definitions and unknown fields fail validation.
- [ ] A Uploader defines method, absolute HTTP(S) URL, optional headers, optional query values, multipart body type, file-field name, optional static multipart fields, required URL extractor, and optional error extractor.
- [ ] Request values are literal strings; no template or environment-variable expansion occurs.
- [ ] `--uploader` overrides the configured default, and flags are accepted before or after the file operand.
- [ ] Missing files, invalid versions, absent defaults, unknown Uploaders, invalid request fields, unsupported body types, invalid JSONPath expressions, directories, and unreadable input fail before a request is sent.
- [ ] Missing configuration reports the exact expected location and points the user to repository examples; no configuration is generated automatically.
- [ ] Version-1 global and Uploader example files are created in this ticket, contain no real credentials or future fields, and pass the same strict loader used by the CLI.
- [ ] On Unix, the credential-bearing Uploader file is rejected when any group or other permission bit is present, with an actionable `chmod 600` correction; the POSIX mode check is not applied on Windows.
- [ ] Configured header, query, and multipart-field values are classified as sensitive and are available to a central diagnostic scrubber.
- [ ] Multipart `Content-Type` remains owned by Upit; a Uploader cannot override its generated boundary.
- [ ] CLI-seam tests prove Uploader precedence, both flag positions, headers, query values, static fields, strict validation, permission rejection, and zero network calls on invalid input.
- [ ] A demo uploads through a non-default named Uploader and shows the server received every configured request value.

## Comments

- 2026-10-03 backlog audit: this ticket is dispositioned `wontfix`/`ready-for-human`. Its version-1 Uploader acceptance conflicts with approved v0.2/v0.3 version-2 cutovers and the no-compatibility principle. Historical version-1 behavior is not restored.
