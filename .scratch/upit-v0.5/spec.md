# Upit v0.5 — Desktop Configuration and Manual Upload

Status: ready-for-agent

## Problem Statement

Upit v0.4 is headless-first and CLI-only. It can upload one file, inspect and validate the Configuration Set, and mutate three Global Configuration fields, but users must still create and maintain Uploader and Shortener definitions by editing strict JSON documents manually. The product has no graphical way to repair malformed configuration, manage named definitions, or perform a Manual Upload with visible progress and cancellation.

A desktop frontend must not fork the existing application rules. Configuration versions, strict validation, safe publication, upload behavior, Shortener fallback, clipboard warnings, cancellation, redaction, and the fixed Configuration Set location already form one production contract. Duplicating those rules in Vue or Wails would create drift and could expose credentials or corrupt configuration.

The release also crosses difficult lifecycle and platform boundaries: Wails 3 remains beta, Linux desktop support depends on modern GTK/WebKit, definition renames can cross document references, stale external edits can be overwritten, invalid documents must remain repairable, and credential-bearing values must be editable without being displayed casually. v0.5 must provide a useful desktop product while keeping the CLI independent, script-compatible, and free of Wails startup or WebView runtime requirements.

## Solution

Add a separately launched `upit-desktop` application while retaining the existing `upit` CLI. The desktop application uses pinned Wails `v3.0.0-beta.26`, Vue 3, TypeScript, Vite, and locally owned shadcn-vue component source. The repository remains one Go module and raises its minimum Go version to 1.25. Wails services are thin adapters over the existing Wails-free application module, which remains the authority for Configuration Set loading, validation, persistence, upload execution, cancellation, progress, Shortener behavior, clipboard behavior, and redacted failures.

Present one English-language, system-themed window with four primary areas: Manual Upload, Global Configuration, Uploaders, and Shorteners. The application runs as a single instance, focuses the existing window on subsequent launches, has no system tray or resident worker, and exits completely after resolving active work when its window closes.

Provide first-run Setup when the complete Configuration Set is absent, structured editing for valid documents, and explicit Raw Repair for partial, malformed, or semantically invalid documents. Global Configuration exposes only its existing three user-managed fields. Uploader and Shortener editors support create, edit, rename, and delete, with valid-only explicit saves, deterministic canonical JSON, stale-save detection, credential masking, and no cross-document transaction journal. Referenced definitions must be changed or cleared in Global Configuration before rename or deletion.

Provide Manual Upload for exactly one regular file selected through a native picker or drag-and-drop. It exposes the same Uploader, Shortener, clipboard, and timeout choices as the CLI, reports preparation and upload progress, supports cancellation, continues while the user navigates within the desktop application, and displays structured success, warning, cancellation, and failure states. It does not add batch upload, upload history, automatic retry, or a second upload protocol.

Build native runnable desktop artifacts for Windows, macOS, and modern Linux systems using GTK4 and WebKitGTK 6.0. Installer creation, signing, notarization, auto-update, shell integration, and other distribution work remain deferred to v1.0.

## User Stories

