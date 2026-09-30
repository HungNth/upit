package cli_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/HungNth/upit/internal/app"
)

func TestRepairStateDisablesValidUploadUntilRepair(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{"version":2,"uploaders":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(home, "repair.txt")
	if err := os.WriteFile(filePath, []byte("repair"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := app.Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.DesktopStartupState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != app.DesktopStartupRepair {
		t.Fatalf("mode = %q, want repair", state.Mode)
	}
	var stdout, stderr bytes.Buffer
	if code := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit code = %d, want config failure; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if called.Load() || !strings.Contains(stderr.String(), "at least one Uploader") {
		t.Fatalf("called=%v stderr=%q, want no request and Repair diagnostic", called.Load(), stderr.String())
	}
}
