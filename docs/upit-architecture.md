# Upit Architecture

## 1. Overview

**Upit** is a lightweight, cross-platform file uploader with a headless-first architecture.

The primary interface is a CLI that can run on:

- Windows
- macOS
- Linux desktops
- Headless Linux servers
- SSH sessions
- CI/CD environments
- Containers
- Automation scripts

A Wails 3 desktop application is a deferred future frontend; v0.3 provides the CLI only, and the core application does not depend on Wails.

Primary workflow:

```text
File
 ↓
Upit Core
 ↓
Custom Upload API
 ↓
Original URL
 ↓
Optional URL Shortener
 ↓
Final URL
 ↓
stdout / clipboard
 ↓
Exit
```

Upit uses an **on-demand / process-per-invocation** runtime model.

When Upit is not being used, no Upit process should remain running.

---

## Current v0.4 implementation

This section is authoritative for the implemented CLI runtime. The application module in `internal/app` owns strict Configuration Set loading and validation, safe Global Configuration persistence, Uploader body modes, HTTP execution, bounded response handling, Response Extractors, URL validation, Shortener processing, clipboard work, and structured failures. The CLI is the external seam for uploads and non-interactive Configuration Management; it does not duplicate validation or persistence logic.

Sections that describe a separate Request Builder, Response Parser, or broader package tree are historical architecture proposals. They remain useful roadmap context, but they are not the v0.4 implementation contract; ADR-0008, ADR-0009, and this section take precedence.

## 2. Product Model

Upit is divided into three main layers:

```text
Upit
├── Core
│   ├── Application / deep upload module (`internal/app`)
│   ├── URL Shortener
│   └── Configuration
│
├── CLI
│   └── Primary headless interface
│
└── Desktop
    └── Optional Wails 3 frontend (future scope)
```

The CLI is considered a first-class interface, not a fallback for the GUI.

The desktop application is only a future frontend for users who want a graphical interface for:

- Managing settings
- Managing uploaders
- Managing URL shorteners
- Selecting and uploading files manually
- Viewing results

---

## 3. Main Goals

Upit should:

- Upload arbitrary files to configurable HTTP endpoints.
- Support multiple custom uploaders.
- Parse returned URLs from API responses.
- Optionally shorten returned URLs.
- Support headless Linux servers.
- Work without a GUI or desktop session.
- Produce script-friendly output.
- Optionally copy the final URL to the clipboard.
- Use Go as the primary language.
- Keep idle memory usage effectively at zero.
- Run only when explicitly invoked.
- A future optional Wails 3 desktop frontend may be added after the CLI release scope.
- Keep upload logic independent from CLI and GUI layers.

Upit should not require:

- A background daemon.
- A system tray process.
- A resident worker.
- A running GUI.
- GTK/WebKit dependencies for the CLI.
- File watchers.
- Periodic polling.
- A permanently running event loop.

---

## 4. Runtime Model

Upit follows an **on-demand / process-per-invocation** architecture.

### Idle state

```text
No invocation
 ↓
No Upit process
 ↓
~0 MB Upit RAM usage
```

### CLI / headless invocation

```text
CLI / SSH / Script / Explorer / Finder / Launcher
                     ↓
                Start Upit
                     ↓
            Load required config
                     ↓
                Upload file
                     ↓
          Optional URL shortening
                     ↓
              Print FinalURL
                     ↓
       Optional clipboard action
                     ↓
           Release resources
                     ↓
                   Exit
```

### Deferred GUI invocation (future; not v0.3)

```text
User opens Upit Desktop
         ↓
Wails GUI starts
         ↓
Manage settings / uploaders / shorteners
         ↓
Optional manual upload
         ↓
User closes application
         ↓
Process exits completely
```

There is no `close to tray` behavior.

---

## 5. Technology Stack (historical roadmap; CLI v0.3 scope)

### Core and CLI

```text
Language: Go
```

Go is responsible for:

- Upload engine
- HTTP requests
- Multipart streaming
- Response parsing
- Configuration
- URL shortening
- Clipboard integration
- Application services
- CLI
- Headless/server operation

### Desktop GUI (v0.5 planned)

```text
Framework: Wails v3.0.0-beta.26
Frontend: Vue 3 + TypeScript + Vite + shadcn-vue
```

