# 09: Prove the notarized macOS unified release

**What to build:** Execute and retain production evidence for the Developer ID-signed, Hardened Runtime, notarized, and stapled macOS Upit DMG. The protected run must prove drag installation, Finder discovery and Repair, File Manager Upload behavior, one-product identity, update continuity, CLI and Manual Upload continuity, and deterministic removal.

**Blocked by:** 07: Unify Upit product version and release metadata.

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Use the protected macOS release workflow and a real macOS 14+ Apple Silicon host. Unit, bundle-shape, direct-helper, and unsigned checks are prerequisites but not substitutes.

**Demo path:** Drag the notarized `Upit.app` into Applications, inspect non-mutating first-launch state, Repair deliberately removed registration, invoke `Upload with Upit` from Finder, update the product, verify CLI and Manual Upload, prepare removal, and confirm the Service disappears.

- [ ] A protected SemVer release signs every nested executable and bundle with Developer ID and Hardened Runtime, notarizes the distributable, staples the ticket, and validates Gatekeeper acceptance.
- [ ] Dragging one `Upit.app` into Applications is the complete consumer install flow; no installation shell script is required.
- [ ] User-visible application, notification, and Service-provider names present one Upit product, while `Upload with Upit` appears under Finder Services or Quick Actions.
- [ ] First launch performs a non-mutating inspection; deliberately absent registration produces `Needs Repair` and explicit Repair restores discoverable registration.
- [ ] Desktop guidance opens Keyboard Settings on a best-effort basis and never claims to observe private Services enablement or live Finder visibility.
- [ ] A real File Manager Upload proves default Uploader, optional Shortener, always-copy behavior, privacy-safe progress, cancellation, warning, Retry, and configuration recovery without opening Desktop ordinarily.
- [ ] Clipboard failure preserves upload success and `Copy Final URL` copies the existing result without a second upload request.
- [ ] Updating the notarized product preserves one versioned identity and does not duplicate Service registration.
- [ ] CLI and Manual Upload remain operational against the same Configuration Set.
- [ ] `Prepare to Remove Upit` unregisters the nested Service and outer application, refreshes dynamic services, and leaves no usable stale action after the bundle is removed.
- [ ] The workflow retains signatures, notarization/stapling evidence, checksums, version metadata, Finder smoke evidence, and cleanup evidence; no production-ready claim is made without all results.
