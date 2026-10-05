# 02: Brand Windows Desktop Binary and Installer

**Parent:** `.scratch/desktop-branding/spec.md`

**What to build:** Embed the official application icon directly into the Windows desktop binary via a committed COFF resource so that Taskbar, Start Menu shortcuts, and File Explorer context menu items show the branded icon, and brand the installer and uninstaller executables with the canonical icon.

**Blocked by:** 01: Establish Master Branding Assets and Reproducible Generation Pipeline

**Status:** ready-for-agent

- [ ] Committed COFF resource embeds the canonical icon at icon resource ID 1 (index 0).
- [ ] Compiling the Windows desktop binary embeds the icon resource without external build-time dependencies.
- [ ] Windows installer configuration defines installer and uninstaller icons using the canonical branding icon.
- [ ] Windows packaging validation asserts that the desktop binary, the built setup executable, and the generated uninstaller binary expose valid branded icon resources.