Wails is confined to the desktop executable and thin frontend adapters. The application modules and CLI do not import Wails packages.

---

## 6. Wails Version Policy (v0.5 planned)

Upit v0.5 pins Wails `v3.0.0-beta.26`. Its API is documented as stable, but the release remains beta; stable Wails 3 is not a prerequisite for v0.5.

Every Wails upgrade is explicit:

```text
Proposed Wails release
 ↓
Review changelog and breaking changes
 ↓
Update the exact pinned version
 ↓
Run application and frontend checks
 ↓
Build and smoke native desktop targets
 ↓
Commit the reviewed upgrade
```

Application modules and the CLI remain independent of Wails so a framework upgrade cannot change headless behavior implicitly.

---

## 7. Binary Strategy (v0.5 planned)

Upit v0.5 keeps the existing `upit` CLI and adds a separately launched `upit-desktop` executable.

Executable names:

```text
upit
upit-desktop
```

### `upit`

The primary CLI/headless binary.

Properties:

- Pure Go application layer.
- No Wails startup.
- No WebView startup.
- No GUI dependency.
- Suitable for Linux servers.
- Suitable for containers.
- Suitable for scripts and CI/CD.

Example:

```bash
upit upload file.zip
```

### `upit-desktop` (v0.5 planned)

The optional Wails desktop application.

Properties:

- Starts Wails/WebView only when explicitly launched.
- Uses the same application service and core engines as the CLI.
- Exits completely when closed.
- Does not run in the system tray.

Architecture:

```text
                ┌──────────────────┐
                │     Upit Core    │
                │ Upload/Shortener │
                └────────┬─────────┘
                         │
             ┌───────────┴───────────┐
             ▼                       ▼
         upit CLI              upit-desktop
         headless                Wails 3
```

---

## 8. Configuration Directory

All runtime configuration is stored under:

```text
~/.config/upit/
```

Recommended structure:

```text
~/.config/upit/
├── config.json
└── custom-uploader.json
```

On a Linux server this may resolve to:

```text
/home/user/.config/upit/
```

For root:

```text
/root/.config/upit/
```

Configuration is resolved relative to the current user.

---

## 9. `config.json`

`config.json` contains global application settings.

Example:

```json
{
    "version": 1,
    "defaultUploader": "personal",
    "copyToClipboard": false
}
```

v0.1 responsibilities:

- Configuration version
- Default Uploader
- Clipboard behavior

URL-shortener and open-after-upload settings are not valid v0.1 fields. They require a later versioned schema.

Clipboard behavior must be optional because headless servers may not have a graphical clipboard service.

---

## 10. `custom-uploader.json`

In v0.1, `custom-uploader.json` stores Uploader definitions. Later versions may add URL-shortener definitions through an explicit schema version change.

API credentials are stored directly in this file.

Because this file may contain secrets:

- Never print secrets in logs.
- Mask sensitive headers in debug output.
- Do not commit the real file to Git.
- On Unix, require mode `0600` before using the file.
- Provide an example file without real credentials.

v0.1 example:

```json
{
    "version": 1,
    "uploaders": {
        "personal": {
            "request": {
                "method": "POST",
                "url": "https://upload.example.com/api/upload",
                "headers": {
                    "Authorization": "Bearer YOUR_API_KEY"
                },
                "query": {},
                "body": "multipart",
                "fileField": "file",
                "fields": {}
            },
            "response": {
                "url": {
                    "type": "json",
                    "path": "$.url"
                },
                "error": {
                    "type": "json",
                    "path": "$.error"
                }
            }
        }
    }
}
```

---

## 11. Configuration Versioning

Configuration schemas should contain a version from the beginning.

```json
{
    "version": 1
}
```

This allows controlled migrations:

```text
v1 → v2
```

without guessing the format of existing user files.

---

## 12. JSON Schema

JSON Schema publication is delivered in v0.4 alongside strict runtime validation. Go remains the execution authority for protocol semantics, expression compilation, filesystem permissions, and cross-document references.

Published artifacts:

```text
schemas/
├── config.schema.json
├── custom-uploader.schema.json
└── custom-shortener.schema.json
```

Editors associate these schemas by Configuration Set filename; user documents do not gain a `$schema` member.

