package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/HungNth/upit/internal/app"
)

func TestSetupCreatesConfigurationSetForValidationAndDefaultCLIUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		fmt.Fprint(w, "https://files.example.test/setup")
	}))
	defer server.Close()

	home := t.TempDir()
	filePath := filepath.Join(home, "setup.txt")
	if err := os.WriteFile(filePath, []byte("setup upload"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := app.Service{HomeDir: func() (string, error) { return home, nil }}
	created, err := service.CreateInitialConfigurationSet(app.GlobalConfigurationDraft{
		DefaultUploader: "setup-uploader",
		CopyToClipboard: false,
	}, app.UploaderEditorDraft{
		Name:     "setup-uploader",
		Request:  app.UploaderRequestDraft{Method: "POST", URL: server.URL, Body: "binary"},
		Response: app.UploaderResponseDraft{URL: app.UploaderExtractorDraft{Type: "body"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Mode != app.DesktopStartupNormal || created.DefaultUploader != "setup-uploader" {
		t.Fatalf("startup state = %#v, want normal setup-uploader state", created)
	}

	var validateOut, validateErr bytes.Buffer
	if code := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &validateOut, &validateErr); code != 0 || validateOut.String() != "Configuration is valid.\n" || validateErr.Len() != 0 {
		t.Fatalf("config validate = %d; stdout = %q; stderr = %q", code, validateOut.String(), validateErr.String())
	}

	var stdout, stderr bytes.Buffer
	if code := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr); code != 0 {
		t.Fatalf("upload exit code = %d; stdout = %q; stderr = %q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "https://files.example.test/setup\n" || stderr.Len() != 0 {
		t.Fatalf("upload stdout = %q; stderr = %q", stdout.String(), stderr.String())
	}
}
