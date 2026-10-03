# Upit

Upit is a headless-first, cross-platform file uploader. `upit` streams one file to a configurable HTTP endpoint, prints the resulting URL, and exits. The separately launched `upit-desktop` adds local Configuration Set management and Manual Upload without changing the CLI runtime model.

## Features

- Streaming multipart and raw-binary uploads with bounded memory
- UTF-8 URL-encoded form and JSON template uploads with bounded-memory transformation
- Named, strictly validated HTTP Uploaders with four Request Body Modes
- JSONPath, response-header, regex, and raw-body Response Extractors
- Optional URL shortening via named, configurable Shorteners
- Non-interactive Configuration Management commands for reading, validating, and changing Global Configuration
- Upit Desktop control surface with Setup, Repair, and structured Global Configuration, Uploader, and Shortener editing
- Manual Upload for exactly one regular file from a native picker or drag-and-drop, with per-upload Uploader, Shortener, clipboard, and timeout choices
- Manual Upload progress, cancellation, explicit Retry, Final URL copying, and nonfatal Shortener or clipboard warnings
- Hand-written Draft 2020-12 schemas for the three Configuration Set documents
- File Manager Upload helper for exactly one regular file from Windows File Explorer or macOS Finder Services
- Windows 11 x64 primary context-menu package route through a native `IExplorerCommand`
- macOS 14+ Apple Silicon Finder Service package route through a background-only `NSServices` helper
- Privacy-minimized native progress, cancellation, Copy, Retry, and configuration recovery actions
- Protected-tag-only Windows and macOS package signing, notarization, checksums, install, and uninstall automation
- Plain URL or machine-readable JSON CLI output
- Context cancellation and optional whole-invocation upload timeout
- Optional, nonfatal clipboard copying
- Windows, macOS, and headless Linux CLI support

> [!NOTE]
> Upit Desktop pins Wails `v3.0.0-beta.26`, a pre-release dependency. It is a separate executable: `upit` never starts Wails, a WebView, a GUI, tray process, or resident worker.

## Requirements

- Go 1.25 or newer to build from source
- Node.js and npm for `upit-desktop` frontend builds
- `wails3` `v3.0.0-beta.26` only when regenerating desktop bindings after exported Go interface changes
- Desktop runtime:
    - Windows: WebView2
    - macOS: system WebKit
    - Linux: GTK4 and WebKitGTK 6.0
- Clipboard command only when clipboard copying is enabled:
    - macOS: `pbcopy`
    - Windows: `clip`
    - Linux: `wl-copy`, `xclip`, or `xsel`

The CLI upload path itself has no desktop or clipboard dependency.

## Build

Build the CLI:

```bash
make build
./bin/upit --help
```

Build Upit Desktop and the Wails-free File Manager Upload helper from the repository root:

```bash
# First clone, or after package-lock.json changes.
make setup-desktop

# Type-checks/bundles the frontend, builds the helper, then embeds the frontend in upit-desktop.
make build-desktop
./bin/upit-desktop
./bin/upit-file-manager.exe <one-regular-file>   # Windows
./bin/upit-file-manager <one-regular-file>       # macOS/Linux
```

On a macOS 14+ Apple Silicon host, build and package the Finder Service integration with:

```bash
make build-macos
make package-macos MACOS_VERSION=0.7.0
```

`make package-macos` creates an unsigned verification DMG unless protected-tag signing and notarization variables are supplied. See [`packaging/macos/README.md`](packaging/macos/README.md). `make generate-desktop-bindings` is only needed after changing an exported desktop Go method or model.

## Upit Desktop and File Manager Upload

Upit Desktop is one single-instance window that follows the operating system light/dark preference. It uses the same fixed `~/.config/upit/` Configuration Set and Wails-free application rules as the CLI. Manual Upload accepts one regular file, supports a native picker or drag-and-drop, blocks start while Configuration Set edits are dirty, reloads configuration before execution, and retains selected inputs for explicit Retry after failure or cancellation.

