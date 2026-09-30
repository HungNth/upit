# Avoid cross-document edit transactions

Upit v0.5 keeps edits to an existing Configuration Set single-document and rejects stale saves instead of adding lock files, automatic merge, or a multi-file transaction journal. Because Global Configuration references named Uploaders and Shorteners stored in other documents, a referenced definition cannot be renamed or deleted until the user changes or clears that reference; this preserves a valid Configuration Set after every published edit without claiming crash-atomic behavior that ordinary cross-file replacement cannot provide.
