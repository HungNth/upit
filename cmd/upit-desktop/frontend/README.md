# Upit Desktop frontend

This Vue 3, TypeScript, and Vite project belongs to the Wails 3 executable in `cmd/upit-desktop`. It stays below that Go package because `main.go` embeds `frontend/dist` with a package-relative `//go:embed` pattern.

Run every command below from the repository root.

## Setup and build

Install frontend dependencies after cloning or changing `package-lock.json`:

```bash
make setup-desktop
```

Build the frontend and then the desktop binary:

```bash
make build-desktop
```

The frontend step type-checks Vue and writes bundled JavaScript and CSS to `dist/`. The Go step then embeds that output into `bin/upit-desktop`. The generated `dist/` files are committed so the Go package remains buildable from a clean checkout.

## Wails bindings

`bindings/` contains generated JavaScript wrappers and model declarations for exported Go interfaces. CSS and frontend-only JavaScript changes do not require binding generation.

Regenerate bindings only after changing an exported Go service method or model used by the frontend. This target requires the pinned `wails3` CLI; ordinary frontend builds do not.

```bash
make generate-desktop-bindings
make build-desktop
```

The generated `bindings/` files are committed and must not be edited manually.

## Native dialog regression

Rename and Delete use in-app Vue dialogs, not `window.prompt` or `window.confirm`: the pinned macOS WebView returns immediately from those JavaScript APIs without presenting a usable native dialog. The in-app dialogs preserve exact names for Go validation, retain diagnostics after failure, close after successful publication, and manage initial focus, Tab containment, Escape cancellation, and focus restoration.

To check the real WebView, build the production frontend, then compile a separate smoke binary with `go build -tags mcp`. Launch it with an isolated temporary `HOME` containing valid documents and unreferenced disposable definitions named `upit-smoke-uploader` and `upit-smoke-shortener`; the disposable Uploader must start with a JSON URL extractor and a valid non-empty JSONPath. Set `WAILS_MCP_PORT=19099` and wait for the normal Configuration Set state; do not run against a real user Configuration Set. MCP is compiled out of ordinary production builds.

```bash
node cmd/upit-desktop/frontend/scripts/native-dialog-smoke.mjs \
    http://127.0.0.1:19099 upit-smoke-uploader upit-smoke-shortener
```

This destructive check edits, renames, and deletes only the supplied `upit-smoke-*` definitions. It proves JSON-to-body transitions for both URL and error extractors save without incompatible hidden fields, verifies both lifecycle dialogs through native WebView events and Go publication, rejects whitespace-bearing names without normalization, checks successful close and Delete confirmation, and asserts accessible control names, validation associations, Tab containment, and initial/restored focus. It does not prove Finder/Explorer integration, OS picker/drop delivery, signing, notarization, or other-platform runtime behavior.
