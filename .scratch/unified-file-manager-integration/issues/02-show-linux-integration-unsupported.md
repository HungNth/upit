# 02: Show File Manager Integration as unsupported on Linux

**What to build:** Establish the platform-neutral File Manager Integration module and add its dedicated Desktop area through a complete Linux slice. Linux Desktop users see a truthful `Not supported on Linux` state without registration, Repair, settings, or verification actions, while Manual Upload and Configuration Set management remain available.

**Blocked by:** None (can start immediately).

**Status:** ready-for-human

**Parent specification:** `../spec.md`

**Testing seam:** Exercise the new Wails-free File Manager Integration module as the single interface used by Desktop for state and action availability; use the Linux adapter as the first concrete slice.

**Demo path:** Launch Upit Desktop on Linux with normal, Setup, and Repair Configuration Set states; open File Manager Integration and observe the same non-blocking unsupported state with no platform actions.

- [x] A deep File Manager Integration module exposes state and available actions without leaking platform commands or package details to Desktop.
- [x] The user-visible state model supports `Registered`, `Needs Repair`, `Reinstall Upit`, and `Not supported on Linux`.
- [x] Desktop adds File Manager Integration as a dedicated fifth sidebar area.
- [x] The area remains reachable while the Configuration Set requires Setup or Repair.
- [x] Linux reports `Not supported on Linux` and explains that this feature provides no Linux file-manager adapter.
- [x] Linux exposes no registration, Repair, operating-system settings, removal, or file-manager verification action.
- [x] File Manager Integration state remains independent from Configuration Set readiness and does not block Manual Upload or configuration management.
- [x] The area has accessible navigation, headings, status announcements, and action semantics consistent with the existing Desktop surface.
- [x] Tests cover the module interface, Linux adapter, Desktop rendering, Setup/Repair accessibility, and failure isolation without testing pass-through wiring.
