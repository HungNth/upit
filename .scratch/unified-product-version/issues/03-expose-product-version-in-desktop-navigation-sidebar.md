# 03: Expose Product Version in Desktop navigation sidebar

**What to build:** Expose the running Product Version in the Desktop user interface by loading the version through the desktop backend service and displaying it statically at the base of the navigation sidebar across all application screens.

**Blocked by:** 01: Establish authoritative Product Version source and runtime provider

**Status:** resolved

- [x] Desktop backend service exposes the runtime Product Version string to the frontend application state.
- [x] The Desktop navigation sidebar renders the Product Version (e.g. `Upit v0.9.0`) below all navigation tabs.
- [x] The version element is visible across all navigation areas without interfering with form inputs or actions.
- [x] Desktop tests verify that the startup state and desktop interface correctly bind and display the Product Version.

