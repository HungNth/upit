# 01: Always copy the File Manager Upload Final URL

**What to build:** Make every File Manager Upload attempt to copy its Final URL regardless of the Global Configuration clipboard preference, while preserving completed upload success and offering `Copy Final URL` when automatic copying fails. Keep the existing clipboard preference as the default only for CLI and Manual Upload.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the existing Wails-free `FileManagerUploadService` with real Configuration Set documents, a local HTTP endpoint, and deterministic clipboard behavior.

**Demo path:** Configure clipboard copying as disabled, invoke one File Manager Upload, and paste the copied Final URL. Then force clipboard failure, observe success-with-warning, choose `Copy Final URL`, and prove the upload endpoint received only one request.

- [x] File Manager Upload uses the default Uploader and optional default Shortener but forces clipboard copying instead of reading the Global Configuration clipboard preference.
- [x] A successful upload copies the Final URL even when the Global Configuration clipboard preference is disabled.
- [x] Automatic clipboard failure preserves upload success, emits only a privacy-safe warning, and creates one usable `Copy Final URL` action.
- [x] `Copy Final URL` copies the existing result without starting another upload, shortening request, or Configuration Set preflight.
- [x] Existing action-token privacy, single-use consumption, private permissions, and expiry behavior remain intact.
- [x] Shortener fallback still retains and attempts to copy the Original URL as the Final URL with a sanitized warning.
- [x] CLI and Manual Upload continue to honor their existing clipboard choices.
- [x] Desktop copy describing the Global Configuration preference states that it applies to CLI and Manual Upload, not File Manager Upload.
- [x] Permanent regression tests prove the always-copy and copy-failure recovery contracts through the existing behavior seam.