Benefits:

- Validation
- Editor autocomplete
- Better error messages
- Easier documentation
- Easier schema evolution

---

## 13. Core Architecture

The core must be independent from:

- CLI implementation
- Wails
- WebView
- Clipboard
- Notifications
- Explorer/Finder integration
- Shell integration
- System tray

Core responsibility:

```text
File + Uploader
          ↓
      HTTP Upload
          ↓
    Parse Response
          ↓
     UploadResult
```

Suggested engine API:

```go
func (e *Engine) Upload(
    ctx context.Context,
    filePath string,
    uploader Uploader,
) (*UploadResult, error)
```

---

## 14. Upload Result

Suggested result model:

```go
type UploadResult struct {
    OriginalURL  string
    FinalURL     string
    ThumbnailURL string
    DeletionURL  string
    StatusCode   int
}
```

The engine returns data.

It does not decide whether to:

- Print it
- Copy it
- Display it in GUI
- Save it to history

Those actions belong to higher layers.

---

## 15. Proposed Request Builder (deferred; not v0.3)

HTTP request construction should be separated from the upload engine.

This is a historical proposal. v0.3 keeps request construction inside the deep upload module in `internal/app`; do not create the package split below for the implemented release.

Suggested structure:

```text
internal/request/
├── builder.go
├── multipart.go
├── binary.go
├── form.go
└── json.go
```

Initial body types:

```text
multipart
binary
form
json
```

The same request infrastructure should be reusable by:

- Upload engine
- URL shortener
- Future HTTP-based processors

---

## 16. Streaming Uploads

File uploads should be streamed.

Do not use:

```go
data, err := os.ReadFile(filePath)
```

for large upload data.

Preferred flow:

```text
os.File
   ↓
io.Reader
   ↓
multipart.Writer / request body
   ↓
HTTP transport
```

For multipart streaming, `io.Pipe` can be used.

Goals:

- Avoid loading the entire file into RAM.
- Keep memory usage relatively constant.
- Allow large file uploads.
- Support cancellation using `context.Context`.

---

## 17. `context.Context`

All network operations should receive a context.

Example:

```go
func (e *Engine) Upload(
    ctx context.Context,
    filePath string,
    uploader Uploader,
) (*UploadResult, error)
```

Benefits:

- Cancel upload
- Timeout
- GUI cancellation
- Graceful shutdown
- CI/CD timeout control

---

## 18. Proposed Response Parser (deferred; not v0.3)

Response parsing should be independent from request logic.

This is a historical proposal. v0.3 keeps response handling and Response Extractors inside the deep upload module in `internal/app`; do not create the package split below for the implemented release.

Suggested structure:

```text
internal/response/
├── parser.go
├── json.go
├── regex.go
├── header.go
└── body.go
```

Initial parser types:

```text
json
header
regex
body
```

Example:

```json
{
    "response": {
        "url": {
            "type": "json",
            "path": "$.data.file.url"
        }
    }
}
```

Different upload APIs should be handled through configuration instead of hardcoded response structures.

---

## 19. URL Shortener (v0.2 historical schema; still version 1 in v0.3)

URL shortening is a separate post-processing step.

This section records the implemented v0.2 Shortener contract. It is retained for compatibility context; v0.3 does not expand Shortener request body or Response Extractor capabilities.

It must not be embedded inside the upload engine.

Flow:

```text
File
 ↓
Upload Engine
 ↓
OriginalURL
 ↓
URL Shortener Engine
 ↓
FinalURL
```

The URL shortener receives:

```text
{input}
```

where `{input}` is the original uploaded URL.

Example:

```json
{
    "request": {
        "method": "POST",
        "url": "https://thienhung.io.vn/api/v2/links",
        "headers": {
            "X-API-Key": "API",
            "Accept": "application/json"
        },
        "data": {
            "target": "{input}"
        }
    },
    "response": {
        "url": {
            "type": "json",
            "path": "$.link"
        }
    }
}
```

---

## 20. Preserve Original and Final URLs

Never discard the original upload URL.

Without shortening:

```text
OriginalURL == FinalURL
```

With shortening:

```text
OriginalURL = https://files.example.com/long/path/file.zip
FinalURL    = https://short.example/abc123
```

Keeping both URLs is useful for:

