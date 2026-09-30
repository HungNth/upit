package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairDocumentPublishesValidUploaderAndReclassifiesStartup(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := []byte(`{"version":2,"uploaders":{}}`)
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), bad, 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	loaded, err := service.LoadRepairDocument(RepairUploaders)
	if err != nil || loaded.Content != string(bad) {
		t.Fatalf("loaded = %#v, err = %v", loaded, err)
	}
	valid := `{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`
	state, err := service.SaveRepairDocument(RepairDocumentDraft{Kind: RepairUploaders, Revision: loaded.Revision, Content: valid})
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupNormal {
		t.Fatalf("startup mode = %q, want normal; diagnostic = %q", state.Mode, state.Diagnostic)
	}
	if err := service.ValidateConfiguration(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(state.Diagnostic, "secret") {
		t.Fatal("repair diagnostic leaked a sensitive marker")
	}
}

func TestRepairDocumentRejectsStaleSave(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "custom-uploader.json")
	original := []byte(`{"version":2,"uploaders":{}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	loaded, err := service.LoadRepairDocument(RepairUploaders)
	if err != nil {
		t.Fatal(err)
	}
	external := []byte(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`)
	if err := os.WriteFile(path, external, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveRepairDocument(RepairDocumentDraft{Kind: RepairUploaders, Revision: loaded.Revision, Content: string(external)}); err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("err = %v, want stale repair rejection", err)
	}
}

func TestRepairDocumentRejectsCrossDocumentDanglingDefaultBeforePublish(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	original := []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`)
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	loaded, err := service.LoadRepairDocument(RepairGlobalConfiguration)
	if err != nil {
		t.Fatal(err)
	}
	invalid := `{"version":2,"defaultUploader":"missing","copyToClipboard":false}`
	if _, err := service.SaveRepairDocument(RepairDocumentDraft{Kind: RepairGlobalConfiguration, Revision: loaded.Revision, Content: invalid}); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v, want dangling-reference rejection", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("cross-document invalid repair changed config.json")
	}
}

func TestRepairDocumentRejectsMalformedContentWithoutPublishing(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "custom-uploader.json")
	original := []byte(`{"version":2,"uploaders":{}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	loaded, err := service.LoadRepairDocument(RepairUploaders)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveRepairDocument(RepairDocumentDraft{Kind: RepairUploaders, Revision: loaded.Revision, Content: "{malformed"}); err == nil || !strings.Contains(err.Error(), "line") {
		t.Fatalf("err = %v, want malformed JSON diagnostic", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatal("malformed Repair content was published")
	}
}

func TestRepairStartupReportsUnsupportedVersion(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":99,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupRepair || !strings.Contains(state.Diagnostic, "want 2") {
		t.Fatalf("state = %#v, want unsupported-version Repair diagnostic", state)
	}
}

func TestRepairStartupReportsMalformedJSONLocation(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{\n  \\\"version\\\": 2,\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupRepair || !strings.Contains(state.Diagnostic, "line") || !strings.Contains(state.Diagnostic, "column") {
		t.Fatalf("state = %#v, want malformed JSON line/column diagnostic", state)
	}
}
