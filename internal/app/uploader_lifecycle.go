package app

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type UploaderRenameDraft struct {
	Revision     string `json:"revision"`
	OriginalName string `json:"originalName"`
	NewName      string `json:"newName"`
}

type UploaderDeleteDraft struct {
	Revision string `json:"revision"`
	Name     string `json:"name"`
}

func (s Service) RenameUploader(draft UploaderRenameDraft) (UploaderEditorState, error) {
	home, document, path, info, err := s.loadUploaderMutationState(draft.Revision)
	if err != nil {
		return UploaderEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if draft.OriginalName == "" || draft.NewName == "" || !validDefinitionName(draft.NewName) {
		return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "Uploader rename requires a valid non-empty name")
	}
	candidate, ok := document.Uploaders[draft.OriginalName]
	if !ok {
		return UploaderEditorState{}, failure("config", "selected Uploader no longer exists; refresh before renaming", nil)
	}
	if draft.OriginalName == draft.NewName {
		return s.LoadUploaderEditor(draft.OriginalName)
	}
	if global.DefaultUploader == draft.OriginalName {
		return UploaderEditorState{}, configLocation{file: absolutePath(filepath.Join(home, ".config", "upit", "config.json")), path: "$.defaultUploader"}.fail(nil, "cannot rename the default Uploader; choose another default first")
	}
	if _, exists := document.Uploaders[draft.NewName]; exists {
		return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "Uploader %q already exists", draft.NewName)
	}
	delete(document.Uploaders, draft.OriginalName)
	document.Uploaders[draft.NewName] = candidate
	if err := validateUploaderDocument(&document, path); err != nil {
		return UploaderEditorState{}, err
	}
	if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, document); err != nil {
		return UploaderEditorState{}, err
	}
	canonical, err := marshalUploaderDocument(document)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if err := publishDocument(path, canonical, info, "Uploader document"); err != nil {
		return UploaderEditorState{}, err
	}
	return s.LoadUploaderEditor(draft.NewName)
}

func (s Service) DeleteUploader(draft UploaderDeleteDraft) (UploaderEditorState, error) {
	home, document, path, info, err := s.loadUploaderMutationState(draft.Revision)
	if err != nil {
		return UploaderEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if _, ok := document.Uploaders[draft.Name]; !ok {
		return UploaderEditorState{}, failure("config", "selected Uploader no longer exists; refresh before deleting", nil)
	}
	if global.DefaultUploader == draft.Name {
		return UploaderEditorState{}, configLocation{file: absolutePath(filepath.Join(home, ".config", "upit", "config.json")), path: "$.defaultUploader"}.fail(nil, "cannot delete the default Uploader; choose another default first")
	}
	if len(document.Uploaders) == 1 {
		return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "cannot delete the final Uploader")
	}
	delete(document.Uploaders, draft.Name)
	if err := validateUploaderDocument(&document, path); err != nil {
		return UploaderEditorState{}, err
	}
	if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, document); err != nil {
		return UploaderEditorState{}, err
	}
	canonical, err := marshalUploaderDocument(document)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if err := publishDocument(path, canonical, info, "Uploader document"); err != nil {
		return UploaderEditorState{}, err
	}
	next := global.DefaultUploader
	if _, ok := document.Uploaders[next]; !ok {
		next = slices.Sorted(maps.Keys(document.Uploaders))[0]
	}
	return s.LoadUploaderEditor(next)
}

func (s Service) loadUploaderMutationState(revision string) (string, uploaderDocument, string, os.FileInfo, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return "", uploaderDocument{}, "", nil, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", uploaderDocument{}, "", nil, failuref("config", err, "read Uploader document %s: %v", absolutePath(path), err)
	}
	if revision == "" || configurationRevision(data) != revision {
		return "", uploaderDocument{}, "", nil, failure("config", "Uploader document changed on disk; refresh before saving", nil)
	}
	info, err := refuseUnsafeDocumentTarget(path, "Uploader document")
	if err != nil {
		return "", uploaderDocument{}, "", nil, err
	}
	document, err := loadUploaderConfiguration(home)
	if err != nil {
		return "", uploaderDocument{}, "", nil, err
	}
	return home, document, path, info, nil
}
