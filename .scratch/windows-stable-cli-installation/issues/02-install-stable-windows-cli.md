# 02: Install the public CLI through a stable Windows launcher

**Parent specification:** Upit — Stable Windows CLI Installation and Transactional Version Layout

**What to build:** A fresh current-user Windows installation exposes `upit` as a stable terminal command while keeping the real CLI, Upit Desktop, and private File Manager helper together in one version-aligned Active Payload. New terminals can invoke the public CLI without knowing an internal payload path, and a failed first installation leaves no broken command or route.

**Blocked by:** 01: Prefactor Windows installation into one recoverable transaction seam.

**Status:** ready-for-agent

- [ ] Fresh installation commits one Active Payload under the readable product-version directory and no committed payload path ends in `.tmp`.
- [ ] The stable root executable is a launcher/forwarder, while the real CLI, Desktop, and File Manager helper remain together in the Active Payload.
- [ ] The Start Menu shortcut targets the Active Payload Desktop and File Manager Integration targets the Active Payload private helper directly; neither route goes through the public CLI.
- [ ] The launcher treats current-user Active Payload metadata as the sole authority, validates that it identifies a coherent committed payload inside the Upit installation root, and fails nonzero without scanning versions when metadata is invalid.
- [ ] The launcher preserves arguments, stdin, stdout, stderr, working directory, environment, console cancellation, Unicode paths, and the real CLI exit status.
- [ ] Fresh installation adds one normalized stable-root entry to User PATH only when an equivalent entry is absent, records whether Upit owns that entry, preserves unrelated entries and order, and broadcasts the Windows environment-change notification.
- [ ] A pre-existing equivalent stable-root PATH entry is preserved and recorded as user-owned rather than duplicated.
- [ ] Pre-commit verification proves version-aligned payload binaries, launcher-to-CLI invocation, Active Payload metadata, Classic Verb, Start Menu shortcut, PATH state, and uninstall metadata without opening Desktop or performing an upload.
- [ ] A failed fresh installation removes every newly published launcher, route, owned PATH entry, shortcut, metadata record, and incomplete payload while retaining explicit recovery material only when compensation cannot be proven.
- [ ] The native Windows smoke invokes `upit --help` and a local upload by command name from a newly started process, including spaces/Unicode, a controlled nonzero outcome, and cancellation behavior.
- [ ] Unsigned installation remains fully functional, and protected signing includes the stable launcher without creating a second product version.

## Implementation evidence

Ticket 02 implemented. Real 0.9.2 unsigned setup passed native fresh-install smoke: exact readable version layout and binary versions, command-name help in a new user environment, relative Unicode-file local upload, stderr/nonzero preservation, invalid/missing metadata rejection, real console cancellation without an orphan CLI, failed launcher publication with exact state restoration and incomplete-payload removal, fail-closed uninstall, and owned PATH removal. A separate run preserved a pre-existing equivalent user-owned PATH entry through uninstall. Windows integration regressions and native-worker/launcher Go vet passed. Evidence: `.build/ticket02-smoke-evidence.json` and `.build/ticket02-preexisting-path-evidence.json`. Update scenarios remain assigned to subsequent tickets; operator Explorer observations and optional signed builds were not exercised.