- History
- Debugging
- GUI display
- Retrying the shortener
- Automation
- Future metadata storage

---

## 21. URL Shortener Failure Behavior

If upload succeeds but URL shortening fails, the upload itself should not normally be treated as failed.

Default behavior:

```json
{
    "urlShortener": {
        "fallbackToOriginal": true
    }
}
```

Flow:

```text
Upload success
 ↓
Shortener fails
 ↓
Use OriginalURL as FinalURL
 ↓
Continue successfully
```

---

## 22. Application Service Layer (historical GUI proposal; v0.3 uses internal/app)

Create an application layer between interfaces and engines.

Recommended:

```text
internal/app/
└── service.go
```

Architecture:

```text
CLI ──────────┐
              │
              ▼
         App Service
              │
              ├── Upload Engine
              ├── URL Shortener
              ├── Clipboard Adapter
              └── Config
              ▲
              │
Wails GUI ────┘
```

The service coordinates workflows such as:

```text
Load config
 ↓
Upload
 ↓
Shorten
 ↓
Optional clipboard
 ↓
Return result
```

---

## 23. CLI as the Primary Interface

The CLI is the primary runtime interface.

Examples:

```bash
upit upload file.zip
```

Use a specific uploader:

```bash
upit upload file.zip --uploader personal
```

Select a URL shortener:

```bash
upit upload file.zip --shortener kutt
```

Disable shortening:

```bash
upit upload file.zip --no-shorten
```

Disable clipboard:

```bash
upit upload file.zip --no-clipboard
```

Machine-readable output:

```bash
upit upload file.zip --json
```

Example:

```json
{
    "success": true,
    "originalUrl": "https://files.example.com/file.zip",
    "finalUrl": "https://short.example/abc123"
}
```

---

## 24. Standard Output Behavior

For CLI usage, `stdout` is the primary result channel.

The default successful output should be easy to consume:

```text
https://short.example/abc123
```

This enables:

```bash
URL=$(upit upload backup.tar.gz --no-clipboard)
```

and:

```bash
echo "$URL"
```

For structured automation:

```bash
upit upload backup.tar.gz --json | jq -r '.finalUrl'
```

This makes Upit suitable for:

- Bash scripts
- PowerShell
- cron
- CI/CD
- SSH sessions
- Backup scripts
- Deployment scripts
- Other applications

---

## 25. Clipboard Is Optional

Clipboard integration is a post-action, not part of upload success.

Deferred desktop clipboard flow (future; not v0.3):

```text
Upload
 ↓
FinalURL
 ↓
Copy to clipboard
```

Headless flow:

```text
Upload
 ↓
FinalURL
 ↓
stdout
```

If clipboard is unavailable, upload should still succeed.

For example, Linux servers may not provide:

- X11
- Wayland
- Clipboard daemon
- Desktop session

Therefore clipboard failure must not invalidate a successful upload unless the user explicitly requests strict clipboard behavior in the future.

---

## 26. Deferred desktop architecture (not v0.3)

Wails is only a frontend layer.

It must not contain upload business logic.

Recommended desktop layout:

```text
cmd/
└── upit-desktop/
    ├── main.go
    └── bindings/
        ├── upload.go
        ├── config.go
        └── settings.go
```

Frontend:

```text
frontend/
├── src/
├── package.json
└── ...
```

Architecture:

```text
Wails Frontend
      ↓
Wails Bindings
      ↓
App Service
      ↓
Core Engines
```

The GUI should call Go application services directly.

It should not execute the CLI binary as a subprocess for normal operations.

---

## 27. Deferred desktop runtime paths (not v0.3)

The CLI must not initialize Wails or WebView.

Correct:

```text
upit upload file.zip
        ↓
Go runtime only
        ↓
Upload
        ↓
Exit
```

Desktop:

```text
User launches upit-desktop
        ↓
Wails/WebView starts
        ↓
User works with GUI
        ↓
Close
        ↓
Exit
```

Avoid:

```text
upit upload file.zip
        ↓
Start Wails
        ↓
Start WebView
        ↓
Upload
```

---

## 28. Headless Linux Server Support

The CLI must work on Linux servers without:

- GUI
- WebView
- GTK
- Wayland
- X11
- Desktop environment

Example:

