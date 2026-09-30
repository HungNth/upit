package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortenerEditorCreatesOptionalDocumentAndPreservesMaskedValues(t *testing.T) {
	home := writeShortenerEditorFixture(t, false)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadShortenerEditor("")
	if err != nil {
		t.Fatal(err)
	}
	state.Draft.Name = "short"
	state.Draft.Request.Method = "POST"
	state.Draft.Request.URL = "https://short.example.test"
	state.Draft.Request.DataJSON = `{"target":"{input}"}`
	state.Draft.Response.URL = UploaderExtractorDraft{Type: "json", Path: "$.link"}
	saved, err := service.SaveShortenerEditor(state.Draft)
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Present || saved.Draft.Name != "short" {
		t.Fatalf("saved state = %#v, want present shortener", saved)
	}
	if err := service.ValidateConfiguration(); err != nil {
		t.Fatal(err)
	}
}

func TestShortenerEditorBlocksReferencedRenameDeleteAndRemovesFinalOptionalDocument(t *testing.T) {
	home := writeShortenerEditorFixture(t, true)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadShortenerEditor("short")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RenameShortener(ShortenerRenameDraft{Revision: state.Revision, OriginalName: "short", NewName: "renamed"}); err == nil || !strings.Contains(err.Error(), "default Shortener") {
		t.Fatalf("err = %v, want default-reference rename guard", err)
	}
	if _, err := service.DeleteShortener(ShortenerDeleteDraft{Revision: state.Revision, Name: "short"}); err == nil || !strings.Contains(err.Error(), "default Shortener") {
		t.Fatalf("err = %v, want default-reference delete guard", err)
	}

	globalPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(globalPath, []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err = service.LoadShortenerEditor("short")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteShortener(ShortenerDeleteDraft{Revision: state.Revision, Name: "short"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "upit", "custom-shortener.json")); !os.IsNotExist(err) {
		t.Fatalf("custom-shortener.json still exists; stat err = %v", err)
	}
}

func writeShortenerEditorFixture(t *testing.T, selected bool) string {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	defaultShortener := ""
	if selected {
		defaultShortener = "short"
	}
	global := `{"version":2,"defaultUploader":"base","copyToClipboard":false}`
	if selected {
		global = `{"version":2,"defaultUploader":"base","defaultShortener":"short","copyToClipboard":false}`
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":"https://upload.example.test/base","body":"binary"},"response":{"url":{"type":"body"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if selected {
		if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(`{"version":1,"shorteners":{"short":{"request":{"method":"POST","url":"https://short.example.test","data":{"target":"{input}"}},"response":{"url":{"type":"json","path":"$.link"}}}}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_ = defaultShortener
	return home
}