1. As an existing CLI user, I want `upit` to retain its current commands and behavior, so that v0.5 does not break scripts or headless workflows.
2. As a desktop user, I want a separate `upit-desktop` executable, so that I can opt into a graphical interface without changing the CLI binary's runtime model.
3. As a headless user, I want the CLI to avoid Wails startup, WebView startup, GUI initialization, and desktop runtime dependencies, so that servers and containers remain supported.
4. As a user, I want the CLI and desktop application to use the same application rules, so that configuration and upload behavior do not depend on the chosen frontend.
5. As a user, I want one desktop window with clear navigation, so that Manual Upload and Configuration Set management remain easy to find.
6. As a user, I want Manual Upload, Global Configuration, Uploaders, and Shorteners shown as separate primary areas, so that unrelated tasks do not share one oversized form.
7. As a user, I want the interface to use English terminology matching the CLI and documentation, so that diagnostics and controls remain consistent.
8. As a user, I want the desktop theme to follow the operating-system light or dark preference, so that the application fits my environment without adding another setting.
9. As a user, I want only one desktop instance, so that two Upit windows do not race to edit the same Configuration Set.
10. As a user who launches Upit Desktop again, I want the existing window restored and focused, so that the second launch does not silently fail or open a competing editor.
11. As a user, I want no system tray behavior, so that closing the desktop application has an unambiguous meaning.
12. As a resource-conscious user, I want the desktop process to exit completely when closed, so that Upit does not become a resident worker.
13. As a first-time user, I want Setup to open when the complete Configuration Set is absent, so that I do not need to copy example JSON manually.
14. As a first-time user, I want Setup to remain an in-memory draft until I finish it, so that incomplete values are not written as runtime configuration.
15. As a first-time user, I want Setup to collect the three Global Configuration fields and one complete Uploader, so that the created Configuration Set can perform an upload immediately.
16. As a first-time user, I want Shortener setup deferred to the normal editor, so that the required first-run path remains bounded.
17. As a first-time user, I want Setup to validate the complete draft before creation, so that Finish never intentionally publishes an invalid Configuration Set.
18. As a first-time user, I want Setup to avoid invented endpoints or credentials, so that no fake configuration appears usable.
19. As a user with a valid Configuration Set, I want the normal desktop areas to open directly, so that returning use does not repeat onboarding.
20. As a user with a partial Configuration Set, I want Repair rather than Setup, so that existing documents are preserved instead of treated as disposable.
21. As a user with malformed JSON, I want the original bytes preserved, so that Upit never truncates or silently resets credentials while trying to help.
22. As a user with semantically invalid configuration, I want exact diagnostics and Repair access, so that I can correct the document that prevents normal operation.
23. As a user in Repair, I want Manual Upload disabled, so that an invalid Configuration Set cannot be used accidentally.
24. As a user in Repair, I want normal structured editing restored automatically after the repaired document becomes valid, so that raw editing is temporary.
25. As a user, I want the fixed `~/.config/upit/` Configuration Set location preserved on every operating system, so that CLI and desktop always operate on the same documents.
26. As a user, I want Global Configuration to expose the default Uploader, optional default Shortener, and clipboard-copying default, so that all current global choices are manageable.
27. As a user, I want no new Global Configuration fields in v0.5, so that desktop delivery does not expand unrelated runtime policy.
28. As a user, I want the selected default Uploader displayed by exact name, so that case-sensitive references remain clear.
29. As a user, I want the optional default Shortener displayed distinctly from no Shortener, so that absence cannot be confused with a definition name.
30. As a user, I want clipboard copying represented as an explicit enabled or disabled choice, so that its default is visible before upload.
31. As a user, I want Global Configuration changes held as dirty edits until Save, so that selecting a control does not write immediately.
32. As a user, I want Save disabled while Global Configuration is invalid, so that a dangling reference cannot be published.
33. As a user, I want Global Configuration Save to validate cross-document references, so that defaults always select existing valid definitions.
34. As a user, I want changing the default Uploader to affect later uploads only, so that an active upload keeps the configuration snapshot with which it started.
35. As a user, I want changing or clearing the default Shortener to affect later uploads only, so that active post-processing remains deterministic.
36. As a user, I want changing the clipboard default to affect later uploads only, so that an active upload does not change behavior mid-flight.
37. As a user, I want a dirty-state prompt before leaving unsaved Global Configuration edits, so that navigation cannot discard them silently.
38. As a user, I want Save, Discard, and Cancel choices when resolving dirty edits, so that I control whether navigation or close proceeds.
39. As a user, I want Uploaders listed by their exact configured names, so that I can inspect every available destination.
40. As a user, I want Shorteners listed by their exact configured names, so that I can inspect every optional post-processing destination.
41. As a user, I want deterministic list ordering, so that the editor does not jump between launches.
42. As a user, I want the active default Uploader indicated in the Uploader list, so that its importance is visible.
43. As a user, I want the active default Shortener indicated in the Shortener list, so that its importance is visible.
44. As a user, I want to create a new Uploader as an unsaved structured draft, so that incomplete values remain local until valid.
45. As a user, I want to create a new Shortener as an unsaved structured draft, so that incomplete values remain local until valid.
46. As a user, I want new definition names validated with the existing exact name rules, so that desktop-created documents match runtime expectations.
47. As a user, I want duplicate definition names rejected, so that creating a definition cannot overwrite another one implicitly.
48. As a user, I want to edit an Uploader through sections corresponding to request, Request Body Mode, and Response Extractor behavior, so that complex configuration remains understandable.
49. As a user, I want to edit a Shortener through sections corresponding to request data and response extraction, so that its narrower protocol remains clear.
50. As a user, I want fields irrelevant to the selected Request Body Mode hidden or disabled, so that incompatible combinations are not presented as valid.
51. As a user, I want Response Extractor controls adapt to the selected extractor type, so that only meaningful fields are editable.
52. As a user, I want dynamic headers, query values, and form fields edited as key-value rows, so that map-shaped data does not require whole-document JSON editing.
53. As a user, I want duplicate or invalid keys diagnosed inline, so that conflicts are found before Save.
54. As a user, I want arbitrary nested request `data` edited in a field-scoped JSON editor, so that current JSON protocol capability is not narrowed.
55. As a user, I want the field-scoped JSON editor to report syntax and placeholder errors, so that invalid nested data cannot be published.
56. As a user, I want to rename an unreferenced Uploader explicitly, so that an accidental name-field edit cannot change identity silently.
57. As a user, I want to rename an unreferenced Shortener explicitly, so that identity changes are deliberate.
58. As a user, I want rename blocked when Global Configuration references the definition, so that no cross-document dangling reference is created.
59. As a user, I want rename to require a valid, unused new name, so that it cannot collide or normalize unexpectedly.
60. As a user, I want deletion to require confirmation, so that destructive actions are not one-click mistakes.
61. As a user, I want deletion blocked when Global Configuration references the definition, so that every published Configuration Set remains valid.
62. As a user, I want deletion of the final Uploader blocked, so that the required Uploader document never becomes empty.
63. As a user, I want deletion of the final unreferenced Shortener to remove the optional Shortener document, so that an invalid empty document is not retained.
64. As a user, I want focus to move predictably after deletion, so that the editor does not display a removed definition.
65. As a user, I want no Duplicate action in v0.5, so that credentials are not copied through a feature outside the agreed CRUD surface.
66. As a user, I want no Import or Export action in v0.5, so that merge, collision, and secret-disclosure policy remain outside this release.
67. As a user, I want inline validation while editing, so that errors appear near the field that caused them.
68. As a user, I want strict document and cross-document validation on Save, so that inline convenience checks cannot replace runtime authority.
69. As a maintainer, I want Go validation to remain authoritative, so that Vue does not become a second implementation of Configuration Set semantics.
70. As a user, I want one deterministic diagnostic order, so that repeated attempts show the same first repair action.
71. As a user, I want diagnostics to include the document and precise configuration path, so that invalid values can be located quickly.
72. As a user, I want malformed JSON diagnostics to include line and column where available, so that Raw Repair remains actionable.
73. As a security-conscious user, I want diagnostics to preserve existing redaction, so that configuration assistance never echoes credentials.
74. As a user, I want configured header, query, field, and data values masked by default, so that opening an editor does not expose them casually.
75. As a user, I want revealing, editing, or copying a masked value to require an explicit action, so that disclosure is deliberate.
76. As a user, I want secret reveal state limited to the current session, so that it is not restored after restart.
77. As a user, I want sensitive drafts excluded from localStorage and persistent frontend logs, so that the UI does not create a second credential store.
78. As a user, I want Raw Repair locked by default, so that opening a broken document does not immediately expose its full contents.
79. As a user, I want an explicit warning before unlocking Raw Repair, so that I understand the document may contain credentials.
80. As a user, I want Raw Repair to show the complete original document after unlock, so that syntax and structural failures remain repairable.
81. As a user, I want Raw Repair to lock again when I leave Repair or close the application, so that full document contents do not remain exposed.
82. As a user, I want Raw Repair Save blocked until strict validation succeeds, so that repair cannot publish another invalid document.
83. As a user, I want structured saves to produce canonical JSON with four-space indentation, stable key ordering, and a final newline, so that file bytes are deterministic.
84. As a user, I accept that structured Save normalizes whitespace and key order, so that the editor does not require a concrete-syntax-tree patcher.
85. As a user, I want existing Configuration Set versions preserved, so that v0.5 does not force a format migration.
86. As a user, I want no automatic migration or compatibility fallback, so that unsupported versions remain explicit repair work.
87. As a user, I want credential-bearing files created with private Unix permissions, so that Setup does not weaken existing filesystem protection.
88. As a user, I want linked or supported reparse-point mutation targets refused, so that publication does not silently replace a link itself.
89. As a user, I want complete staged content written, synchronized, and closed before publication, so that readers never observe partial JSON.
90. As a user, I want failed pre-publication writes to preserve the original document and remove ordinary staged files, so that errors do not cause data loss or clutter.
91. As a user, I want no backup or lock files, so that configuration management does not retain extra credential copies or stale coordination state.
92. As a user, I want no automatic multi-file transaction journal, so that definition editing does not add a hidden recovery protocol.
93. As a user, I want Save rejected when a relevant document changed since load, so that desktop edits cannot overwrite CLI or external-editor changes silently.
94. As a user, I want stale-save errors to require Refresh or reload rather than automatic merge, so that conflict resolution remains explicit.
95. As a user with clean state, I want external changes detected when the window regains focus, so that the displayed Configuration Set can refresh promptly.
96. As a user, I want a manual Refresh action, so that I can request a current snapshot without restarting.
97. As a user, I want freshness checked again before Save and Manual Upload, so that critical actions do not trust an old snapshot.
98. As a user with dirty edits, I want external changes reported as a conflict rather than triggering automatic reload, so that my draft is not discarded.
99. As a resource-conscious user, I want no continuous filesystem watcher, so that focus, Refresh, Save, and upload preflight remain the only refresh triggers.
100. As a user, I want Manual Upload to accept exactly one regular file, so that desktop behavior preserves the existing application contract.
101. As a user, I want a native file picker, so that I can select a file using platform conventions.
102. As a user, I want drag-and-drop, so that I can start from the system file manager.
103. As a user, I want multi-file drops rejected with an inline explanation, so that Upit never silently uploads only the first file.
104. As a user, I want directory and non-regular-file selections rejected, so that unsupported input fails before network activity.
105. As a user, I want the configured default Uploader preselected, so that the common upload path needs no extra choice.
106. As a user, I want to select another Uploader for one Manual Upload, so that the Global Configuration remains unchanged.
107. As a user, I want the configured default Shortener preselected, so that desktop and CLI defaults agree.
108. As a user, I want to choose another Shortener or disable shortening for one Manual Upload, so that temporary behavior does not mutate Global Configuration.
109. As a user, I want clipboard behavior inherited from Global Configuration by default, so that the same post-action policy applies across frontends.
110. As a user, I want to enable or disable clipboard copying for one Manual Upload, so that a temporary override does not mutate Global Configuration.
111. As a user, I want an optional whole-upload timeout equivalent to the CLI timeout, so that long-running operations can be bounded.
112. As a user, I want no timeout by default, so that desktop behavior matches the existing upload contract.
113. As a user, I want Manual Upload blocked while any Configuration Set edit is dirty, so that execution never uses an ambiguous mix of saved and unsaved state.
114. As a user, I want Manual Upload to reload and validate the current on-disk Configuration Set before starting, so that it uses the same authority as the CLI.
115. As a user, I want the active upload to retain its starting configuration snapshot, so that later edits do not alter an in-flight request or post-processing step.
116. As a user, I want only one Manual Upload active at a time, so that results, progress, cancellation, and clipboard behavior remain unambiguous.
117. As a user, I want a preparation phase while Upit validates input and configuration, so that work before network transfer is visible.
118. As a user, I want byte progress when file bytes are measurable, so that large uploads provide useful feedback.
119. As a user, I want phase labels for uploading, response processing, shortening, and clipboard work, so that non-byte work is still understandable.
120. As a user, I want progress events to exclude credentials and request values, so that UI telemetry does not weaken redaction.
121. As a user, I want to cancel an active Manual Upload, so that I can stop unwanted or stalled work.
122. As a user, I want cancellation to propagate through upload and Shortener operations, so that the application does not continue hidden network work.
123. As a user, I want a canceled operation shown as canceled rather than failed, so that intentional interruption is recognizable.
124. As a user, I want my selected file and options retained after cancellation, so that I can adjust and retry deliberately.
125. As a user, I want an active upload to continue when I navigate to another desktop area, so that long uploads do not block configuration inspection.
126. As a user, I want global progress and Cancel access while viewing another area, so that the active operation remains visible and controllable.
127. As a user, I want configuration edits made during an active upload to apply only to later uploads, so that current behavior remains deterministic.
128. As a user, I want successful Manual Upload to show the Final URL prominently, so that the shareable result is clear.
129. As a user, I want the Original URL shown only when it differs from the Final URL, so that Shortener behavior is visible without duplicate noise.
130. As a user, I want Shortener fallback and clipboard failures shown as warnings without turning upload success into failure, so that desktop behavior matches the CLI.
131. As a user, I want a Copy action that copies the Final URL, so that I can copy manually regardless of automatic clipboard policy.
132. As a user, I want a structured failure state containing stage, message, and HTTP status when available, so that troubleshooting matches CLI diagnostics.
133. As a user, I want the selected file and options retained after failure, so that I can correct configuration or retry without starting over.
134. As a user, I want Retry to be explicit, so that Upit never repeats a potentially non-idempotent upload automatically.
135. As a user, I want no upload queue or concurrent batch, so that one-file behavior remains the only desktop execution model.
136. As a user, I want no upload history, so that v0.5 does not add persistence or retention policy for local paths and URLs.
137. As a user closing the application during an upload, I want a confirmation before cancellation, so that closing the window does not stop work accidentally.
138. As a user who confirms close during an upload, I want Upit to cancel and wait for the operation to stop before exiting, so that no background work survives the window.
139. As a user closing with dirty edits, I want Save, Discard, and Cancel choices, so that unsaved work is resolved explicitly.
140. As a user closing during publication, I want the application to wait for the publication result, so that it does not abandon a write mid-operation.
141. As a Windows user, I want a native runnable desktop artifact using the system WebView2 runtime, so that Upit does not bundle Chromium.
142. As a macOS user, I want a native runnable desktop artifact using system WebKit, so that the application follows the platform webview model.
143. As a modern Linux desktop user, I want a native runnable artifact using GTK4 and WebKitGTK 6.0, so that Upit follows the supported Wails 3 stack.
144. As a user on an older Linux distribution, I want the headless CLI to remain available, so that lack of the modern desktop webview stack does not remove core functionality.
145. As a keyboard user, I want every interactive operation reachable and understandable without a mouse, so that the desktop application meets baseline accessibility.
146. As a screen-reader user, I want controls, dialogs, errors, progress, and state changes labeled semantically, so that visual presentation is not the only source of meaning.
147. As a user, I want destructive dialogs to receive focus and return focus predictably, so that confirmation remains safe and accessible.
148. As a maintainer, I want Wails pinned exactly, so that beta updates cannot change the application implicitly.
149. As a maintainer, I want Wails services to remain thin adapters, so that desktop framework churn does not spread into application behavior.
150. As a maintainer, I want the Wails-free application module to be the primary permanent test seam, so that configuration and upload behavior are deterministic without a WebView.
151. As a maintainer, I want existing CLI command-running tests preserved, so that desktop work cannot regress headless contracts.
152. As a maintainer, I want frontend code to use locally owned shadcn-vue components only as needed, so that unused component source is not added speculatively.
153. As a maintainer, I want no router, global state library, or external form framework unless the implemented interface proves one necessary, so that the frontend remains small.
154. As a maintainer, I want actual desktop smoke verification in addition to tests, so that bindings, native dialogs, drag-and-drop, progress, cancellation, and visible state are exercised on the running application.
155. As a maintainer, I want the v0.5 architecture, README, build instructions, and Configuration Set documentation updated with delivered behavior, so that users are not left with the old CLI-only roadmap.