```bash
upit upload backup.tar.gz
```

With JSON output:

```bash
upit upload backup.tar.gz --json
```

Example in automation:

```bash
FINAL_URL=$(upit upload backup.tar.gz --no-clipboard)
```

The server package should contain only the CLI binary and required configuration.

---

## 29. Linux User Configuration

Configuration is user-scoped.

Normal user:

```text
/home/nth/.config/upit/
├── config.json
└── custom-uploader.json
```

Another service user:

```text
/home/backup/.config/upit/
├── config.json
└── custom-uploader.json
```

Root:

```text
/root/.config/upit/
├── config.json
└── custom-uploader.json
```

Automation must account for which user executes Upit.

---

## 30. Docker Compatibility

The headless CLI should be suitable for minimal containers.

Example:

```dockerfile
FROM alpine

COPY upit /usr/local/bin/upit

ENTRYPOINT ["upit"]
```

Example usage:

```bash
docker run \
  -v ~/.config/upit:/root/.config/upit \
  -v "$PWD:/data" \
  upit upload /data/file.zip
```

The Docker image must not include Wails or WebView dependencies.

---

## 31. No System Tray (CLI v0.3; desktop roadmap deferred)

System tray support is intentionally removed.

Reason:

```text
System tray
   ↓
Resident process
   ↓
Continuous RAM usage
```

Instead:

```text
Future desktop settings flow (deferred; not v0.3):
→ Launch upit-desktop.

Need upload?
→ Invoke upit.

Finished?
→ Process exits.
```

---

## 32. No Background Services

Upit should not include by default:

```text
daemon
background service
startup service
polling worker
file watcher
hidden GUI
resident event loop
tray process
```

If a future feature requires a background process, it should be implemented as a separate optional component rather than changing the default runtime model.

---

## 33. Resource Management

Every invocation should release resources immediately when finished.

Important resources:

- Open files
- HTTP response bodies
- Pipe readers/writers
- Request bodies
- Temporary files
- Network connections
- Contexts

Resources should not stay alive longer than necessary.

---

## 34. HTTP Response Limits

Do not allow arbitrary API responses to consume unlimited memory.

Response parsers should apply a maximum response size.

Uploader APIs normally return small JSON objects, so excessively large responses should be rejected safely.

---

## 35. Lazy Configuration Loading

Load only what is required for the current invocation.

Example:

```text
upit upload file.zip
 ↓
Load config.json
 ↓
Load selected uploader
 ↓
Load selected shortener only if enabled
```

Avoid unnecessary initialization.

---

## 36. Concurrency

Initial MVP should focus on one file per invocation.

```bash
upit upload file.zip
```

If multiple-file upload is added later, concurrency must be limited.

Example future model:

```text
Max concurrent uploads: 2–4
```

This prevents unnecessary:

- RAM usage
- Open file descriptors
- Network saturation

---

## 37. Error Model

Use structured errors.

Example:

```go
type UploadError struct {
    Stage      string
    StatusCode int
    Message    string
    Cause      error
}
```

Suggested stages:

```text
config
validation
request
network
upload
response
parse
shortener
clipboard
```

CLI example:

```text
Upload failed
Stage: response
HTTP: 401
Message: Invalid API token
```

Secrets must never appear in error output.

---

## 38. Proposed Project Structure (historical; not v0.3)


This package tree predates the implemented v0.3 seam. It is retained as roadmap context only; v0.3 uses the `internal/app` deep upload module described above.
```text
upit/
├── cmd/
│   ├── upit/
│   │   └── main.go
│   │
│   └── upit-desktop/
│       ├── main.go
│       └── bindings/
│           ├── upload.go
│           ├── config.go
│           └── settings.go
│
├── internal/
│   ├── app/
│   │   └── service.go
│   │
│   ├── engine/
│   │   ├── engine.go
│   │   └── result.go
│   │
│   ├── shortener/
│   │   ├── shortener.go
│   │   └── result.go
│   │
│   ├── uploader/
│   │   ├── uploader.go
│   │   ├── loader.go
│   │   └── validator.go
│   │
│   ├── request/
│   │   ├── builder.go
│   │   ├── multipart.go
│   │   ├── binary.go
│   │   ├── form.go
│   │   └── json.go
│   │
│   ├── response/
│   │   ├── parser.go
│   │   ├── json.go
│   │   ├── regex.go
│   │   ├── header.go
│   │   └── body.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   └── loader.go
│   │
│   └── clipboard/
│       └── clipboard.go
│
├── frontend/
│   ├── src/
│   └── package.json
│
├── schemas/
│   ├── config.schema.json
│   └── custom-uploader.schema.json
│
├── examples/
│   ├── config.example.json
│   └── custom-uploader.example.json
│
├── wails.json
├── go.mod
└── README.md
```

