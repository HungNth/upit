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
