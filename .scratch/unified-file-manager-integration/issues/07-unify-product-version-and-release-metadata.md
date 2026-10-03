# 07: Unify Upit product version and release metadata

**What to build:** Release Desktop and File Manager Integration under one Upit version across platform artifacts, metadata, checksums, documentation, and CI inputs. Ordinary verification proves one product identity and the complete unsigned behavior without claiming protected signing or notarization.

**Blocked by:** 01: Always copy the File Manager Upload Final URL; 04: Deliver one-product macOS installation and removal; 06: Deliver one-product Windows installation and Repair.

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Verify public artifact metadata and run the highest available unsigned product smokes: File Manager Upload behavior, Desktop integration states, package shape, install/update logic, and Linux unsupported behavior.

**Demo path:** Build Windows, macOS, and Linux Desktop artifacts from one version input; inspect matching product metadata, run unsigned platform verification, and observe that only one Upit identity is documented and exposed.

- [ ] One version input controls Desktop and every private platform component included in a release.
- [ ] Windows and macOS artifact names, embedded metadata, checksums, and release metadata report the same Upit version.
- [ ] No user-facing helper or adapter has an independent product version or installer identity.
- [ ] Release and installation documentation presents one Upit product while preserving technical documentation of internal process boundaries.
- [ ] Historical v0.6 and v0.7 specifications and tickets remain unchanged; the new specification and ADRs identify the superseding clipboard and product-delivery decisions.
- [ ] Ordinary CI exercises the always-copy regression, Desktop integration state tests, platform package validation, negative Repair paths, update invariants, and uninstall shape without signing credentials.
- [ ] Linux Desktop smoke shows `Not supported on Linux`, no unsupported actions, and unaffected Manual Upload and Configuration Set management.
- [ ] Ordinary artifacts and CI output clearly state that unsigned verification does not prove production registration, publisher trust, notarization, or native menu visibility.
- [ ] Release documentation names the protected Windows and macOS proof tickets as required gates before production readiness.