File Manager Upload is a separate Wails-free one-shot operation. The Windows 11 x64 package registers `Upload with Upit` in File Explorer's primary menu through a signed package-identity `IExplorerCommand`. The macOS 14+ Apple Silicon package registers the same action through a background-only `NSServices` helper under Finder Services or Quick Actions. Both adapters validate only native selection shape and pass the untrusted selection to the shared helper; Configuration Set loading, upload, notifications, clipboard actions, retry state, cancellation, and privacy-safe progress remain in the shared application module.

File Manager Upload always attempts to copy the Final URL, regardless of the Global Configuration clipboard preference. Copy failure preserves the completed upload and offers `Copy Final URL` recovery without uploading again. The clipboard preference applies only to CLI and Manual Upload.

Desktop includes a File Manager Integration area even while the Configuration Set requires Setup or Repair. Linux reports `Not supported on Linux` and exposes no registration or Repair actions; Manual Upload remains independent.

On macOS, Desktop inspects the installed outer bundle, nested Finder Service and worker, matching bundle versions, and public Launch Services discovery. Repair is explicit; `Registered` does not assert Services enablement or Finder menu visibility. Use Keyboard Settings guidance to enable `Upload with Upit`. Before deleting the bundle, choose `Prepare to Remove Upit`, confirm unregistering the Service, then move `Upit.app` to Trash after Desktop closes. The Configuration Set is retained.

On Windows, Repair requires the retained signed registration package, trusted matching signatures, and external payload hashes bound to that signed package. Missing, unsigned, untrusted, or mismatched repair material produces `Reinstall Upit`. Repair runs only after an explicit user action, in the current-user context; unsigned verification builds cannot advertise production Repair.

Native Setup/Repair navigation can be checked against a Desktop built with `-tags mcp` and a disposable `HOME`: run `node cmd/upit-desktop/frontend/scripts/native-integration-smoke.mjs http://127.0.0.1:19109 setup` (or `repair`) with `WAILS_MCP_PORT=19109`. This checks the actual webview and backend; it does not replace protected Finder/Explorer release proof.

The v0.6 release target is Windows 11 x64 and the v0.7 release target is macOS 14+ Apple Silicon. Windows 10 classic verbs, Windows ARM64, Intel Macs, older macOS, Mac App Store sandboxed extensions, portable registration, batch selection, queues, upload history, and resident workers remain out of scope. Signing, notarization, and publishing are permitted only from protected SemVer tags; ordinary changes receive unsigned verification artifacts.

## Configuration

Upit always reads user configuration from:

```text
~/.config/upit/
├── config.json
├── custom-uploader.json
└── custom-shortener.json (optional)
```

This path is intentionally the same on Linux, macOS, and Windows. Copy the examples:

```bash
mkdir -p ~/.config/upit
cp examples/config.example.json ~/.config/upit/config.json
cp examples/custom-uploader.example.json ~/.config/upit/custom-uploader.json
cp examples/custom-shortener.example.json ~/.config/upit/custom-shortener.json
chmod 600 ~/.config/upit/custom-uploader.json ~/.config/upit/custom-shortener.json
```

On Unix, Upit refuses to run when `custom-uploader.json` or `custom-shortener.json` (when used) has group or other permission bits present.

### Editor schema associations

Configuration documents stay free of a `$schema` member. In an editor, associate each filename with the repository schema while working inside a Configuration Set:

| Filename | Draft 2020-12 schema |
| --- | --- |
| `config.json` | `schemas/config.schema.json` |
| `custom-uploader.json` | `schemas/custom-uploader.schema.json` |
| `custom-shortener.json` | `schemas/custom-shortener.schema.json` |

The schemas cover structural rules expressible in Draft 2020-12. Go remains the production authority for protocol semantics, expression compilation, permissions, and cross-document references.

### `config.json`

```json
{
    "version": 2,
    "defaultUploader": "personal",
    "defaultShortener": "shortened",
    "copyToClipboard": false
}
```

### `custom-uploader.json`

