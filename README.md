# Upit

Upit is a headless-first, cross-platform file uploader. It streams one file to a configurable HTTP endpoint, prints the resulting URL, and exits—no daemon, tray process, GUI, or resident worker.

## Features

- Streaming multipart uploads with bounded memory use
- Named, configurable HTTP Uploaders
- RFC 9535 JSONPath response extraction
- Plain URL or machine-readable JSON output
- Context cancellation and optional upload timeout
- Optional, nonfatal clipboard copying
- Windows, macOS, and headless Linux support

> [!NOTE]
> v0.1 is CLI-only. URL shortening, additional upload body types, JSON Schema, configuration commands, and the Wails desktop application are later roadmap items.

## Requirements

- Go 1.23 or newer to build from source
- Clipboard command only when clipboard copying is enabled:
    - macOS: `pbcopy`
    - Windows: `clip`
    - Linux: `wl-copy`, `xclip`, or `xsel`

The upload path itself has no desktop or clipboard dependency.

## Build

```bash
go build -o bin/upit ./cmd/upit
```

Show CLI help:

```bash
./bin/upit --help
```

## Configuration

Upit always reads user configuration from:

```text
~/.config/upit/
├── config.json
└── custom-uploader.json
```

This path is intentionally the same on Linux, macOS, and Windows. Copy the version-1 examples:

```bash
mkdir -p ~/.config/upit
cp examples/config.example.json ~/.config/upit/config.json
cp examples/custom-uploader.example.json ~/.config/upit/custom-uploader.json
chmod 600 ~/.config/upit/custom-uploader.json
```

On Unix, Upit refuses to use `custom-uploader.json` when group or other permission bits are present.

### `config.json`

```json
{
    "version": 1,
    "defaultUploader": "personal",
    "copyToClipboard": false
}
```

### `custom-uploader.json`

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

Configuration is strict: unknown fields, unsupported versions, invalid request settings, and invalid JSONPath expressions fail before any network request.

## Usage

Upload with the configured default Uploader:

```bash
./bin/upit upload file.zip
```

Select another Uploader:

```bash
./bin/upit upload file.zip --uploader personal
```

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

Clipboard copying is disabled by default:

```bash
./bin/upit upload image.png --clipboard
./bin/upit upload image.png --no-clipboard
```

A clipboard failure does not invalidate a successful upload. Upit still exits 0, preserves stdout, and writes a warning to stderr.

## Output contract

| Outcome                | stdout                     | stderr                                 | Exit code |
| ---------------------- | -------------------------- | -------------------------------------- | --------: |
| Success                | Final URL, or success JSON | Clipboard warning only when applicable |         0 |
| Runtime/upload failure | Empty                      | Plain or JSON error                    |         1 |
| Invalid usage          | Empty                      | Usage error                            |         2 |
| Interrupted upload     | Empty                      | Plain or JSON cancellation error       |       130 |

Upit never writes progress or logs to stdout.

## Security

- Credentials are stored directly in `custom-uploader.json`.
- Do not commit real configuration or credentials.
- Request headers, query values, and multipart fields are treated as sensitive in diagnostics.
- Redirects and automatic retries are disabled.
- Only HTTP 2xx responses can succeed.
- Response bodies are limited to 1 MiB.

## Development

```bash
go test ./...
go vet ./...
```

Architecture, domain language, and accepted decisions are documented under `docs/`, `CONTEXT.md`, and `.scratch/upit-v0.1/`.
