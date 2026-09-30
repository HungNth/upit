package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalConfigurationEditorLoadsAndSavesCanonicalValues(t *testing.T) {
	home := writeGlobalEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}

	state, err := service.LoadGlobalConfigurationEditor()
	if err != nil {
		t.Fatal(err)
	}
	if state.DefaultUploader != "first" || state.DefaultShortener != "" || state.CopyToClipboard {
		t.Fatalf("state = %#v, want first/no Shortener/clipboard disabled", state)
	}
	if len(state.Uploaders) != 2 || state.Uploaders[0] != "first" || state.Uploaders[1] != "second" {
		t.Fatalf("uploaders = %#v, want sorted definitions", state.Uploaders)
	}
	if !state.ShortenersPresent || len(state.Shorteners) != 1 || state.Shorteners[0] != "short" {
		t.Fatalf("shorteners = %#v, present = %v, want short/present", state.Shorteners, state.ShortenersPresent)
	}

	saved, err := service.SaveGlobalConfiguration(GlobalConfigurationDraft{
		Revision:         state.Revision,
		DefaultUploader:  "second",
		DefaultShortener: "short",
		CopyToClipboard:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision == state.Revision || saved.DefaultUploader != "second" || saved.DefaultShortener != "short" || !saved.CopyToClipboard {
		t.Fatalf("saved state = %#v, want updated values and revision", saved)
	}

	data, err := os.ReadFile(filepath.Join(home, ".config", "upit", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{
    "version": 2,
    "defaultUploader": "second",
    "defaultShortener": "short",
    "copyToClipboard": true
}
`
	if string(data) != want {
		t.Fatalf("config bytes = %q, want canonical %q", data, want)
	}
}

func TestGlobalConfigurationEditorRejectsStaleSaveWithoutOverwritingDisk(t *testing.T) {
	home := writeGlobalEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadGlobalConfigurationEditor()
	if err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(home, ".config", "upit", "config.json")
	external := []byte(`{
    "version": 2,
    "defaultUploader": "first",
    "defaultShortener": "short",
    "copyToClipboard": true
}
`)
	if err := os.WriteFile(configPath, external, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = service.SaveGlobalConfiguration(GlobalConfigurationDraft{
		Revision:         state.Revision,
		DefaultUploader:  "second",
		DefaultShortener: "short",
		CopyToClipboard:  false,
	})
	if err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("err = %v, want stale-save diagnostic", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(external) {
		t.Fatalf("disk bytes changed after stale save: %q", data)
	}
}

func TestGlobalConfigurationEditorRejectsInvalidDraftWithoutPublishing(t *testing.T) {
	home := writeGlobalEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadGlobalConfigurationEditor()
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SaveGlobalConfiguration(GlobalConfigurationDraft{
		Revision:         state.Revision,
		DefaultUploader:  "missing",
		DefaultShortener: "",
		CopyToClipboard:  true,
	})
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v, want missing-Uploader validation", err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("invalid Global Configuration draft was published")
	}
}

func writeGlobalEditorFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "first",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "first": {"request": {"method": "POST", "url": "https://upload.example.test/first", "body": "binary"}, "response": {"url": {"type": "body"}}},
    "second": {"request": {"method": "POST", "url": "https://upload.example.test/second", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(`{
  "version": 1,
  "shorteners": {
    "short": {
      "request": {"method": "POST", "url": "https://short.example.test", "data": {"target": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.link"}}
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}
