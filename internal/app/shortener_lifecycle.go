package app

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

func (s Service) RenameShortener(draft ShortenerRenameDraft) (ShortenerEditorState, error) {
	home, document, path, info, err := s.loadShortenerMutationState(draft.Revision)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	if draft.OriginalName == "" || draft.NewName == "" || !validDefinitionName(draft.NewName) {
		return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "Shortener rename requires a valid non-empty name")
	}
	candidate, ok := document.Shorteners[draft.OriginalName]
	if !ok {
		return ShortenerEditorState{}, failure("config", "selected Shortener no longer exists; refresh before renaming", nil)
	}
	if draft.OriginalName == draft.NewName {
		return s.LoadShortenerEditor(draft.OriginalName)
	}
	if global.DefaultShortener == draft.OriginalName {
		return ShortenerEditorState{}, configLocation{file: absolutePath(filepath.Join(home, ".config", "upit", "config.json")), path: "$.defaultShortener"}.fail(nil, "cannot rename the default Shortener; clear or change the default first")
	}
	if _, exists := document.Shorteners[draft.NewName]; exists {
		return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "Shortener %q already exists", draft.NewName)
	}
	delete(document.Shorteners, draft.OriginalName)
	document.Shorteners[draft.NewName] = candidate
	if err := validateShortenerDocument(&document, path); err != nil {
		return ShortenerEditorState{}, err
	}
	if err := validateShortenerSetReferences(home, global, document); err != nil {
		return ShortenerEditorState{}, err
	}
	canonical, err := marshalShortenerDocument(document)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	if err := publishDocument(path, canonical, info, "Shortener document"); err != nil {
		return ShortenerEditorState{}, err
	}
	return s.LoadShortenerEditor(draft.NewName)
}

func (s Service) DeleteShortener(draft ShortenerDeleteDraft) (ShortenerEditorState, error) {
	home, document, path, info, err := s.loadShortenerMutationState(draft.Revision)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	if _, ok := document.Shorteners[draft.Name]; !ok {
		return ShortenerEditorState{}, failure("config", "selected Shortener no longer exists; refresh before deleting", nil)
	}
	if global.DefaultShortener == draft.Name {
		return ShortenerEditorState{}, configLocation{file: absolutePath(filepath.Join(home, ".config", "upit", "config.json")), path: "$.defaultShortener"}.fail(nil, "cannot delete the default Shortener; clear or change the default first")
	}
	if len(document.Shorteners) == 1 {
		if err := os.Remove(path); err != nil {
			return ShortenerEditorState{}, failuref("config", err, "remove optional Shortener document %s: %v", absolutePath(path), err)
		}
		return s.LoadShortenerEditor("")
	}
	delete(document.Shorteners, draft.Name)
	if err := validateShortenerDocument(&document, path); err != nil {
		return ShortenerEditorState{}, err
	}
	if err := validateShortenerSetReferences(home, global, document); err != nil {
		return ShortenerEditorState{}, err
	}
	canonical, err := marshalShortenerDocument(document)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	if err := publishDocument(path, canonical, info, "Shortener document"); err != nil {
		return ShortenerEditorState{}, err
	}
	next := slices.Sorted(maps.Keys(document.Shorteners))[0]
	return s.LoadShortenerEditor(next)
}

func (s Service) loadShortenerMutationState(revision string) (string, shortenerDocument, string, os.FileInfo, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return "", shortenerDocument{}, "", nil, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", shortenerDocument{}, "", nil, failuref("config", err, "read Shortener document %s: %v", absolutePath(path), err)
	}
	if revision == "" || configurationRevision(data) != revision {
		return "", shortenerDocument{}, "", nil, failure("config", "Shortener document changed on disk; refresh before saving", nil)
	}
	info, err := refuseUnsafeDocumentTarget(path, "Shortener document")
	if err != nil {
		return "", shortenerDocument{}, "", nil, err
	}
	document, err := loadShortenerConfiguration(home)
	if err != nil {
		return "", shortenerDocument{}, "", nil, err
	}
	return home, document, path, info, nil
}

func validateShortenerSetReferences(home string, global settings, document shortenerDocument) error {
	globalPath := filepath.Join(home, ".config", "upit", "config.json")
	globalLocation := configLocation{file: absolutePath(globalPath), path: "$"}
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return err
	}
	uploaders, err := loadUploaderConfiguration(home)
	if err != nil {
		return err
	}
	if _, ok := uploaders.Uploaders[global.DefaultUploader]; !ok {
		return globalLocation.child("defaultUploader").fail(nil, "default Uploader %q does not exist; choose one listed by config list-uploaders", global.DefaultUploader)
	}
	if global.DefaultShortener != "" {
		if _, ok := document.Shorteners[global.DefaultShortener]; !ok {
			return globalLocation.child("defaultShortener").fail(nil, "default Shortener %q does not exist; choose one listed by config list-shorteners", global.DefaultShortener)
		}
	}
	return nil
}

func marshalShortenerDocument(document shortenerDocument) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "    ")
	if err != nil {
		return nil, failuref("config", err, "encode Shortener document: %v", err)
	}
	return append(data, '\n'), nil
}
