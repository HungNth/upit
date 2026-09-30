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

func TestUploaderEditorPublishesAllBodyModesForCLIUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		switch r.URL.Path {
		case "/multipart":
			fmt.Fprint(w, "https://files.example.test/multipart")
		case "/binary":
			w.Header().Set("Location", "https://files.example.test/binary")
		case "/form":
			fmt.Fprint(w, "https://files.example.test/form")
		case "/json":
			fmt.Fprint(w, `{"url":"https://files.example.test/json"}`)
		case "/regex":
			fmt.Fprint(w, `URL=https://files.example.test/regex`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home, filePath := writeEditorCLIFixture(t)
	service := app.Service{HomeDir: func() (string, error) { return home, nil }}
	modes := []struct {
		name     string
		body     string
		path     string
		response app.UploaderExtractorDraft
		fields   []app.UploaderMapEntry
		dataJSON string
	}{
		{name: "edited-multipart", body: "multipart", path: "/multipart", response: app.UploaderExtractorDraft{Type: "body"}, fields: []app.UploaderMapEntry{{Key: "kind", Value: "file"}}},
		{name: "edited-binary", body: "binary", path: "/binary", response: app.UploaderExtractorDraft{Type: "header", Header: "Location"}},
		{name: "edited-form", body: "form", path: "/form", response: app.UploaderExtractorDraft{Type: "body"}, fields: []app.UploaderMapEntry{{Key: "content", Value: "{input}"}}},
		{name: "edited-json", body: "json", path: "/json", response: app.UploaderExtractorDraft{Type: "json", Path: "$.url"}, dataJSON: `{"content":"{input}"}`},
		{name: "edited-regex", body: "binary", path: "/regex", response: app.UploaderExtractorDraft{Type: "regex", Pattern: `(https://files.example.test/regex)`, Group: "1"}},
	}
	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			state, err := service.LoadUploaderEditor("")
			if err != nil {
				t.Fatal(err)
			}
			draft := state.Draft
			draft.Name = mode.name
			draft.Request.Method = http.MethodPost
			draft.Request.URL = server.URL + mode.path
			draft.Request.Body = mode.body
			draft.Request.FileField = "file"
			draft.Request.Fields = mode.fields
			draft.Request.DataJSON = mode.dataJSON
			draft.Response.URL = mode.response
			if mode.body != "multipart" {
				draft.Request.FileField = ""
			}
			if _, err := service.SaveUploaderEditor(draft); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			if code := cliRunner(home).Run(context.Background(), []string{"upload", filePath, "--uploader", mode.name}, &stdout, &stderr); code != 0 {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q", code, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "https://files.example.test/"+strings.TrimPrefix(mode.path, "/")) {
				t.Fatalf("stdout = %q, want edited %s result", stdout.String(), mode.body)
			}
		})
	}
}

func TestUploaderLifecycleKeepsDefaultCLIUploadWorking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		fmt.Fprint(w, "https://files.example.test/base")
	}))
	defer server.Close()

	home, filePath := writeLifecycleCLIFixture(t, server.URL)
	service := app.Service{HomeDir: func() (string, error) { return home, nil }}
	created, err := service.LoadUploaderEditor("")
	if err != nil {
		t.Fatal(err)
	}
	created.Draft.Name = "temporary"
	created.Draft.Request.Method = http.MethodPost
	created.Draft.Request.URL = server.URL + "/temporary"
	created.Draft.Request.Body = "binary"
	if _, err := service.SaveUploaderEditor(created.Draft); err != nil {
		t.Fatal(err)
	}
	temporary, err := service.LoadUploaderEditor("temporary")
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := service.RenameUploader(app.UploaderRenameDraft{Revision: temporary.Revision, OriginalName: "temporary", NewName: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteUploader(app.UploaderDeleteDraft{Revision: renamed.Revision, Name: "renamed"}); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", code, stdout.String(), stderr.String())
	}
	if stdout.String() != "https://files.example.test/base\n" {
		t.Fatalf("stdout = %q, want current default upload result", stdout.String())
	}
}

func writeEditorCLIFixture(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test/base","body":"binary"},"response":{"url":{"type":"body"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(home, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample upload contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, filePath
}

func writeLifecycleCLIFixture(t *testing.T, endpoint string) (string, string) {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	uploader := fmt.Sprintf(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":%q,"body":"binary"},"response":{"url":{"type":"body"}}}}}`, endpoint+"/base")
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(uploader), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(home, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample upload contents"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, filePath
}