---

## 39. Dependency Boundaries (CLI v0.3; GUI future)

Main dependency rule:

```text
CLI ────────────────┐
                    │
Wails GUI ──────────┼──> App Service
                    │        │
Future integrations ┘        │
                             ▼
                       Core engines
```

Core packages must not depend on:

```text
CLI
Wails
WebView
GUI components
System tray
Explorer
Finder
Launchers
```

The CLI must not depend on:

```text
Wails
WebView
GTK
desktop environment
```

---

## 40. Build Strategy (v0.3 CLI only; desktop deferred)

CLI build:

```bash
go build ./cmd/upit
```

This build should not require Wails.

Desktop build:

```bash
wails3 build
```

or the equivalent project-specific Wails command.

The two build paths should remain independent.

---

## 41. Release Strategy (historical roadmap; not v0.3)

Recommended CLI/headless release artifacts:

```text
upit_1.0.0_windows_amd64.zip
upit_1.0.0_windows_arm64.zip

upit_1.0.0_darwin_amd64.tar.gz
upit_1.0.0_darwin_arm64.tar.gz

upit_1.0.0_linux_amd64.tar.gz
upit_1.0.0_linux_arm64.tar.gz
```

Desktop artifacts can be published separately:

```text
Upit_1.0.0_windows_x64.exe
Upit_1.0.0_macos_universal.dmg
Upit_1.0.0_linux_amd64.AppImage
```

Linux server users only need the headless CLI artifact.

---

## 42. Initial MVP (historical v0.1; v0.3 status follows above)

### v0.1

```text
Go core
versioned strict JSON configuration
named multipart Uploaders
RFC 9535 JSONPath response extraction
CLI plain and JSON output
optional nonfatal clipboard copying
streaming uploads
context cancellation and optional timeout
structured errors
OriginalURL and FinalURL (equal in v0.1)
```

Primary command:

```bash
upit upload file.zip
```

Expected stdout:

```text
https://files.example.com/file.zip
```

---

## 43. Roadmap and release status

### v0.1 — Upload Core

- Go upload engine
- Fixed `~/.config/upit/` path on every OS
- Strict version-1 `config.json` and `custom-uploader.json`
- Named multipart Uploaders with headers, query parameters, and static fields
- RFC 9535 JSONPath via a pinned implementation
- Plain and JSON CLI output
- `OriginalURL` and `FinalURL`, equal until shortening exists
- Optional nonfatal clipboard copying
- Streaming upload
- Context cancellation and optional timeout
- Structured errors and bounded 1 MiB responses
- Headless Linux support without Wails or WebView

### v0.2 — URL Shortener

- Named Shortener engine with configurable method, URL, headers, and query parameters
- `{input}` substitution in JSON request body object
- Preserves `OriginalURL` and updates `FinalURL`
- Safe runtime fallback to `OriginalURL` with warning on stderr
- Dedicated `custom-shortener.json` version 1 and global `config.json` version 2
- `--shortener <name>` and `--no-shorten` selection flags

### v0.3 — More Upload Protocols (implemented)

- `custom-uploader.json` is version 2; version 1 is rejected without migration or compatibility fallback
- Request Body Modes are `multipart`, `binary`, `form`, and `json`
- Multipart remains streamed file-part upload; binary streams exact file bytes with default or configured media type
- Form and JSON require valid UTF-8 input, exactly one full-string `{input}` value, and bounded-memory two-pass transformation
- Response Extractors are JSONPath `json`, response `header`, RE2 `regex`, and trimmed UTF-8 `body`
- URL extraction requires one absolute HTTP(S) URL; error extraction only improves diagnostics
- Only 2xx responses succeed; redirects and retries remain disabled; response bodies remain limited to 1 MiB
- Generated Content-Type values are owned by multipart, form, and JSON modes; binary may configure a valid media type
- Managed transport headers and incompatible mode/extractor fields are rejected during whole-document validation
- CLI flags, output/result fields, Shortener version 1, and the headless runtime remain unchanged

