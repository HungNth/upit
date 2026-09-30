package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

type schemaParityCase struct {
	name           string
	config         string
	uploader       string
	shortener      string
	configValid    bool
	uploaderValid  bool
	shortenerValid bool
	runtimeValid   bool
}

func TestSchemaParityWithConfigValidate(t *testing.T) {
	root := filepath.Join("..", "..")
	schemas := map[string]*jsonschema.Schema{
		"config":    compileSchema(t, filepath.Join(root, "schemas", "config.schema.json")),
		"uploader":  compileSchema(t, filepath.Join(root, "schemas", "custom-uploader.schema.json")),
		"shortener": compileSchema(t, filepath.Join(root, "schemas", "custom-shortener.schema.json")),
	}
	cases := []schemaParityCase{
		{
			name:        "valid complete Configuration Set",
			config:      `{"version":2,"defaultUploader":"test","defaultShortener":"selected","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			shortener:   `{"version":1,"shorteners":{"selected":{"request":{"method":"POST","url":"https://short.example.test","data":{"target":"{input}"}},"response":{"url":{"type":"json","path":"$.url"}}}}}`,
			configValid: true, uploaderValid: true, shortenerValid: true, runtimeValid: true,
		},
		{
			name:        "Global Configuration additional property",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false,"extra":true}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			configValid: false, uploaderValid: true, runtimeValid: false,
		},
		{
			name:        "multipart Uploader missing fileField",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"multipart"},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "Shortener non-JSON extractor",
			config:      `{"version":2,"defaultUploader":"test","defaultShortener":"selected","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			shortener:   `{"version":1,"shorteners":{"selected":{"request":{"method":"POST","url":"https://short.example.test","data":{"target":"{input}"}},"response":{"url":{"type":"header","header":"Location"}}}}}`,
			configValid: true, uploaderValid: true, shortenerValid: false, runtimeValid: false,
		},
		{
			name:        "Global Configuration missing required clipboard field",
			config:      `{"version":2,"defaultUploader":"test"}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			configValid: false, uploaderValid: true, runtimeValid: false,
		},
		{
			name:        "Uploader empty definition map",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "Uploader additional property",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary","extra":true},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "valid form Body Mode conditional",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"form","fields":{"payload":"{input}"}},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: true, runtimeValid: true,
		},
		{
			name:        "valid JSON Body Mode conditional",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"json","data":{"payload":"{input}"}},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: true, runtimeValid: true,
		},
		{
			name:        "JSON extractor incompatible field",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"json","path":"$.url","header":"Location"}}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "Shortener empty definition map",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			shortener:   `{"version":1,"shorteners":{}}`,
			configValid: true, uploaderValid: true, shortenerValid: false, runtimeValid: false,
		},
		{
			name:        "null is not an omitted optional extractor",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"},"error":null}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "Global Configuration null required field",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":null}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			configValid: false, uploaderValid: true, runtimeValid: false,
		},
		{
			name:        "form Body Mode rejects data",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"form","fields":{"payload":"{input}"},"data":{}},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "json Body Mode rejects fields",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"json","fields":{},"data":{"payload":"{input}"}},"response":{"url":{"type":"body"}}}}}`,
			configValid: true, uploaderValid: false, runtimeValid: false,
		},
		{
			name:        "Shortener response URL is required",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			shortener:   `{"version":1,"shorteners":{"selected":{"request":{"method":"POST","url":"https://short.example.test","data":{"target":"{input}"}},"response":{}}}}`,
			configValid: true, uploaderValid: true, shortenerValid: false, runtimeValid: false,
		},
		{
			name:        "Shortener request data is not nullable",
			config:      `{"version":2,"defaultUploader":"test","copyToClipboard":false}`,
			uploader:    `{"version":2,"uploaders":{"test":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`,
			shortener:   `{"version":1,"shorteners":{"selected":{"request":{"method":"POST","url":"https://short.example.test","data":null},"response":{"url":{"type":"json","path":"$.url"}}}}}`,
			configValid: true, uploaderValid: true, shortenerValid: false, runtimeValid: false,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			validateSchemaFixture(t, schemas["config"], test.config, test.configValid)
			validateSchemaFixture(t, schemas["uploader"], test.uploader, test.uploaderValid)
			if test.shortener != "" {
				validateSchemaFixture(t, schemas["shortener"], test.shortener, test.shortenerValid)
			}

			home := t.TempDir()
			configDir := filepath.Join(home, ".config", "upit")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			for name, data := range map[string]string{
				"config.json":          test.config,
				"custom-uploader.json": test.uploader,
			} {
				if err := os.WriteFile(filepath.Join(configDir, name), []byte(data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if test.shortener != "" {
				if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(test.shortener), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			var stdout, stderr bytes.Buffer
			exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
			if (exitCode == 0) != test.runtimeValid || (!test.runtimeValid && stdout.Len() != 0) {
				t.Errorf("config validate exit code = %d; stdout = %q; stderr = %q; want valid=%v", exitCode, stdout.String(), stderr.String(), test.runtimeValid)
			}
			if test.runtimeValid && stdout.String() != "Configuration is valid.\n" {
				t.Errorf("stdout = %q, want validation success", stdout.String())
			}
		})
	}
}

func compileSchema(t *testing.T, path string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("compile schema %s: %v", path, err)
	}
	return schema
}

func validateSchemaFixture(t *testing.T, schema *jsonschema.Schema, raw string, wantValid bool) {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	err := schema.Validate(value)
	if (err == nil) != wantValid {
		t.Fatalf("schema validation error = %v, want valid=%v", err, wantValid)
	}
}

func TestRepositoryExamplesPassSchemasAndConfigValidate(t *testing.T) {
	root := filepath.Join("..", "..")
	schemas := map[string]*jsonschema.Schema{
		"config":    compileSchema(t, filepath.Join(root, "schemas", "config.schema.json")),
		"uploader":  compileSchema(t, filepath.Join(root, "schemas", "custom-uploader.schema.json")),
		"shortener": compileSchema(t, filepath.Join(root, "schemas", "custom-shortener.schema.json")),
	}
	files := map[string]string{
		"config.json":           filepath.Join(root, "examples", "config.example.json"),
		"custom-uploader.json":  filepath.Join(root, "examples", "custom-uploader.example.json"),
		"custom-shortener.json": filepath.Join(root, "examples", "custom-shortener.example.json"),
	}
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for filename, source := range files {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		schemaName := map[string]string{
			"config.json":           "config",
			"custom-uploader.json":  "uploader",
			"custom-shortener.json": "shortener",
		}[filename]
		validateSchemaFixture(t, schemas[schemaName], string(data), true)
		if err := os.WriteFile(filepath.Join(configDir, filename), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Configuration is valid.\n" || stderr.Len() != 0 {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want example validation success", exitCode, stdout.String(), stderr.String())
	}
}
