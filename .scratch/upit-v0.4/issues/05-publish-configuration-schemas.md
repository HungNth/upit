# 05: Publish strict Draft 2020-12 configuration schemas

**What to build:** Publish hand-written Draft 2020-12 schemas at `schemas/config.schema.json`, `schemas/custom-uploader.schema.json`, and `schemas/custom-shortener.schema.json`, plus editor-association guidance and shared parity fixtures proving every schema-expressible rule agrees with production validation.

**Blocked by:** 02: Read and validate the Configuration Set from `upit config`

**Status:** resolved

**Parent specification:** `../spec.md`

**Testing seam:** Evaluate shared valid and invalid fixtures through both a conforming Draft 2020-12 test validator and the production `config validate` CLI seam. Keep runtime-only semantic cases in the production validator suite and document them through schema descriptions instead of tautological source-text tests.

**Demo path:** Associate the three schemas with their configuration filenames in an editor, observe autocomplete and validation for representative valid and invalid documents, then run `config validate` on the same fixtures and observe parity for every schema-expressible rule.

- [x] `schemas/config.schema.json`, `schemas/custom-uploader.schema.json`, and `schemas/custom-shortener.schema.json` describe the Global Configuration, Uploader document, and Shortener document respectively using Draft 2020-12.
- [x] Each schema declares the Draft 2020-12 meta-schema and omits `$id` until a stable schema-hosting URL exists.
- [x] User configuration documents do not gain a `$schema` property. The README documents filename associations scoped to an opened Configuration Set: `config.json` → `schemas/config.schema.json`, `custom-uploader.json` → `schemas/custom-uploader.schema.json`, and `custom-shortener.json` → `schemas/custom-shortener.schema.json`.
- [x] Unsupported-version diagnostics identify the matching published schema path now that the referenced artifact exists.
- [x] Fixed-shape objects reject additional properties and express required members, current version constants, primitive types, enums, non-empty definition maps, and supported conditional fields.
- [x] Uploader schemas express existing v0.3 Request Body Mode and Response Extractor field compatibility without adding protocol capabilities.
- [x] The Shortener schema preserves the existing JSON-only Response Extractor and current request-template shape.
- [x] Schema descriptions explain runtime-only rules that Draft 2020-12 cannot express exactly, including printable/trimmed Unicode names, duplicate keys, HTTP and header semantics, JSONPath and regular-expression compilation, MIME parsing, recursive placeholder counts, permissions, and cross-document references.
- [x] Go remains the production execution authority; runtime code does not load schemas and gains no production schema-validator dependency.
- [x] Shared parity fixtures cover versions, required fields, strict properties, non-empty maps, field presence versus null, Request Body Mode conditionals, Response Extractor conditionals, and every other representable rule.
- [x] Runtime-only fixtures cover duplicate keys, name semantics, case-insensitive headers, HTTP/URL rules, expression compilation, placeholder counts, permissions, and default references without pretending the schema enforces them.
- [x] All repository example configuration documents and representative canonical documents pass both applicable schema validation and production configuration validation.
- [x] Tests assert behavior and validation outcomes, not mere schema file presence, non-empty content, or expected source strings.

## Comments

- 2026-10-03: `go test -race ./...` and `go vet ./...` passed, including `TestSchemaParityWithConfigValidate` and `TestRepositoryExamplesPassSchemasAndConfigValidate`. No production schema-validator dependency or generated schema system was added.
