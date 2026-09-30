//go:build !windows

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairStartupReportsCredentialPermissionFailure(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "custom-uploader.json")
	if err := os.WriteFile(path, []byte(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test","body":"binary"},"response":{"url":{"type":"body"}}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupRepair || !strings.Contains(state.Diagnostic, "permission") {
		t.Fatalf("state = %#v, want credential permission Repair diagnostic", state)
	}
}
