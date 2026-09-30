# 06: Create the first Configuration Set

**What to build:** Make a completely absent Configuration Set an intentional first-run Setup experience. A user can create the fixed directory, all three Global Configuration choices, and one valid Uploader without manually copying JSON. The completed slice produces a Configuration Set immediately usable by both Desktop and CLI while making no false cross-document atomicity claim.

**Blocked by:** 02: Edit Global Configuration safely; 03: Create and edit Uploaders

**Status:** ready-for-agent

**Parent specification:** `../spec.md`

**Testing seam:** Use the Wails-free application module with nonexistent temporary homes to exercise Setup draft, validation, creation, cleanup/error classification, and subsequent normal load. The desktop smoke covers visible Setup and Finish; CLI validation proves the created documents are ordinary production configuration.

**Demo path:** Start Desktop with no Configuration Set, enter the three Global Configuration choices and one complete Uploader, Finish, relaunch into normal state, run `upit config validate`, then upload a file through a local endpoint using the created default.

- [x] Complete absence of the fixed Configuration Set enters Setup rather than a generic read failure.
- [x] Setup keeps Global Configuration and first-Uploader values in memory until explicit Finish and does not invent endpoint URLs, credentials, extractors, or fake defaults.
- [x] Setup reuses the structured Uploader editor contract and requires one complete valid Uploader plus valid Global Configuration references before Finish is enabled.
- [x] Finish creates the fixed Configuration Set directory and required documents only after complete validation; optional Shortener configuration is not required.
- [x] New credential-bearing documents receive the existing private Unix permission policy; Windows retains its existing non-simulated permission behavior.
- [x] A normal publication failure is surfaced honestly and leaves a diagnosable partial state for Repair; no backup, lock, hidden migration, or cross-document transaction claim is added.
- [x] A subsequent Desktop launch classifies the created documents as valid normal state, and `upit config validate` accepts them.
- [x] A local upload smoke through the existing CLI proves the newly created Configuration Set works outside the desktop process.

**Verification note:** Setup tests and the permanent CLI smoke assert normal startup classification, `config validate`, and a successful default CLI upload against a local endpoint. Native WebView interaction was not directly automated.