## Implementation Decisions

- v0.5 is named **Desktop Configuration and Manual Upload**.
- The existing `upit` executable remains the primary CLI/headless interface. A separate `upit-desktop` executable owns Wails and WebView startup.
- The repository remains one Go module and raises its minimum Go version from 1.23 to 1.25.
- The desktop framework is pinned to Wails `v3.0.0-beta.26`. Stable Wails 3 is not a prerequisite. Any upgrade requires changelog review, an exact version change, complete checks, and native desktop smoke.
- The frontend uses the official Wails Vue template with Vue 3, TypeScript, and Vite.
- shadcn-vue supplies selected component source copied into the repository. Reka UI primitives provide dialog, focus, keyboard, and ARIA behavior. Only components used by the implemented interface are added.
- Tailwind and the minimal shadcn-vue utility dependencies are used. No router, Pinia/Vuex store, external form suite, general design system, or alternate component registry is added without a demonstrated need.
- The primary application module remains Wails-free. It owns Configuration Set state, strict validation, canonical serialization, safe publication, stale-snapshot checks, upload orchestration, progress, cancellation, Shortener behavior, clipboard behavior, redacted failures, and startup-mode classification.
- Wails services are thin adapters. They validate frontend trust-boundary inputs, call application-module operations, translate results to binding data, and emit desktop events. They do not reimplement domain validation, filesystem policy, upload selection, or redaction.
- The desktop uses one main window with sidebar navigation for Manual Upload, Global Configuration, Uploaders, and Shorteners. Setup and Repair replace normal content when startup state requires them.
- The UI is English-only in v0.5. It follows the system light/dark preference and adds no persisted theme setting.
- Wails single-instance support prevents competing desktop processes. A second launch restores and focuses the existing window. v0.5 does not process second-launch file arguments or add OS integration.
- Window close is intercepted when an upload, publication, or dirty edit is active. Confirmed upload close cancels and waits; dirty edits offer Save, Discard, and Cancel; publication completes before exit.
- There is no system tray, close-to-tray behavior, daemon, resident worker, or process remaining after the desktop window closes.
- The Configuration Set remains fixed at `~/.config/upit/` on Windows, macOS, and Linux.
- Global Configuration remains version 2, the Uploader document remains version 2, and the Shortener document remains version 1. v0.5 adds no migration, compatibility fallback, profile format, provider format, or schema version.
- Startup has three states. A valid Configuration Set enters the normal application. Complete absence enters Setup. Any partial, malformed, unsupported-version, permission-invalid, or semantically invalid state enters Repair without modifying existing bytes.
- Setup is an in-memory draft that collects the three existing Global Configuration fields and one Uploader. Shortener setup is deferred to the normal editor.
- Setup publishes no files until the draft is complete and valid. It creates the fixed directory and required documents only after explicit Finish.
- Setup never invents endpoint URLs, credentials, extractor expressions, or a fake Uploader. The first Uploader is completed through the same structured controls used by the normal editor.
- Partial existing state is never treated as fresh Setup. Repair preserves every existing document and reports missing required documents explicitly.
- Global Configuration exposes only default Uploader, optional default Shortener, and default clipboard copying.
- Uploader and Shortener lists are deterministic and identify selected defaults.
- Definition creation opens one unsaved structured draft. Save remains unavailable until the draft and owning document are valid.
- Definition names remain exact, case-sensitive, printable Unicode with no leading or trailing Unicode whitespace. Names are never trimmed, normalized, or case-folded.
- Uploader structured editing follows request, Request Body Mode, and response sections. Fields shown for a Request Body Mode match the existing runtime protocol exactly.
- Shortener structured editing follows request and JSON response-extraction sections. v0.5 does not add Shortener extractor types or protocols.
- Headers, query values, and form fields use dynamic key-value controls with immediate duplicate-key and basic field diagnostics.
- Arbitrary nested request `data` uses a field-scoped JSON editor inside the structured form. This editor validates JSON syntax and delegates semantic and placeholder validation to Go.
- Valid documents do not expose a whole-document raw editing mode. Raw Repair is available only when an existing document cannot be represented safely by the structured editor.
- Go strict decoding and semantic validation remain authoritative. Frontend checks improve immediacy but cannot make an invalid Go document publishable.
- Inline diagnostics use the authoritative path and message returned by the application module where possible. Save performs complete document and cross-document validation.
- Only valid documents are published. Invalid drafts remain in memory and do not create draft sidecar files.
- Editing uses explicit Save. There is no autosave, save-on-navigation, or implicit save-on-close.
- Navigation, item switching, Refresh, and close resolve dirty state explicitly. No dirty draft is silently discarded or persisted outside the Configuration Set.
- Structured Save serializes the complete owning document as deterministic canonical JSON with four-space indentation, stable key ordering, and a final newline. Lexical whitespace and previous key order are not preserved.
- Configured header, query, field, and body values are treated as sensitive. They are masked by default and require explicit reveal, edit, or copy actions.
- Sensitive values and drafts are not written to localStorage, frontend persistence, analytics, telemetry, or persistent UI logs.
- Raw Repair starts locked. Explicit unlock shows the full document with a credential warning for the current session only. Leaving Repair or closing the application locks it again.
- Rename is a separate explicit operation. A definition referenced by Global Configuration cannot be renamed until the reference is changed or cleared.
- Delete always asks for confirmation. A referenced definition cannot be deleted. The final Uploader cannot be deleted. Deleting the final unreferenced Shortener removes the optional Shortener document.
- v0.5 adds no Duplicate, Import, Export, provider template, shared registry, or cloud synchronization action.
- Edits to an existing Configuration Set remain single-document publications. v0.5 does not add a cross-document transaction journal, lock file, backup file, or automatic merge.
- Before Save, the application compares relevant document snapshots with disk. Any external change rejects the stale Save and requires Refresh or reload.
- Clean state checks for external changes when the window gains focus, when the user requests Refresh, before Save, and before Manual Upload. There is no continuous file watcher.
- Dirty state is never auto-reloaded. External changes produce a conflict state while preserving the in-memory draft until the user copies, discards, or otherwise resolves it.
- Existing safe publication is generalized to every writable Configuration Set document: uniquely staged same-directory content, complete write, synchronization, close, permission handling, link-target refusal, and native publication without remove-before-replace.
- Pre-publication failures leave the original document unchanged and clean ordinary staged files. Ambiguous Windows publication retains recoverable staged content and reports its path without exposing contents.
- Existing POSIX permission checks remain authoritative. Setup creates credential-bearing documents with private permissions. Windows does not simulate Unix permission bits.
- Reading and Repair may follow ordinary filesystem behavior. Mutations refuse symlinked or supported reparse-point document targets during point-in-time preflight.
- The filesystem threat model remains a user-owned Configuration Set directory without adversarial same-user path racing. v0.5 does not add handle-relative filesystem hardening.
- First-run creation may publish multiple previously absent documents, but it never replaces a pre-existing valid Configuration Set. A creation failure is reported as incomplete Setup/Repair state; no stronger cross-file crash-atomicity claim is made.
- Manual Upload accepts one path to one regular file. Native picker selection is single-file. Drag-and-drop validates that exactly one regular file was dropped and rejects the complete drop otherwise.
- Manual Upload preselects Global Configuration defaults and allows per-upload Uploader, Shortener, no-shorten, clipboard, no-clipboard, and optional timeout overrides. It adds no JSON-output choice because results are rendered in the desktop interface.
- Timeout defaults to none and uses the same whole-operation meaning as the CLI.
- Manual Upload cannot start while any Configuration Set edit is dirty or while startup state is Setup or Repair.
- Immediately before upload, the application checks freshness, reloads, and strictly validates the on-disk Configuration Set. The upload then uses an immutable selection snapshot for its lifetime.
- Only one Manual Upload may be active. Navigation remains available, and a global status surface keeps progress and Cancel accessible outside the Manual Upload area.
- Progress has explicit phases. File validation and transformation preparation may be indeterminate; transfer reports processed file bytes against the selected file size where meaningful; response extraction, Shortener work, and clipboard work use phase state rather than invented percentages.
- Progress is advisory and never changes upload success semantics. It carries no configured values, endpoint URL, request values, response body, or credentials.
- Wails binding cancellation propagates to the application context. Cancellation during upload or Shortener work remains cancellation rather than fallback success.
- Success presents Final URL, Original URL only when different, and warnings in their existing order. Automatic clipboard failure and Shortener fallback remain nonfatal.
- The Copy action always copies Final URL and reports copy failure without invalidating the completed upload.
- Failure presents the existing structured stage, sanitized message, and HTTP status when available. Inputs remain selected for explicit Retry.
- Retry invokes a new upload only after a fresh Configuration Set check. There is no automatic retry because configured endpoints are not assumed idempotent.
- Configuration edits saved during an active upload affect later uploads only. The active operation continues using its starting snapshot.
- Desktop builds target Windows and macOS native webviews and modern Linux with GTK4 and WebKitGTK 6.0. The legacy GTK3 build tag and older Linux desktop stack are not v0.5 release targets.
- v0.5 produces native runnable desktop artifacts but no installer, signing, notarization, auto-update, package-manager manifest, Explorer/Finder integration, launcher plugin, shell integration, or release automation.
- The architecture roadmap, build documentation, README, and relevant configuration documentation are updated when implementation is delivered. Historical specifications remain archival.

