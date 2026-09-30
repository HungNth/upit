package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUploaderEditorLoadsDeterministicMaskedDraft(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}

	state, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	if state.DefaultUploader != "first" || !reflect.DeepEqual(state.Uploaders, []string{"first"}) {
		t.Fatalf("state = %#v, want first default and sorted list", state)
	}
	if state.Draft.Request.Headers[0].Value != redactedConfigurationValue || !state.Draft.Request.Headers[0].Sensitive {
		t.Fatalf("header = %#v, want masked sensitive value", state.Draft.Request.Headers[0])
	}
	if state.Draft.OriginalName != "first" || state.Draft.Name != "first" {
		t.Fatalf("draft names = %q/%q, want first/first", state.Draft.OriginalName, state.Draft.Name)
	}
}

func TestUploaderEditorSavesNewUploaderAndPreservesExistingDefinition(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadUploaderEditor("")
	if err != nil {
		t.Fatal(err)
	}

	saved, err := service.SaveUploaderEditor(UploaderEditorDraft{
		Revision: state.Revision,
		Name:     "new",
		Request: UploaderRequestDraft{
			Method: "PUT",
			URL:    "https://upload.example.test/new",
			Body:   "binary",
		},
		Response: UploaderResponseDraft{URL: UploaderExtractorDraft{Type: "body"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(saved.Uploaders, []string{"first", "new"}) {
		t.Fatalf("uploaders = %#v, want first/new", saved.Uploaders)
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "upit", "custom-uploader.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"first"`) || !strings.Contains(string(data), `"new"`) {
		t.Fatalf("saved uploader document = %s, want both definitions", data)
	}
}

func TestUploaderEditorClearsIncompatibleFieldsWhenSavingBinaryMode(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	state.Draft.Request.Body = "binary"
	state.Draft.Request.FileField = "stale-file-field"
	state.Draft.Request.Fields = []UploaderMapEntry{{Key: "stale", Value: "value"}}
	state.Draft.Request.DataJSON = `{malformed`
	if _, err := service.SaveUploaderEditor(state.Draft); err != nil {
		t.Fatal(err)
	}
	saved, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Draft.Request.FileField != "" || len(saved.Draft.Request.Fields) != 0 || saved.Draft.Request.DataJSON != "" {
		t.Fatalf("saved binary draft retained incompatible fields: %#v", saved.Draft.Request)
	}
}

func TestUploaderEditorRejectsStaleSaveWithoutOverwritingDocument(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	external, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	external = append(external, '\n')
	if err := os.WriteFile(path, external, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = service.SaveUploaderEditor(state.Draft)
	if err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("err = %v, want stale-save diagnostic", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(external) {
		t.Fatal("stale Uploader save overwrote the external document")
	}
}

func writeUploaderEditorFixture(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"first","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "first": {
      "request": {"method": "POST", "url": "https://upload.example.test/first", "headers": {"Authorization": "secret"}, "body": "binary"},
      "response": {"url": {"type": "body"}}
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}