### v0.4 — Configuration Management (implemented)

- Non-interactive `config path`, `show`, `validate`, list, and targeted Global Configuration mutation commands
- Shared strict decoder with duplicate-key, unknown-field, deterministic-path, redacted diagnostics
- Safe staged Global Configuration publication with native POSIX/Windows replacement behavior
- Optional Shortener-document validation matrix and explicit clipboard state management
- Draft 2020-12 schemas for Global Configuration, Uploader, and Shortener documents
- Uploader and Shortener documents remain manually managed; no migration, editor, or provider adapter was added

### v0.5 — Wails 3 Desktop (planned)

- Keep the headless `upit` CLI and add a separate `upit-desktop` executable built with pinned Wails 3, Vue, TypeScript, Vite, and shadcn-vue.
- Provide Manual Upload for exactly one file selected by picker or drag-and-drop, with CLI-parity overrides, progress, cancellation, structured failures, manual retry, and Final URL results.
- Manage the three existing Global Configuration fields and the complete lifecycle of Uploaders and Shorteners through structured editors, strict validation, explicit valid-only saves, first-run setup, and raw repair for invalid documents.
- Reject stale saves and block rename or deletion of referenced definitions instead of adding lock files, automatic merge, or cross-document transaction journals.
- Run one desktop instance, never use a system tray or resident worker, and exit completely after resolving active uploads and unsaved edits when the window closes.
- Build native runnable desktop artifacts for Windows, macOS, and modern GTK4/WebKitGTK 6.0 Linux; defer installers, signing, auto-update, and operating-system integration to v1.0.

### v1.0 — Distribution and OS Integration

- Stable headless CLI
- Stable desktop frontend
- Windows Explorer integration
- macOS Finder integration
- Flow Launcher integration
- Wox integration
- Raycast integration
- Shell integration
- Linux server release artifacts
- Container-friendly CLI distribution

All integrations should invoke Upit on demand.

---

## 44. Final Architecture Summary (v0.4 current seam)

```text
┌──────────────────────────────┐
│ upit CLI (headless/server)   │
└──────────────┬───────────────┘
               ▼
┌──────────────────────────────┐
│ Application / deep upload    │
│ internal/app                 │
│ request bodies + extractors  │
└───────┬──────────┬───────────┘
        ▼          ▼
    Uploader    Shortener
        │          │
        └────┬─────┘
             ▼
      Clipboard and output
```

Headless/server flow:

```text
upit upload file
       ↓
Go core only
       ↓
Upload
       ↓
Optional shorten
       ↓
stdout
       ↓
Exit
```

Deferred desktop flow (not v0.3):

```text
Launch upit-desktop
       ↓
Wails starts
       ↓
Use GUI
       ↓
Close app
       ↓
Exit
```

Idle principle:

```text
No work
 ↓
No process
 ↓
No Upit background RAM usage
```

---

## 45. Key Decisions

```text
Project name:
Upit

Primary language:
Go

Primary interface:
CLI / headless

Desktop interface:
Deferred future Wails 3 application (not v0.3)

CLI binary:
upit

Desktop binary:
Future upit-desktop binary (not v0.3)

Wails policy:
Deferred with desktop; select and pin an exact version when desktop enters scope

Configuration directory:
~/.config/upit/

Global configuration:
~/.config/upit/config.json

Uploader definitions:
~/.config/upit/custom-uploader.json

Credentials:
Stored directly in custom-uploader.json

Configuration format:
JSON

Upload model:
Streaming

Runtime model:
On-demand / process-per-invocation

Background daemon:
No

System tray:
No

Hidden GUI process:
No

Headless Linux support:
Yes

GUI required for upload:
No

Clipboard required:
No

Primary CLI result channel:
stdout

URL shortener:
Optional post-processing step

Shortener failure:
Fallback to original URL by default

GUI behavior:
Deferred until desktop scope; not part of v0.3

Core architecture:
internal/app deep upload module → CLI (Wails desktop deferred)
```
