package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploaderLifecycleRenamesAndDeletesUnreferencedDefinition(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	newState, err := service.LoadUploaderEditor("")
	if err != nil {
		t.Fatal(err)
	}
	newState.Draft.Name = "temporary"
	newState.Draft.Request.Method = "POST"
	newState.Draft.Request.URL = "https://upload.example.test/temporary"
	newState.Draft.Request.Body = "binary"
	if _, err := service.SaveUploaderEditor(newState.Draft); err != nil {
		t.Fatal(err)
	}

	temporary, err := service.LoadUploaderEditor("temporary")
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := service.RenameUploader(UploaderRenameDraft{Revision: temporary.Revision, OriginalName: "temporary", NewName: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Draft.Name != "renamed" {
		t.Fatalf("renamed draft = %q, want renamed", renamed.Draft.Name)
	}

	remaining, err := service.DeleteUploader(UploaderDeleteDraft{Revision: renamed.Revision, Name: "renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if remaining.Draft.Name != "first" {
		t.Fatalf("remaining draft = %q, want adjacent/default first Uploader", remaining.Draft.Name)
	}
	if err := service.ValidateConfiguration(); err != nil {
		t.Fatalf("Configuration Set after rename/delete = %v", err)
	}
}

func TestUploaderLifecycleRejectsInvalidDuplicateAndStaleRename(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	created, err := service.LoadUploaderEditor("")
	if err != nil {
		t.Fatal(err)
	}
	created.Draft.Name = "temporary"
	created.Draft.Request.Method = "POST"
	created.Draft.Request.URL = "https://upload.example.test/temporary"
	created.Draft.Request.Body = "binary"
	if _, err := service.SaveUploaderEditor(created.Draft); err != nil {
		t.Fatal(err)
	}
	temporary, err := service.LoadUploaderEditor("temporary")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RenameUploader(UploaderRenameDraft{Revision: temporary.Revision, OriginalName: "temporary", NewName: "first"}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("err = %v, want duplicate-name rejection", err)
	}
	if _, err := service.RenameUploader(UploaderRenameDraft{Revision: temporary.Revision, OriginalName: "temporary", NewName: " invalid"}); err == nil || !strings.Contains(err.Error(), "valid non-empty name") {
		t.Fatalf("err = %v, want invalid-name rejection", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RenameUploader(UploaderRenameDraft{Revision: temporary.Revision, OriginalName: "temporary", NewName: "renamed"}); err == nil || !strings.Contains(err.Error(), "changed on disk") {
		t.Fatalf("err = %v, want stale rename rejection", err)
	}
}

func TestUploaderLifecycleRejectsFinalUploaderWhenReferenceIsAlreadyInvalid(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":2,"defaultUploader":"other","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeleteUploader(UploaderDeleteDraft{Revision: state.Revision, Name: "first"}); err == nil || !strings.Contains(err.Error(), "final Uploader") {
		t.Fatalf("err = %v, want final-Uploader rejection", err)
	}
}

func TestUploaderLifecycleBlocksDefaultAndFinalDeletion(t *testing.T) {
	home := writeUploaderEditorFixture(t)
	service := Service{HomeDir: func() (string, error) { return home, nil }}
	state, err := service.LoadUploaderEditor("first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RenameUploader(UploaderRenameDraft{Revision: state.Revision, OriginalName: "first", NewName: "renamed"}); err == nil || !strings.Contains(err.Error(), "default Uploader") {
		t.Fatalf("err = %v, want default-rename guard", err)
	}
	if _, err := service.DeleteUploader(UploaderDeleteDraft{Revision: state.Revision, Name: "first"}); err == nil || !strings.Contains(err.Error(), "default Uploader") {
		t.Fatalf("err = %v, want default-delete guard", err)
	}
}