## Testing Decisions

- The primary permanent seam is the Wails-free application module. Tests exercise its exported behavior with temporary homes, real temporary Configuration Set documents, local HTTP endpoints, deterministic clipboard adapters, and captured progress/cancellation observations.
- The application-module seam covers startup classification, Setup, valid load, Repair, snapshots, validation, CRUD, canonical serialization, safe publication, stale-save rejection, upload selection, progress, cancellation, Shortener behavior, clipboard behavior, and redacted failures.
- Tests use the same application interface that the Wails adapter calls. They do not create a parallel desktop-only validation or persistence implementation.
- Existing CLI command-running tests remain intact and continue to prove upload parsing, Configuration Management commands, stdout/stderr, JSON output, exit codes, cancellation, Shortener fallback, clipboard warnings, and current help contracts.
- CLI regression tests prove adding Wails and Go 1.25 does not alter `upit` behavior or introduce Wails/WebView startup into the headless executable.
- Setup tests cover complete absence, an existing empty parent location, one valid first Uploader, optional Shortener omission, invalid drafts, publication failure, private credential-file permissions, and a subsequent normal startup from the created Configuration Set.
- Startup-state tests cover valid, completely absent, partial, malformed, unsupported-version, permission-invalid, linked-target, and semantically invalid Configuration Sets without rewriting existing bytes.
- Repair tests cover locked initial state, explicit unlock, exact original bytes, malformed JSON line/column diagnostics, semantic path diagnostics, valid-only Save, relock on navigation, and transition back to structured editing.
- Security tests prove configuration values are redacted from diagnostics, progress events, ordinary UI state payloads, logs, and failure results.
- Secret-handling tests focus on observable masking and explicit reveal behavior. They do not assert CSS classes or copied shadcn-vue source text.
- Global Configuration tests cover all three fields, exact references, dirty state, validation, no-op Save, canonical bytes, and later-upload-only effects.
- Uploader and Shortener CRUD tests cover valid creation, invalid drafts, name rules, collisions, edit, unreferenced rename, referenced rename rejection, confirmation-required deletion at the application command level, referenced deletion rejection, final-Uploader rejection, and final-Shortener document removal.
- Nested `data` tests cover valid objects, malformed JSON, nested exact-one-placeholder semantics, sensitive nested values, and canonical output without narrowing the existing protocol.
- Validation tests reuse existing strict-decoder cases for duplicate keys, unknown fields, explicit null, deterministic definition ordering, Request Body Mode rules, Response Extractor rules, versions, cross-document references, and Unix permissions.
- Persistence tests cover explicit Save only, deterministic canonical output, preservation of unrelated definitions, same-directory staged replacement, staged-file cleanup, no remove-before-replace, mode handling, link refusal, and Windows ambiguous-failure recovery reporting.
- Stale-save tests change Global Configuration, Uploader, and Shortener documents externally after load and prove Save is rejected without overwriting either disk or the in-memory draft.
- Refresh tests cover clean focus refresh, manual Refresh, dirty conflict preservation, Save-time recheck, and Manual Upload preflight recheck. No test expects a background file watcher.
- Concurrency tests prove one active Manual Upload, configuration saves affect only later uploads, and complete-file visibility remains intact under concurrent external writers. They do not assert a deterministic winner for unrelated external writes.
- Manual Upload tests use real local HTTP servers and cover picker-equivalent paths, one-file drop validation at the adapter/UI surface, non-regular files, every Uploader Request Body Mode, every Uploader Response Extractor, named and default selection, Shortener override/no-shorten, clipboard overrides, timeout, warnings, and structured failures.
- Progress tests cover ordered phases, monotonic byte counts within one measurable phase, correct total file size, no invented percentage for indeterminate phases, cancellation, and absence of configured or response values in emitted payloads.
- Cancellation tests cover preparation, request streaming, response wait, Shortener work, navigation while active, confirmed close, and release of file/network resources.
- Success tests cover equal and different Original URL/Final URL values, Shortener fallback, automatic clipboard success/failure, manual Copy behavior, warning order, and no history persistence.
- Failure tests cover validation, request, network, response, parse, timeout, and cancellation states; exact stage/message/status rendering is checked through returned application data rather than source-text assertions.
- Retry tests prove there is no automatic request repetition, inputs remain selected after failure/cancellation, and explicit Retry performs a fresh Configuration Set check before one new attempt.
- The Wails adapter receives focused tests only for trust-boundary validation or behavior that cannot be observed through the application seam, such as context cancellation propagation and event payload shape. Pass-through forwarding, generated bindings, call counts, and mock echoes are not permanent test targets.
- Frontend checks include Vue TypeScript typechecking and the production Vite build. No frontend test framework is added solely to test static rendering, copied shadcn-vue markup, or Wails-generated bindings.
- Any nontrivial frontend-only state transition not observable through the application seam must receive one small behavior check at the highest available frontend state seam. Do not add a broad component suite preemptively.
- Accessibility verification covers keyboard-only navigation, visible focus, dialog focus trapping, labels, validation associations, non-color-only states, progress announcements, and focus restoration after dialogs.
- Completion requires running the actual desktop application, not only tests. The smoke exercises first-run Setup, one Global Configuration Save, one Uploader edit, one Shortener create/delete path, one Manual Upload through a local endpoint, progress, cancellation, navigation during upload, success result Copy, failure Retry, dirty close, and single-instance focus.
- Native Windows smoke is required for WebView2, single-instance behavior, file dialogs, drag-and-drop, clipboard, native replacement, and executable exit.
- Native Linux smoke is required on GTK4/WebKitGTK 6.0 for window startup, file dialogs, drag-and-drop, clipboard behavior when available, publication, and process exit.
- Native macOS runtime smoke is required before claiming a verified macOS release. If no macOS environment is available during this delivery, Darwin build verification is compile-only and the limitation is reported explicitly rather than inferred away.
- Full Go tests, Go vet, frontend typecheck/build, CLI builds, desktop native builds available on the host environments, and actual application smoke must pass before implementation is called complete.
- Tests verify observable contracts and invariants. They do not pin private helper names, source layout, incidental default object identity, CSS class lists, generated code text, or simple value forwarding.

