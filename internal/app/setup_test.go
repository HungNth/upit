package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateInitialConfigurationSetPublishesUsableDocuments(t *testing.T) {
	home := t.TempDir()
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.CreateInitialConfigurationSet(GlobalConfigurationDraft{
		DefaultUploader: "first",
		CopyToClipboard: false,
	}, UploaderEditorDraft{
		Name:     "first",
		Request:  UploaderRequestDraft{Method: "POST", URL: "https://upload.example.test", Body: "binary"},
		Response: UploaderResponseDraft{URL: UploaderExtractorDraft{Type: "body"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.Mode != DesktopStartupNormal || state.DefaultUploader != "first" {
		t.Fatalf("state = %#v, want normal first Uploader", state)
	}
	if err := service.ValidateConfiguration(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.json", "custom-uploader.json"} {
		if _, err := os.Stat(filepath.Join(home, ".config", "upit", name)); err != nil {
			t.Fatalf("created %s: %v", name, err)
		}
	}
}

func TestCreateInitialConfigurationSetRejectsInvalidDraftWithoutCreatingDocuments(t *testing.T) {
	home := t.TempDir()
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	_, err := service.CreateInitialConfigurationSet(GlobalConfigurationDraft{
		DefaultUploader: "missing",
		CopyToClipboard: false,
	}, UploaderEditorDraft{
		Name:     "first",
		Request:  UploaderRequestDraft{Method: "POST", URL: "not-a-url", Body: "binary"},
		Response: UploaderResponseDraft{URL: UploaderExtractorDraft{Type: "body"}},
	})
	if err == nil || !strings.Contains(err.Error(), "absolute HTTP(S)") {
		t.Fatalf("err = %v, want invalid URL validation", err)
	}
	if _, statErr := os.Stat(filepath.Join(home, ".config", "upit")); !os.IsNotExist(statErr) {
		t.Fatalf("Configuration Set directory exists after rejected Setup: %v", statErr)
	}
}
