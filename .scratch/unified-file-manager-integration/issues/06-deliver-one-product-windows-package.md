# 06: Deliver one-product Windows installation and Repair

**What to build:** Deliver Windows as one signed Upit installer/package containing Desktop, the Explorer command, the private one-shot worker, package identity, and the repair material required by Desktop. Installation and updates register `Upload with Upit` without consumer PowerShell, Repair is self-contained for trusted installs, and uninstall removes the integration.

**Blocked by:** 05: Manage Windows File Manager Integration from Desktop.

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Verify installer/package contents and trust inputs mechanically, then exercise install, state inspection, Repair, update, Explorer invocation, and uninstall on Windows without duplicating File Manager Upload behavior tests.

**Demo path:** Install Upit through the consumer package, invoke `Upload with Upit` from the Windows 11 primary menu, remove package registration, Repair it from Desktop without elevation, update Upit without a duplicate command, and uninstall without a stale Explorer action.

- [ ] One consumer installer/package delivers Desktop, the native Explorer command, the private one-shot worker, package identity, and signed registration material as one product.
- [ ] User-visible installer, application, notification, and Explorer surfaces use `Upit` or `Upload with Upit`; private component names do not create separate application entries.
- [ ] Consumer installation and update require no manual PowerShell flow and register or refresh the Windows 11 primary-menu command automatically.
- [ ] The installed product retains the exact signed registration artifact and matching external payload information needed for current-user Repair.
- [ ] Desktop Repair succeeds without elevation when the retained artifact and payload are intact and trusted.
- [ ] Missing, unsigned, untrusted, or mismatched repair inputs fail closed to reinstall guidance.
- [ ] Updates replace compatible Desktop, adapter, and worker components atomically and do not create duplicate package identities or Explorer commands.
- [ ] Uninstall removes package identity, COM registration, Explorer integration, and product payload without leaving a usable stale action.
- [ ] Ordinary unsigned verification validates package shape and negative Repair behavior without claiming signed installability.
- [ ] Existing CLI and Manual Upload behavior remains unchanged against the same Configuration Set.
- [ ] A local Windows smoke proves consumer install, registration, Desktop inspection, trusted Repair with development credentials, update continuity, and cleanup; protected publisher proof remains ticket 08.