## Out of Scope

- Replacing, deprecating, renaming, or wrapping the `upit` CLI.
- Combining CLI and desktop startup into one binary or one `desktop` subcommand.
- Wails 2, Electron, Tauri, bundled Chromium, or a browser-hosted web application.
- Waiting for Wails 3 stable before delivering v0.5.
- Multiple desktop windows or multiple writable desktop instances.
- System tray, close-to-tray, daemon, resident worker, background uploader, or process surviving window close.
- Multiple Configuration Set locations, profiles, workspace-local configuration, environment-selected configuration, or native per-platform config directories.
- Global Configuration fields beyond default Uploader, optional default Shortener, and clipboard copying.
- Configuration schema version changes, migrations, compatibility readers, automatic upgrade, or version fallback.
- New Uploader Request Body Modes, Response Extractors, Shortener protocols, provider adapters, or provider-specific fields.
- Duplicate, Import, Export, shared registry, cloud sync, provider templates, or downloadable definition catalogs.
- Autosave, save-on-navigation, draft sidecar files, backup files, lock files, automatic merge, or a cross-document transaction journal.
- Preserving original whitespace or key order after structured Save.
- A normal whole-document JSON editor for already-valid Configuration Set documents.
- Operating-system keychains, encrypted configuration, secret references, environment-variable substitution, or a new credential schema.
- Persisting revealed secrets, raw documents, dirty drafts, or upload inputs in localStorage, analytics, telemetry, or logs.
- Continuous filesystem watching or background polling for external changes.
- Adversarial same-user path-race hardening, handle-relative filesystem APIs, full metadata preservation, or stronger durability guarantees than the native publication primitives.
- Multi-file drag-and-drop, batch upload, upload queue, concurrent uploads, upload history, result history, retry queues, or scheduled uploads.
- Automatic upload retry, redirect following, or changing the current no-retry HTTP contract.
- Opening Final URL automatically, embedded browser preview, thumbnails, deletion URLs, or file-preview rendering.
- Endpoint reachability tests, synthetic requests, or a separate Test Connection action outside Manual Upload.
- Desktop JSON output, stdout result contracts, or using the GUI as an automation interface.
- UI localization, language packs, runtime language selection, or non-English diagnostics.
- A persisted Light/Dark/System theme selector or other desktop-only preference document.
- Legacy GTK3/WebKit2GTK 4.1 desktop builds for older Linux distributions.
- Installers, package-manager manifests, code signing, notarization, auto-update, release publishing, or delta updates.
- Windows Explorer, macOS Finder, shell, Flow Launcher, Wox, Raycast, or other launcher integration.
- Mobile applications, browser extensions, or remote configuration management.
- Rewriting historical v0.1 through v0.4 specifications or issue statuses.

