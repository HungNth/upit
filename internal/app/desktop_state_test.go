package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDesktopStartupStateClassifiesMissingConfigurationDirectoryAsSetup(t *testing.T) {
	home := t.TempDir()
	state, err := (Service{HomeDir: func() (string, error) { return home, nil }}).DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupSetup {
		t.Fatalf("mode = %q, want %q", state.Mode, DesktopStartupSetup)
	}
	if state.ConfigurationPath != filepath.Join(home, ".config", "upit") {
		t.Fatalf("configuration path = %q, want %q", state.ConfigurationPath, filepath.Join(home, ".config", "upit"))
	}
	if state.Diagnostic != "" {
		t.Fatalf("diagnostic = %q, want empty setup diagnostic", state.Diagnostic)
	}
}

func TestDesktopStartupStateClassifiesEmptyConfigurationDirectoryAsSetup(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	state, err := (Service{HomeDir: func() (string, error) { return home, nil }}).DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupSetup {
		t.Fatalf("mode = %q, want %q", state.Mode, DesktopStartupSetup)
	}
	if state.Diagnostic != "" {
		t.Fatalf("diagnostic = %q, want empty setup diagnostic", state.Diagnostic)
	}
}

func TestDesktopStartupStateClassifiesPartialConfigurationDirectoryAsRepair(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"missing","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}

	state, err := (Service{HomeDir: func() (string, error) { return home, nil }}).DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupRepair {
		t.Fatalf("mode = %q, want %q", state.Mode, DesktopStartupRepair)
	}
	if !strings.Contains(state.Diagnostic, "custom-uploader.json") {
		t.Fatalf("diagnostic = %q, want missing Uploader document", state.Diagnostic)
	}
}

func TestDesktopStartupStateLoadsValidConfigurationDeterministically(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeDesktopStateFixture(t, configDir, `{
  "version": 2,
  "defaultUploader": "z-upload",
  "copyToClipboard": false
}`, `{
  "version": 2,
  "uploaders": {
    "z-upload": {"request": {"method": "POST", "url": "https://upload.example.test/z", "body": "binary"}, "response": {"url": {"type": "body"}}},
    "a-upload": {"request": {"method": "POST", "url": "https://upload.example.test/a", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`)

	state, err := (Service{HomeDir: func() (string, error) { return home, nil }}).DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupNormal {
		t.Fatalf("mode = %q, want %q; diagnostic = %q", state.Mode, DesktopStartupNormal, state.Diagnostic)
	}
	if state.DefaultUploader != "z-upload" || state.DefaultShortener != "" || state.CopyToClipboard {
		t.Fatalf("global state = %#v, want z-upload, no Shortener, clipboard disabled", state)
	}
	if want := []string{"a-upload", "z-upload"}; !reflect.DeepEqual(state.Uploaders, want) {
		t.Fatalf("uploaders = %#v, want %#v", state.Uploaders, want)
	}
	if len(state.Shorteners) != 0 || state.ShortenersPresent {
		t.Fatalf("shorteners = %#v, present = %v, want absent", state.Shorteners, state.ShortenersPresent)
	}
}

func TestDesktopStartupStateClassifiesInvalidConfigurationWithoutWriting(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeDesktopStateFixture(t, configDir, `{
  "version": 2,
  "defaultUploader": "broken",
  "copyToClipboard": false
}`, `{"version": 2, "uploaders": {}}`)
	configPath := filepath.Join(configDir, "config.json")
	uploaderPath := filepath.Join(configDir, "custom-uploader.json")
	beforeConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeUploader, err := os.ReadFile(uploaderPath)
	if err != nil {
		t.Fatal(err)
	}

	state, err := (Service{HomeDir: func() (string, error) { return home, nil }}).DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupRepair {
		t.Fatalf("mode = %q, want %q", state.Mode, DesktopStartupRepair)
	}
	if !strings.Contains(state.Diagnostic, filepath.Clean(uploaderPath)) || !strings.Contains(state.Diagnostic, "at least one Uploader") {
		t.Fatalf("diagnostic = %q, want uploader path and validation cause", state.Diagnostic)
	}
	afterConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	afterUploader, err := os.ReadFile(uploaderPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterConfig, beforeConfig) || !reflect.DeepEqual(afterUploader, beforeUploader) {
		t.Fatal("DesktopStartupState changed invalid Configuration Set bytes")
	}
}

func writeDesktopStateFixture(t *testing.T, configDir, global, uploaders string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(uploaders), 0o600); err != nil {
		t.Fatal(err)
	}
}
