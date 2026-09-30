# Primary-Source Research: Desktop Configuration Set Manager Precedents

## 1. Wails 3 Release Status and Platform Prerequisites (as of 2026-09-30)

### Release and Stability Status
- **Current Release Line**: Wails v3 is currently in public pre-release / beta. The latest release on GitHub as of 2026-09-30 is `v3.0.0-beta.26`, tagged and published on 2026-09-25. The release notes explicitly document: *"⚠️ Beta Warning: This is pre-release software. The API is stable, but you may still encounter issues before the final 3.0 release."* [Wails v3.0.0-beta.26 Release Notes](https://github.com/wailsapp/wails/releases/tag/v3.0.0-beta.26), [Wails GitHub Releases](https://api.github.com/repos/wailsapp/wails/releases?per_page=10).
- **Architecture and Programming Model**: Wails v3 shifts from the v2 declarative single-window model to a procedural multi-window and service application architecture. CLI installation targets `github.com/wailsapp/wails/v3/cmd/wails3@latest` (or explicit `@v3.0.0-beta.26`). [Wails v3 Quick-Start Installation](https://v3.wails.io/quick-start/installation), [What's New in Wails v3](https://v3.wails.io/whats-new).

### Official Development and Runtime Prerequisites
Source: [Wails v3 Quick-Start Installation Guide](https://v3.wails.io/quick-start/installation).

- **Core / Toolchain**:
  - Go 1.25 or later is required across all platforms.
  - Node.js / npm (optional for backend-only Go services, but standard/recommended when packaging HTML/JS/CSS frontend assets).
- **Windows**:
  - Development / Toolchain: C compiler (MinGW-w64 via MSYS2) in `PATH` when CGo compilation is required.
  - Runtime: Microsoft Edge WebView2 Runtime. Pre-installed by default on Windows 10 and 11. If absent, runtime installer from Microsoft or diagnostic check via `wails3 doctor`.
- **macOS**:
  - Development / Toolchain: Xcode Command Line Tools (`xcode-select --install`).
  - Runtime: System WebKit (shipped as part of macOS). Universal binaries (`arm64` + `x86_64`) supported natively via `wails3 tool lipo`.
- **Linux**:
  - Development / Toolchain: C toolchain (`gcc` or `build-essential`), `pkg-config`.
  - GUI & Web Runtime Libraries: Default stack requires **GTK4** (`libgtk-4-dev` / `gtk4-devel` / `gtk4`) and **WebKitGTK 6.0** (`libwebkitgtk-6.0-dev` / `webkitgtk6.0-devel` / `webkit-gtk:6`). Supported out-of-the-box on Ubuntu 24.04+, Debian 13+, Fedora 40+, Arch.
  - Legacy Linux Opt-in: For distributions lacking WebKitGTK 6.0 (such as Ubuntu 22.04 LTS or Debian 12), developers must install GTK3 and WebKit2GTK 4.1 (`libgtk-3-dev`, `libwebkit2gtk-4.1-dev`) and compile with `-tags gtk3`. WebKit2GTK 4.0 (Ubuntu 20.04) is unsupported.

---

## 2. Precedents for Custom Upload Destination and Configuration Editors

### Precedent A: ShareX Custom Uploader Settings
Sources:
- [ShareX Official Custom Uploader Documentation](https://getsharex.com/docs/custom-uploader.html)
- [ShareX Source: CustomUploaderSettingsViewModel.cs](https://raw.githubusercontent.com/ShareX/ShareX/develop/ShareX/Presentation/CustomUploaderSettings/CustomUploaderSettingsViewModel.cs)
- [ShareX Source: CustomUploaderSettingsModels.cs](https://raw.githubusercontent.com/ShareX/ShareX/develop/ShareX/Presentation/CustomUploaderSettings/CustomUploaderSettingsModels.cs)
- [ShareX Source: CustomUploaderItem.cs](https://raw.githubusercontent.com/ShareX/ShareX/develop/ShareX/ShareX.UploadersLib/CustomUploader/CustomUploaderItem.cs)

- **Editor Shape**: Two-pane master-detail presentation. The left pane provides an uploader list with search filtering, add, duplicate, remove, and multi-file `.sxcu` import/export. The right pane divides the active uploader definition into structured functional sections:
  1. *Overview*: Name, optional icon, and destination category assignments (Image, Text, File, URL Shortener, URL Sharing Service).
  2. *Request*: HTTP method (GET, POST, PUT, PATCH, DELETE), request URL, URL query parameter key-value table, and HTTP headers key-value table.
  3. *Body*: Request payload type selector (No body, Form data multipart, Form URL encoded, JSON, XML, Binary), form field arguments table, and file form parameter name.
  4. *Response*: Response URL extraction parser expression (JSONPath `json:`, XPath `xml:`, regex, header lookup `header:`, raw response `response`, redirection `responseurl`), thumbnail URL, deletion URL, and error message parser expressions.
  5. *Test*: Inline response simulation and live endpoint verification.
- **Validation Timing**: Form field bindings immediately update in-memory model fields on property change (`INotifyPropertyChanged`). Key-value tables (headers, query parameters, form fields) perform immediate duplicate-key checks; duplicate entries immediately flag validation errors (`HasDuplicateKey`). Export and test operations trigger blocking pre-flight checks (requiring non-empty `RequestURL` and an assigned destination category).
- **Secret Handling**: ShareX treats API keys, bearer tokens, and credentials as plain text within header and parameter string dictionaries, with optional macro substitution (`{base64:...}`). There is no encryption-at-rest or UI masking for custom uploader entries; `.sxcu` files export secrets in clear text unless the user manually removes them.
- **Create / Rename / Delete Behavior**:
  - *Create*: Appends a new default uploader instance (`CustomUploaderItem.Init()`) to `Uploaders` and selects it.
  - *Rename*: Editing `Name` immediately updates both the editor title and the master list entry. If `Name` is blank, ShareX falls back dynamically to the hostname extracted from `RequestURL`.
  - *Delete*: Removes the item from `CustomUploadersList`, adjusts active destination selection indices (`FixSelectionsAfterRemoval`), and shifts focus to the adjacent item or clears the selection.
- **Invalid-Document Repair**: During `.sxcu` JSON file import or startup loading, deserialization failures throw catchable exceptions, logging to debug and showing status errors without corrupting existing in-memory configurations. Individual items run backwards-compatibility migrations (`CheckBackwardCompatibility()`).

### Precedent B: Postman Environments and Variables Editor
Sources:
- [Postman Official Documentation: Edit and set environment variables](https://learning.postman.com/docs/use/send-requests/variables/environment-variables.md)
- [Postman Official Documentation: Define variables](https://learning.postman.com/docs/use/send-requests/variables/define-variables.md)
- [Postman Official Documentation: Store secrets in Postman Vault](https://learning.postman.com/docs/use/postman-vault/postman-vault-secrets.md)

- **Editor Shape**: Tabular grid representation. Each row corresponds to a variable definition with columns for active checkbox toggle, variable name, secret/secure toggle, local value, and optional description / shared value. Environment switcher in workbench header provides master selection between distinct environment sets.
- **Validation Timing**: Key names and values validate inline as typed. Unresolved or circular variable references (e.g. `{{var}}`) flag ambient warning icons in request builders and hover popovers.
- **Secret Handling**: Postman distinguishes plain variables from secure variables and vault secrets:
  - *Masking*: Secure variables mask values in the UI by default.
  - *Encryption*: Secure variables are encrypted locally using AES-256 before storage on disk.
  - *Isolation*: Sensitive values are kept local by default and are not exported or synced to shared cloud workspaces unless explicitly opted into via Shared Vault or Shared Value columns.
- **Create / Rename / Delete Behavior**: Adding an entry appends a row at the bottom of the table. Renaming a variable key immediately cascades to local references. Deleting a variable removes the row, or the user can clear the row's checkbox to disable the entry without deleting its content.
- **Invalid-Document Repair**: Postman isolates environment parse errors. When an environment file cannot be parsed or loaded, it is marked with an error state, preventing accidental overwrite of corrupt data on disk while permitting manual re-import or deletion.

### Precedent C: Visual Studio Code Settings Editor and `settings.json`
Sources:
- [Visual Studio Code Documentation: User and workspace settings](https://code.visualstudio.com/docs/configure/settings)
- [Visual Studio Code Documentation: Editing JSON with VS Code](https://code.visualstudio.com/docs/languages/json)
- [Visual Studio Code Documentation: Extension configuration contribution point](https://code.visualstudio.com/api/references/contribution-points#contributes.configuration)

- **Editor Shape**: Dual-mode configuration editor. A searchable, categorized graphical Settings Editor UI operates over a structured schema, accompanied by direct access to the raw backing JSON file (`settings.json`). Changes made in the GUI instantly update the JSON file on disk, and manual edits in the JSON file immediately reflect in the GUI.
- **Validation Timing**: Continuous schema-backed validation. The JSON document validates against a published JSON Schema with IntelliSense, flagging unknown keys, type mismatches, and syntax errors with inline squiggles. In the GUI, input controls validate types and regex constraints on change/blur.
- **Secret Handling**: Plain settings files store values as unencrypted JSON text. Sensitive tokens (such as GitHub / Azure authentication tokens) are explicitly excluded from `settings.json` and delegated to the native OS credential store via VS Code's SecretStorage API (Windows Credential Manager, macOS Keychain, Linux libsecret).
- **Create / Rename / Delete Behavior**:
  - *Create*: New keys are added by populating a non-default value in the GUI or inserting a key in the JSON object.
  - *Delete / Reset*: The GUI provides a "Reset Setting" action via gear context menu, which completely removes the key from `settings.json`, reverting it to the default schema value.
- **Invalid-Document Repair**: If `settings.json` contains syntax errors or invalid JSON, VS Code prevents GUI write operations: it surfaces a blocking notification (*"Unable to write into user settings. Please open user settings to correct errors/warnings in it and try again."*) and directs the user to open `settings.json` directly to repair the text rather than overwriting or truncating the corrupt document.

---

## 3. Product Implications for Upit v0.5

The current v0.5 direction has two desktop responsibilities:

1. Manage the complete Configuration Set: the three existing Global Configuration fields plus full create/edit/rename/delete operations for Uploaders and Shorteners.
2. Provide Manual Upload for one file selected through a picker or drag-and-drop, with CLI-parity overrides, one active upload with progress and cancellation, and Final URL/Original URL/warning/copy result presentation.

### Sourced Architectural Facts and Direct Implications

1. **Keep configuration editing and upload execution as separate UI workflows**:
   - ShareX combines destination configuration, upload testing, and execution in one product, but its editor still separates request, body, response, and test concerns.
   - Upit can keep one desktop process while presenting distinct Configuration Set management and Manual Upload views. The accepted one-file contract does not require a queue, history, concurrent upload scheduler, or resident process.
2. **Schema-driven form shape**:
   - ShareX organizes an HTTP destination into Overview, Request, Body, Response, and Test.
   - Upit already has validated schema definitions for Uploaders and Shorteners. A desktop editor needs an inspectable master list and a detail view corresponding to the existing request, Request Body Mode, and Response Extractor fields.
3. **Validation timing and durability**:
   - In ShareX, editing mutates an in-memory object graph before persistence.
   - VS Code and Postman protect file integrity by surfacing validation failures rather than blindly replacing corrupt configuration. Upit v0.4 already establishes strict validation and atomic Global Configuration replacement. v0.5 must preserve valid-only publication when extending persistence to the other Configuration Set documents.
4. **Invalid-document resilience**:
   - If an existing Configuration Set document contains syntax or validation errors, the desktop app must not overwrite or truncate it. The unresolved UI choice is whether repair happens in an integrated raw editor or through an external-editor handoff.
5. **Secret handling**:
   - ShareX stores raw tokens in JSON. VS Code keeps ordinary settings in JSON and routes application-managed secrets to OS keychains. Postman offers local encryption and vault storage.
   - Upit Configuration Set documents are portable JSON files and can contain credentials in headers, query values, fields, or request data. v0.5 still needs an explicit masking and storage decision.
6. **Manual Upload reuse**:
   - The accepted desktop behavior matches the existing single-file application workflow instead of defining a batch protocol. Progress reporting needs a new observation seam, but cancellation and Uploader/Shortener/clipboard/timeout choices should retain the existing application semantics.

---

## 4. Recommendations and Unresolved Decisions

### Separated Recommendations

1. **Editor topology**: Use a master-detail layout based on the ShareX and Postman precedents: named Uploaders and Shorteners in a sidebar, with structured request, body, and response sections in the detail view.
2. **Persistence lifecycle**: Keep edits in memory until an explicit valid Save. Reuse Upit's strict validation and native staged replacement instead of introducing autosave, backup files, or lock files.
3. **Dirty state and safe navigation**: Prompt before switching definitions, reloading, or closing when unsaved edits exist.
4. **Wails 3 adoption stance**: Wails v3 is currently Beta (`v3.0.0-beta.26`) with a documented stable API but pre-release status. Isolate Wails services as thin adapters so configuration and upload behavior remain testable as standard Go.

### Decisions Settled During Grilling

1. Use structured forms for valid documents and an integrated raw repair mode only when an existing document cannot be represented safely by the forms.
2. Keep the existing plaintext JSON storage model; mask configured values in the UI by default and require an explicit reveal/edit/copy action.
3. Publish only through explicit Save after strict validation; prompt before losing dirty edits.
4. Reject stale saves when any relevant Configuration Set document changed since load; do not auto-merge or create lock files.
5. Run one desktop instance and focus its existing window on subsequent launches.

---

## 5. Wails 3 Frontend Template Options

### Officially Supported Built-in Templates and Build Tooling (as of 2026-09-30)
Sources:
- [Wails v3 Starter Templates Redesign (commit 23eb116 / PR #5642)](https://github.com/wailsapp/wails/commit/23eb1165bbc2434ebe5176240e0a5f394c752c76)
- [Wails v3 Template Registry: `v3/internal/templates/templates.go`](https://raw.githubusercontent.com/wailsapp/wails/master/v3/internal/templates/templates.go)
- [Wails v3 Build Assets Taskfile: `v3/internal/commands/build_assets/Taskfile.tmpl.yml`](https://raw.githubusercontent.com/wailsapp/wails/master/v3/internal/commands/build_assets/Taskfile.tmpl.yml)
- [Wails v3 Guide: Using Other Frontend Frameworks (`docs/src/content/docs/guides/dev/frontend-frameworks.mdx`)](https://github.com/wailsapp/wails/blob/master/docs/src/content/docs/guides/dev/frontend-frameworks.mdx)

1. **Built-in Template Registry**:
   In Wails v3 (PR #5642, merged June 2026), built-in starters were consolidated to four frameworks. TypeScript is the default across all four frameworks and claims the bare template name. JavaScript variants use an explicit `-js` suffix:
   - `vanilla` (TypeScript, default) / `vanilla-js`
   - `react` (TypeScript) / `react-js`
   - `vue` (TypeScript) / `vue-js`
   - `svelte` (TypeScript) / `svelte-js`
   Earlier standalone starters (`preact`, `lit`, `solid`, `qwik`, `sveltekit`, `react-swc`, and legacy `-ts` aliases) were retired from internal templates in favor of a standard Vite BYO-frontend workflow.

2. **Generated Frontend Toolchain and Build Orchestration**:
   - **Bundler / Dev Server**: All official templates run on **Vite** (`vite ^8.0.5`). During `wails3 dev`, Wails launches Vite on a fixed loopback port (`WAILS_VITE_PORT`, default `9245`, `strictPort: true`) and proxies dev requests with HMR. Production builds run `vite build` into `frontend/dist/`.
   - **Package Manager Integration**: Projects ship a `Taskfile.yml` orchestrated by `go-task` that delegates to `build/Taskfile.yml`. The Taskfile dispatches commands through `install:frontend:deps:{{.PACKAGE_MANAGER}}` and `frontend:run:{{.PACKAGE_MANAGER}}`, supporting `npm` (default), `pnpm`, `yarn`, or `bun` without wrapper scripts.
   - **Bindings Generation**: `generate:bindings` compiles Go service definitions into typed modules under `frontend/bindings/<module-path>/`. TypeScript templates generate typed interface/class wrappers with method promises and model declarations.
   - **Asset Pipeline Contract**: The Go backend embeds `//go:embed all:frontend/dist` via `application.AssetFileServerFS(assets)`. Any toolchain producing static assets in `frontend/dist/` satisfies the runtime contract.

### Comparison: Vanilla TypeScript vs. Smallest Maintained Component Framework

Evaluating Vanilla TypeScript against Svelte (the smallest maintained component framework officially shipped in Wails v3) for Upit's single-window desktop UI (four product areas: Global Configuration, Uploaders, Shorteners, and Manual Upload):

| Architectural Concern | Vanilla TypeScript (`vanilla` template) | Svelte 5 (`svelte` template) |
| :--- | :--- | :--- |
| **Dependencies** | Minimum possible: `typescript`, `vite`, `@wailsio/runtime`. | Minimal framework overhead: `svelte` (^5.46.4), `@sveltejs/vite-plugin-svelte`, `svelte-check`, `typescript`, `vite`, `@wailsio/runtime`. Zero runtime dependencies outside the compiler. |
| **State & Reactivity** | **Manual**: requires hand-rolled observer/event-emitter stores, manual dirty-flag checks across nested inputs, and imperative element updates. | **Native compiler runes**: `$state()` and `$derived()` handle reactive inputs, dirty-state comparison, and derived validation summaries without external state libraries. |
| **Form Binding & Inputs** | **Manual DOM wiring**: each form field (HTTP methods, nested headers, multipart fields, URL parsers, query parameters) needs manual `addEventListener("input")` and `element.value` extraction. | **Two-way binding**: `bind:value`, `bind:checked`, and `bind:group` bind nested model objects directly to inputs with automatic type coercion. |
| **Master-Detail & Dynamic Collections** | **Manual DOM mutations**: adding, deleting, or reordering uploader list items, header rows, or form arguments requires manual `document.createElement`, element recycling, or innerHTML replacement with manual listener re-binding. | **Declarative blocks**: `{#each items as item (item.id)}` renders master lists and dynamic key-value tables cleanly with keyed DOM reconciliation. |
| **Dirty-State Navigation Guard** | **Manual tracking**: requires deep equality checks against original snapshots before master-list selection changes, window close, or tab navigation. | **Declarative snapshots**: compare reactive `$state` against a frozen baseline copy; trigger modal prompts or cancel navigation in one centralized check. |
| **Failure Modes & Maintenance** | Risk of DOM-state desynchronization, listener memory leaks, and brittle selector queries during frequent schema-driven form updates. | Compiler-verified template types via `svelte-check`; clear component separation (`Sidebar.svelte`, `UploaderDetail.svelte`, `ManualUpload.svelte`). |

### Initial Research Recommendation (Superseded)

**Superseded recommendation: `svelte` (Svelte 5 + TypeScript + Vite).**

*Reasoning*:
1. **Boring, single-purpose ergonomics without boilerplate**: Upit's v0.5 desktop UI is form-dense. Uploaders and Shorteners contain multi-section HTTP requests, dynamic key-value headers, multipart fields, body types, and response parser extractors. Hand-rolling two-way binding, dirty tracking, validation error displays, and dynamic array addition/removal in Vanilla TypeScript would require hundreds of lines of fragile DOM manipulation infrastructure.
2. **Lightest maintained footprint**: Svelte compiles down to minimal surgical JavaScript with no virtual DOM runtime. Unlike React (which adds `react`, `react-dom`, `@types/*`, hook lifecycle subtleties, and form helper libraries), Svelte handles dynamic forms and dirty tracking natively via built-in `$state` runes.
3. **First-party, official Wails 3 alignment**: Svelte is one of the four curated, first-party supported templates in Wails v3, maintained directly within `v3/internal/templates/svelte`. It integrates out of the box with the standard Wails `Taskfile.yml`, `wails3 dev` HMR loop, and `@wailsio/runtime` bindings without custom Vite adapter maintenance.
4. **Isolated surface**: In line with Upit's architectural boundaries, Svelte remains strictly contained within `frontend/`. All business logic, configuration schema parsing, persistence, secret encryption/storage, and upload engines remain standard, Wails-free Go.

---

## 6. Selected Vue and shadcn-vue Stack

### Decision Record: Superseding Framework Choice
- **Status**: Accepted (2026-09-30).
- **Decision**: The user explicitly selected **Vue 3 + TypeScript + Vite + shadcn-vue** for the Upit v0.5 desktop frontend.
- **Supersedes**: This choice supersedes the Section 5 research recommendation (Svelte 5). While Section 5 preserves the historical technical comparison between Vanilla TypeScript and Svelte 5, all v0.5 implementation work proceeds strictly on the Vue 3 + shadcn-vue foundation documented below.

### Primary-Source Architectural Facts (as of 2026-09-30)

1. **Wails v3 Official Starter**:
   - Upit builds upon the official Wails v3 `vue` starter template (`wails3 init -t vue`), which defaults to TypeScript and Vite ([Wails v3 Starter Templates PR #5642](https://github.com/wailsapp/wails/commit/23eb1165bbc2434ebe5176240e0a5f394c752c76), [Wails v3 Template Registry](https://raw.githubusercontent.com/wailsapp/wails/master/v3/internal/templates/templates.go)).
   - Baseline frontend dependencies shipped in the starter:
     - Runtime: `vue` (`^3.2.45`), `@wailsio/runtime` (`latest`).
     - Dev tooling: `vite` (`^8.0.5`), `@vitejs/plugin-vue` (`^6.0.0`), `typescript` (`^4.9.3`), `vue-tsc` (`^1.0.11`).
     - Vite configuration (`vite.config.ts`) includes `@vitejs/plugin-vue` and `@wailsio/runtime/plugins/vite` binding resolution on loopback port 9245 (`strictPort: true`).

2. **shadcn-vue Architecture & Delivery Model**:
   - Sources: [shadcn-vue Documentation](https://www.shadcn-vue.com/docs), [shadcn-vue CLI Specification](https://raw.githubusercontent.com/unovue/shadcn-vue/dev/skills/shadcn-vue/cli.md), [shadcn-vue Rules](https://raw.githubusercontent.com/unovue/shadcn-vue/dev/skills/shadcn-vue/SKILL.md).
   - **Copied Source Code, Not a Runtime Library**: shadcn-vue is not distributed as a monolithic component NPM dependency (`node_modules`). Instead, components are generated as individual, editable source files directly into the repository (`frontend/src/components/ui/`) via the official CLI (`npx shadcn-vue@latest add <component>`).
   - **Primitive Component Foundation (`reka-ui`)**: Unstyled headless behavior, keyboard navigation, focus trapping, and ARIA state management are powered by **Reka UI** (`reka-ui`, the v2 rebranding of Radix Vue by the unovue project). Reka UI provides WAI-ARIA authoring practices compliance and screen reader compatibility out-of-the-box.
   - **Maintenance & Accessibility Implications**:
     - *Local Ownership*: The application directly owns component markup, transitions, and styling. Upstream enhancements or fixes require diffing via CLI (`npx shadcn-vue@latest add <component> --diff`) rather than semver package bumps.
     - *Accessible Primitives*: Compound components (e.g. `Dialog`, `Sheet`, `Select`, `DropdownMenu`) inherit full keyboard interaction standards (Escape dismissal, tab loops, Arrow-key navigation) and accessibility attributes (`aria-expanded`, `aria-controls`, `aria-invalid`) from Reka UI primitives. Dialogs strictly enforce accessible headers (`DialogTitle`, `SheetTitle`).

3. **Styling and Tailwind Integration**:
   - shadcn-vue generates utility-driven markup styled via Tailwind CSS, using `clsx` and `tailwind-merge` packaged in a local utility helper (`src/lib/utils.ts` exporting `cn(...)`).
   - Variant generation uses `class-variance-authority` (`cva`).
   - Tailwind v4 is supported directly with CSS-first configuration (`@theme inline`), and Tailwind v3 is supported via standard `tailwind.config.js`.

4. **Minimum Necessary Frontend Additions (Zero Speculative Extras)**:
   To integrate shadcn-vue cleanly into Wails' Vite frontend without extraneous packages or boilerplate:
   - **Path Aliasing**:
     - `tsconfig.json`: add `"baseUrl": "."` and `"paths": { "@/*": ["./src/*"] }` to resolve `@/` imports.
     - `vite.config.ts`: add `resolve: { alias: { "@": path.resolve(__dirname, "./src") } }` (using Node standard `node:path`).
     - `devDependencies`: `@types/node` for path resolution typing.
   - **Core Utility & Primitive Runtime Packages**:
     - `reka-ui`: underlying headless component primitives.
     - `clsx`, `tailwind-merge`: class composition helper (`cn`).
     - `class-variance-authority`: component variants.
     - `lucide-vue-next` (or `@lucide/vue`): official icon set for actions, status indicators, and controls.
   - **Styling**:
     - `tailwindcss` (and bundler plugin `@tailwindcss/vite` for Tailwind v4 or `postcss` / `autoprefixer` for v3).
   - **Component Configuration**:
     - Root `components.json` with framework `vite`, base `reka`, style `new-york`, and path aliases mapping `components` to `@/components` and `utils` to `@/lib/utils`.
   - **Excluded / Non-Goals**: No state management frameworks (Pinia/Vuex), no external form suites (VeeValidate/Zod forms can remain plain reactive Vue composition unless needed), no UI router (single-window desktop tabs), no extra registries.

---

## 7. Wails 3 Capability Check for Settled Desktop Behavior

Primary-source mapping against Wails `v3.0.0-beta.26` official source (`github.com/wailsapp/wails/v3` at tag `v3.0.0-beta.26`).

### Feasibility and API Mapping

| Capability / Settled Behavior | Framework Support Status | Manager / Method / Event Names (`v3.0.0-beta.26`) | Primary Citation & Architecture Details |
| :--- | :--- | :--- | :--- |
| **Single-instance focus** | Direct framework support | `application.Options{ SingleInstance: &application.SingleInstanceOptions{ UniqueID: "...", OnSecondInstanceLaunch: func(data application.SecondInstanceData) { window.Focus() } } }` | `v3/pkg/application/single_instance.go:34-48`, `v3/pkg/application/application_options.go:38`. Built-in cross-platform locking (Windows named mutex + message window via `user32.dll`, macOS `NSDistributedNotificationCenter`, Linux D-Bus `SendMessage`). When a second launch occurs, `OnSecondInstanceLaunch` fires on the primary process, passing args, working directory, and custom payload. Focusing the existing window requires an explicit call to `window.Focus()` (or `window.Restore()` if minimized) inside the callback. |
| **Native open-file dialog** | Direct framework support | `app.Dialog.OpenFile()`, `app.Dialog.OpenFileWithOptions(opts)` returning `*application.OpenFileDialogStruct`. Execution via `.PromptForSingleSelection()` or `.PromptForMultipleSelection()`. Chainable options: `.CanChooseFiles(true)`, `.CanChooseDirectories(false)`, `.AddFilter(name, pattern)`, `.AttachToWindow(w)`. | `v3/pkg/application/dialog_manager.go:19-27`, `v3/pkg/application/dialogs.go:128-185`. Backed by native platform file choosers: `IFileOpenDialog` on Windows (`dialogs_windows.go`), `NSOpenPanel` on macOS (`dialogs_darwin.go`), GTK `GtkFileChooserNative` on Linux (`dialogs_linux.go`). Returns selected path `string` and `error` synchronously or via modal window attachment. |
| **One-file drag and drop** | Direct framework support with single-file validation flag | Window configuration: `application.WebviewWindowOptions{ EnableFileDrop: true }`. Frontend DOM markup: HTML element with attribute `data-file-drop-target="true"`. Backend window event listener: `window.OnWindowEvent(events.Common.WindowFilesDropped, func(e *application.WindowEvent) { paths := e.Context().DroppedFiles() })`. | `v3/pkg/application/webview_window_options.go:138-142`, `v3/pkg/application/webview_window.go:1625-1638`, `v3/pkg/application/context_window_event.go:14-25`, `v3/pkg/events/events.go:42`. Wails delivers a slice of path strings (`[]string`). Upit's settled single-file constraint requires custom validation logic in the handler (inspecting `len(paths) == 1` and rejecting or taking the first item if multiple are dragged). |
| **Go-to-frontend progress events** | Direct framework support | Backend dispatch: `app.Event.Emit("upload:progress", payload)`. Frontend listener: `events.On("upload:progress", callback)` from `@wailsio/runtime`. | `v3/pkg/application/event_manager.go:27-41`, `v3/internal/runtime/desktop/@wailsio/runtime/src/events.ts`. In v3, custom events route through `app.Event.Emit` on the Go side and `events.On` / `events.Emit` in `@wailsio/runtime`. Replaces v2's context-bound `runtime.EventsEmit`. Payload is automatically serialized as JSON to the webview runtime. |
| **Cancellation calls (Frontend to Go)** | Direct framework support | Go service method signature: `func (s *UploadService) Upload(ctx context.Context, ...) error`. Frontend invocation: `const promise = Call({ methodName: "...", args: [...] })` (or generated binding); cancellation via `promise.cancel()`. | `v3/pkg/application/bindings.go:46-77,159-220`, `v3/pkg/application/messageprocessor_call.go:18-80`, `v3/internal/runtime/desktop/@wailsio/runtime/src/cancellable.ts`, `v3/internal/runtime/desktop/@wailsio/runtime/src/calls.ts:60-95`. When a bound service method takes `context.Context` as its first parameter (`needsContext`), Wails passes a child context linked to the in-flight binding ID. Calling `.cancel()` on the frontend's `CancellablePromise` dispatches `CancelCall` (internal code `CallCancel`), which calls the Go `context.CancelFunc` and immediately terminates context-sensitive operations (e.g. `http.Request` with context). |
| **Window-closing hooks** | Direct framework support | `window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { if isDirty { e.Cancel() } })`. | `v3/pkg/application/webview_window.go:366-370,968-985,1613-1623`, `v3/pkg/events/events.go:38`. `RegisterHook` registers a synchronous interceptor executed before the window closes. Calling `e.Cancel()` marks the `WindowEvent` cancelled (`cancelled.Store(true)`), preventing window destruction. Enables showing a confirmation dialog when unsaved changes exist. |
| **Window-focus events** | Direct framework support | Window event listener: `window.OnWindowEvent(events.Common.WindowFocus, func(e *application.WindowEvent) { ... })` and `window.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) { ... })`. Imperative focus: `window.Focus()`. | `v3/pkg/application/webview_window.go:942-965,1688-1693`, `v3/pkg/events/events.go:43,46`. Native platform window focus/blur notifications are normalized to `events.Common.WindowFocus` and `WindowLostFocus`. `window.Focus()` triggers native window activation (`SetForegroundWindow` on Windows, `makeKeyAndOrderFront` on macOS, `gtk_window_present` on Linux). |
| **Native runnable builds (Win/macOS/Linux)** | Direct framework support via standard Taskfile tooling | `Taskfile.yml` orchestrated build pipeline using `wails3 task`: Windows: `task build:native` or `task package` (NSIS/MSIX); macOS: `task build:native`, `task build:universal`, `task package` (`.app` bundle / `.dmg`); Linux: `task build:native`, `task package` (AppImage, deb, rpm via nfpm). | `v3/internal/commands/build_assets/Taskfile.tmpl.yml`, `v3/internal/commands/build_assets/{windows,darwin,linux}/Taskfile.yml`. Transparent build definitions driven by `go-task`. Cross-compilation across platforms uses Docker toolchain images (`Dockerfile.cross` with Zig C compiler for Linux/Windows CGo). Produces native standalone executables embedding the webview frontend. |

### Flags and Custom Logic Requirements

All eight settled desktop interaction requirements map cleanly to existing, first-party Wails `v3.0.0-beta.26` APIs without framework omissions. Two minor product constraints require application-level handling:

1. **One-File Drag-and-Drop Constraint**:
   Wails' native drag-and-drop mechanism passes all dropped files as a slice (`[]string`) via `e.Context().DroppedFiles()`. The framework does not have an option to reject multi-file drops at the OS level before they hit the handler.
   - *Custom Code Needed*: A small guard in the `WindowFilesDropped` callback or frontend handler verifying `len(files) == 1`. If `len(files) > 1`, either ignore the drop, surface an inline validation warning, or select only `files[0]` according to product UX specification.

2. **Single-Instance Restore on Launch**:
   Wails handles cross-process mutual exclusion and payload handoff via `SingleInstanceOptions`, but does not automatically focus or restore the existing window upon receiving a second instance notification.
   - *Custom Code Needed*: Inside `OnSecondInstanceLaunch: func(data application.SecondInstanceData)`, the application must explicitly invoke `window.Restore()` (in case the window was minimized) and `window.Focus()`.

### Framework State and Unknowns

- **Upstream Status**: Wails v3 is in active beta (`v3.0.0-beta.26` released 2026-09-25). The core windowing, service binding, and event APIs are confirmed stable and verified against the repository source.
- **Linux Runtime Dependency Note**: Linux native builds use GTK4 and WebKitGTK 6.0 by default. Distributions without WebKitGTK 6.0 (e.g. Ubuntu 22.04 LTS) require compilation with `-tags gtk3` against WebKit2GTK 4.1.