## Further Notes

- **Manual Upload** is the canonical domain term recorded in `CONTEXT.md`. It means one interactive user-initiated upload with picker or drag-and-drop input, per-upload overrides, progress, cancellation, warnings, and Original URL/Final URL results.
- ADR 0010 records the exact Wails `v3.0.0-beta.26` pin, Go 1.25 single-module choice, and Wails-free CLI/application boundary.
- ADR 0011 records the decision to reject stale saves and avoid cross-document edit transactions by blocking rename or deletion of referenced definitions.
- ADR 0012 records Vue 3, TypeScript, Vite, and shadcn-vue as the desktop frontend stack.
- Primary-source Wails, ShareX, Postman, Visual Studio Code, Vue, and shadcn-vue research is retained with the v0.5 planning artifacts.
- Wails beta.26 provides first-party support for single-instance callbacks, native file dialogs, file-drop events, Go-to-frontend events, cancellable bindings, close hooks, focus events, and native desktop builds. One-file drop rejection and restore/focus on second launch remain small application guards.
- v0.1 through v0.4 specifications remain authoritative for upload protocols, output semantics, cancellation, Shortener fallback, clipboard warnings, strict diagnostics, schema/runtime authority, and native publication except where this specification explicitly changes the desktop-facing contract.
- The primary test seam was explicitly approved as the Wails-free application module. The Wails adapter and Vue frontend remain deliberately thin and are proven together by actual desktop smoke.
- Implementation must be decomposed into approved v0.5 tickets before code work begins. Repository changes remain uncommitted until the user explicitly authorizes a commit.
