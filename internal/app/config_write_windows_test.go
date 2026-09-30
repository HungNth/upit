//go:build windows

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetDefaultUploaderUsesNativeWindowsReplacement(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":2,"defaultUploader":"old","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "next": {"request": {"method": "POST", "url": "https://upload.example.test", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, err := (Service{HomeDir: func() (string, error) { return home, nil }}).SetDefaultUploader("next")
	if err != nil || !changed {
		t.Fatalf("changed = %v; err = %v, want native replacement success", changed, err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{\n    \"version\": 2,\n    \"defaultUploader\": \"next\",\n    \"defaultShortener\": \"\",\n    \"copyToClipboard\": false\n}\n" {
		t.Errorf("config bytes = %q, want canonical replacement", string(data))
	}
	matches, err := filepath.Glob(filepath.Join(configDir, ".config.json.upit-tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Errorf("staged files remain after Windows replacement: %v", matches)
	}
}
