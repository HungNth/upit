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
	"strings"
	"testing"

	"github.com/HungNth/upit/internal/app"
)

func TestShortenerEditorPublishesFinalURLAndFallbackThroughCLI(t *testing.T) {
	for _, test := range []struct {
		name        string
		shortenFail bool
		wantURL     string
		wantWarn    bool
	}{
		{name: "final URL", wantURL: "https://short.example.test/final"},
		{name: "fallback", shortenFail: true, wantURL: "https://files.example.test/original", wantWarn: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				switch r.URL.Path {
				case "/upload":
					fmt.Fprint(w, "https://files.example.test/original")
				case "/shorten":
					if test.shortenFail {
						w.WriteHeader(http.StatusBadGateway)
						fmt.Fprint(w, `{"error":"shortener failed"}`)
						return
					}
					fmt.Fprint(w, `{"link":"https://short.example.test/final"}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			home, filePath := writeShortenerCLIFixture(t, server.URL)
			service := app.Service{HomeDir: func() (string, error) { return home, nil }}
			shortenerState, err := service.LoadShortenerEditor("")
			if err != nil {
				t.Fatal(err)
			}
			shortenerState.Draft.Name = "edited"
			shortenerState.Draft.Request.Method = http.MethodPost
			shortenerState.Draft.Request.URL = server.URL + "/shorten"
			shortenerState.Draft.Request.DataJSON = `{"target":"{input}"}`
			shortenerState.Draft.Response.URL = app.UploaderExtractorDraft{Type: "json", Path: "$.link"}
			if _, err := service.SaveShortenerEditor(shortenerState.Draft); err != nil {
				t.Fatal(err)
			}
			globalState, err := service.LoadGlobalConfigurationEditor()
			if err != nil {
				t.Fatal(err)
			}
			globalDraft := app.GlobalConfigurationDraft{
				Revision:         globalState.Revision,
				DefaultUploader:  globalState.DefaultUploader,
				DefaultShortener: "edited",
				CopyToClipboard:  globalState.CopyToClipboard,
			}
			if _, err := service.SaveGlobalConfiguration(globalDraft); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			if code := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q", code, stdout.String(), stderr.String())
			}
			if stdout.String() != test.wantURL+"\n" {
				t.Fatalf("stdout = %q, want %q", stdout.String(), test.wantURL+"\n")
			}
			if test.wantWarn != strings.Contains(stderr.String(), "Warning: shorten URL:") {
				t.Fatalf("stderr = %q, warning presence = %v, want %v", stderr.String(), strings.Contains(stderr.String(), "Warning: shorten URL:"), test.wantWarn)
			}
		})
	}
}

func writeShortenerCLIFixture(t *testing.T, endpoint string) (string, string) {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader := fmt.Sprintf(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":%q,"body":"binary"},"response":{"url":{"type":"body"}}}}}`, endpoint+"/upload")
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(uploader), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(home, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample upload contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, filePath
}
