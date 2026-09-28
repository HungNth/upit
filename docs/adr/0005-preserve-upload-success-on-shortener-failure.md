# Preserve upload success when URL shortening fails

After an upload returns an Original URL, ordinary Shortener network, HTTP, parse, and extraction failures fall back to that Original URL as the Final URL, emit a sanitized warning on stderr, and keep the command successful. Shortener configuration is validated before upload, and an explicit Ctrl-C remains an interruption instead of being converted to success; this avoids hiding configuration mistakes or user intent while preventing a downstream post-processing outage from invalidating an irreversible completed upload.
