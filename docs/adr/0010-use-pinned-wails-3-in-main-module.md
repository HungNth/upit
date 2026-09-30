# Use pinned Wails 3 in the main Go module

Upit v0.5 pins Wails `v3.0.0-beta.26` and raises the repository's single Go module from Go 1.23 to Go 1.25 rather than creating a separate desktop module or adopting Wails 2. Wails 3's API is documented as stable but remains pre-release, so upgrades are deliberate and changelog-driven; Wails services stay thin, and the CLI and application packages remain free of Wails imports so headless builds retain their existing runtime boundary.