```json
{
    "version": 2,
    "uploaders": {
        "personal": {
            "request": {
                "method": "POST",
                "url": "https://upload.example.com/api/upload",
                "headers": {},
                "query": {},
                "body": "multipart",
                "fileField": "file",
                "fields": {}
            },
            "response": {
                "url": {
                    "type": "json",
                    "path": "$.data.url"
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

#### Uploader v2 protocol rules

Request Body Modes:

| Mode        | Configuration                                                     | Request body                                             |
| ----------- | ----------------------------------------------------------------- | -------------------------------------------------------- |
| `multipart` | `fileField` plus optional string `fields`                         | Streamed multipart file part and literal fields          |
| `binary`    | No `fileField`, `fields`, or `data`                               | Exact file bytes; defaults to `application/octet-stream` |
| `form`      | Flat string `fields` with exactly one full-string `{input}`       | UTF-8 file text as `application/x-www-form-urlencoded`   |
| `json`      | Object `data` with exactly one full-string `{input}` string value | UTF-8 file text JSON-escaped inside the object           |

`multipart`, `form`, and `json` own their generated `Content-Type` and reject a configured override. Binary accepts a valid configured media type. Form and JSON validate the complete file as UTF-8 before the request, then stream the transformed body; empty UTF-8 files are valid. Placeholder substitution is value-only and full-string-only—there is no filename, base64, key, substring, header, query, or URL substitution.

Response Extractors:

| Type     | Fields                                    | Input                                          |
| -------- | ----------------------------------------- | ---------------------------------------------- |
| `json`   | `path`                                    | RFC 9535 JSONPath selecting one JSON string    |
| `header` | `header`                                  | One non-empty response-header value            |
| `regex`  | `pattern`, optional `group` (default `0`) | First match in the bounded UTF-8 response body |
| `body`   | none                                      | Trimmed non-empty UTF-8 response body          |

URL extraction always requires one absolute HTTP(S) URL. Error extraction uses the same extractor types only for diagnostics; it never creates success or falls back to another extractor. Only 2xx responses succeed, redirects are not followed, each upload is attempted once, and response bodies are limited to 1 MiB.

The Uploader document is a clean version-2 cutover: version 1 is rejected and is not migrated. Invalid configuration is reported before any endpoint request. Failure stages remain `config`, `validation`, `request`, `network`, `response`, and `parse`; parent cancellation exits 130 while ordinary upload failures exit 1. Endpoint-provided Uploader and Shortener messages are trimmed and normalized to one printable line before configured values are redacted. Shortener fallback remains post-upload behavior and does not turn an Uploader failure into success.

For an existing multipart Uploader, the migration is mechanical: change only the document version from `1` to `2`. Keep its method, URL, headers, query, `fileField`, fields, JSONPath URL extractor, and optional JSONPath error extractor unchanged.

### `custom-shortener.json`

```json
{
    "version": 1,
    "shorteners": {
        "shortened": {
            "request": {
                "method": "POST",
                "url": "https://kutt.it/api/v2/links",
                "headers": {
                    "X-API-KEY": "YOUR_API_KEY"
                },
                "query": {},
                "data": {
                    "target": "{input}"
                }
            },
            "response": {
                "url": {
                    "type": "json",
                    "path": "$.link"
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

Configuration is strict: unknown fields, unsupported versions, invalid request settings, and invalid JSONPath expressions fail before any network request.

### Configuration Management commands

Configuration commands use stdout for success, stderr for configuration failures, and exit 2 for invalid command usage. They do not support `--json`, prompts, editors, or alternate Configuration Set locations.

```bash
./bin/upit config path
./bin/upit config show
./bin/upit config list-uploaders
./bin/upit config list-shorteners
./bin/upit config validate
```

`config path` prints the fixed absolute `~/.config/upit/` directory even when it does not exist. `show` prints `Default Uploader`, `Default Shortener` (`none` when clear), and `Copy to Clipboard`. List commands validate only their own document; an absent optional Shortener document prints `No Shorteners configured.`.

The complete validator requires `config.json` and `custom-uploader.json`. A Shortener document may be absent only when no default Shortener is selected; when present, every Shortener is validated. A selected default requires a matching valid Shortener.

Only Global Configuration fields have mutation commands:

```bash
./bin/upit config set-default-uploader <name>
./bin/upit config set-default-shortener <name>
./bin/upit config clear-default-shortener
./bin/upit config enable-clipboard
./bin/upit config disable-clipboard
```

Mutations preserve unrelated Global Configuration fields, print literal confirmations, validate before publishing, and do not create missing directories or documents. Uploader and Shortener creation, editing, renaming, deletion, migration, and interactive management remain manual or out of scope.

Mutations refuse a symlinked or supported Windows reparse-point Global Configuration during point-in-time preflight. Writes then use a uniquely staged `.config.json.upit-tmp-*` file, complete writes, sync, close, and the native replacement primitive. No backup or lock file is created; concurrent successful writers are last-completed-write-wins. POSIX mode bits are preserved. Windows replacement does not promise POSIX rollback or extended ACL/stream preservation. If Windows publication state is ambiguous, Upit retains the staged recovery file and reports its absolute path. This is not a power-loss durability guarantee.

## Usage

Upload with the configured default Uploader:

```bash
./bin/upit upload file.zip
```

Select another Uploader:

```bash
./bin/upit upload file.zip --uploader personal
```

### URL shortening

Shorten with a named Shortener:

```bash
./bin/upit upload file.zip --shortener shortened
```

Disable shortening for one upload when a default Shortener is configured:

```bash
./bin/upit upload file.zip --no-shorten
```

If shortening fails at runtime, Upit falls back to the Original URL as the Final URL, writes a warning to stderr (`Warning: shorten URL: ...`), and exits 0. If shortening is interrupted via Ctrl-C, Upit exits 130.

Flags may appear before or after the file path.

### JSON output

```bash
./bin/upit upload file.zip --json
```

```json
{
    "success": true,
    "originalUrl": "https://files.example.com/file.zip",
    "finalUrl": "https://files.example.com/file.zip"
}
```

Failures use stderr and a nonzero exit code. With `--json`, stderr contains:

```json
{
    "success": false,
    "stage": "response",
    "message": "invalid token",
    "statusCode": 401
}
```

### Timeout and cancellation

Uploads have no default overall deadline. Set one when needed:

```bash
./bin/upit upload backup.tar.gz --timeout 10m
```

Press Ctrl-C to cancel an in-flight upload.

### Clipboard

CLI clipboard copying is disabled by default:

```bash
./bin/upit upload image.png --clipboard
./bin/upit upload image.png --no-clipboard
```

A clipboard failure does not invalidate a successful upload. Upit still exits 0, preserves stdout, and writes a warning to stderr.

## Output contract

| Outcome                | stdout                     | stderr                                           | Exit code |
| ---------------------- | -------------------------- | ------------------------------------------------ | --------: |
| Success                | Final URL, or success JSON | Shortener/clipboard warning only when applicable |         0 |
| Runtime/upload failure | Empty                      | Plain or JSON error                              |         1 |
| Invalid usage          | Empty                      | Usage error                                      |         2 |
| Interrupted upload     | Empty                      | Plain or JSON cancellation error                 |       130 |

Upit never writes progress or logs to stdout.

## Security

- Credentials are stored directly in `custom-uploader.json` and `custom-shortener.json`.
- Do not commit real configuration or credentials.
- Request headers, query values, multipart/form fields, static JSON values, and Shortener request bodies are treated as sensitive in diagnostics.
- Redirects and automatic retries are disabled.
- Only HTTP 2xx responses can succeed.
- Response bodies are limited to 1 MiB.

## Development

```bash
go test ./...
go vet ./...
```

Architecture, domain language, accepted decisions, and roadmap specifications are documented under `docs/`, `CONTEXT.md`, and `.scratch/upit-v0.1/` through `.scratch/upit-v0.7/`.
