---
status: accepted
---

# Use a stable Windows CLI launcher and version directories

On Windows, Upit exposes `%LOCALAPPDATA%\Programs\Upit\upit.exe` as the public CLI entry point and idempotently adds the installation root to the current user's `PATH`. The root executable is a stable launcher; the real CLI, Desktop, and File Manager helper remain version-aligned in one Active Payload at `versions\X.Y.Z`, while `HKCU\Software\Upit\PayloadPath` remains the authoritative active-payload pointer. Start Menu and File Manager Integration routes point directly to the selected payload rather than through additional stable launchers.

Updates extract into temporary staging and commit only after deterministic, side-effect-free verification of the required binaries and product version, CLI invocation through the stable launcher, Classic Verb registration, active metadata, Start Menu shortcut, and owned `PATH` entry. A different-version update stages side-by-side and may defer deletion of a locked inactive payload until a later install or uninstall. A same-version reinstall stages a replacement, asks the user to close every process executing from the Active Payload, renames the old payload to recovery storage, moves the staged payload to `versions\X.Y.Z`, and restores the prior state on any pre-commit failure.

The launcher has only an internal technical format version and is replaced only when a new format is required; replacement also requires the running launcher to be closed. The installer never force-terminates processes, and a silent install aborts when required files remain in use. Automatic rollback covers the installer transaction only, not failures observed after a successful commit. The installer owns only the stable-root `PATH` entry that it added and removes only that entry during uninstall.

An existing installation whose recorded `PayloadPath` uses `versions\<random>.tmp` receives a one-time transactional cutover to the new layout; the old layout is not supported after commit. The installer does not remove a payload-specific `PATH` entry that the user added manually. The user must remove that obsolete entry, which may otherwise shadow the new stable-root entry until it is deleted.

## Considered options

A flat installation directly under the root was rejected because Windows file locks can prevent safe in-place replacement and a partial overwrite cannot preserve an intact rollback payload. A separate deployment-instance directory below each product version was rejected in favor of one readable final directory per `X.Y.Z`; same-version replacement therefore requires process closure and a staged directory swap.
