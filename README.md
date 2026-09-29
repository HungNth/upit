# Upit

Upit is a headless-first, cross-platform file uploader. It streams one file to a configurable HTTP endpoint, prints the resulting URL, and exits—no daemon, tray process, GUI, or resident worker.

## Features

- Streaming multipart and raw-binary uploads with bounded memory
- UTF-8 URL-encoded form and JSON template uploads with bounded-memory transformation
- Named, strictly validated HTTP Uploaders with four Request Body Modes
- JSONPath, response-header, regex, and raw-body Response Extractors
- Optional URL shortening via named, configurable Shorteners
- Plain URL or machine-readable JSON output
- Context cancellation and optional whole-invocation upload timeout
- Optional, nonfatal clipboard copying
- Windows, macOS, and headless Linux support

> [!NOTE]
> v0.3 is CLI-only. It adds Uploader body modes and response extractors without changing command flags, output fields, Shortener schema, or desktop scope.

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

Clipboard copying is disabled by default:

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

Architecture, domain language, and accepted decisions are documented under `docs/`, `CONTEXT.md`, and `.scratch/upit-v0.1/` through `.scratch/upit-v0.3/`.
