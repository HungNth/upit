package app

import (
	"errors"
	"os"
	"path/filepath"
)

type RepairDocumentKind string

const (
	RepairGlobalConfiguration RepairDocumentKind = "config"
	RepairUploaders           RepairDocumentKind = "uploaders"
	RepairShorteners          RepairDocumentKind = "shorteners"
)

type RepairDocumentDraft struct {
	Kind     RepairDocumentKind `json:"kind"`
	Revision string             `json:"revision"`
	Content  string             `json:"content"`
}

type RepairDocumentState struct {
	Kind     RepairDocumentKind `json:"kind"`
	Path     string             `json:"path"`
	Revision string             `json:"revision"`
	Content  string             `json:"content"`
	Present  bool               `json:"present"`
}

func (s Service) LoadRepairDocument(kind RepairDocumentKind) (RepairDocumentState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return RepairDocumentState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	path, err := repairDocumentPath(home, kind)
	if err != nil {
		return RepairDocumentState{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return RepairDocumentState{Kind: kind, Path: absolutePath(path)}, nil
	}
	if err != nil {
		return RepairDocumentState{}, failuref("config", err, "read repair document %s: %v", absolutePath(path), err)
	}
	return RepairDocumentState{Kind: kind, Path: absolutePath(path), Revision: configurationRevision(data), Content: string(data), Present: true}, nil
}

func (s Service) SaveRepairDocument(draft RepairDocumentDraft) (DesktopStartupState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return DesktopStartupState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	path, err := repairDocumentPath(home, draft.Kind)
	if err != nil {
		return DesktopStartupState{}, err
	}
	current, readErr := os.ReadFile(path)
	present := readErr == nil
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return DesktopStartupState{}, failuref("config", readErr, "read repair document %s: %v", absolutePath(path), readErr)
	}
	if configurationRevision(current) != draft.Revision {
		return DesktopStartupState{}, failure("config", "repair document changed on disk; reload before saving", nil)
	}
	var info os.FileInfo
	if present {
		info, err = refuseUnsafeDocumentTarget(path, "repair document")
		if err != nil {
			return DesktopStartupState{}, err
		}
	}
	canonical, err := validateAndCanonicalRepairDocument(draft.Kind, []byte(draft.Content), path)
	if err != nil {
		return DesktopStartupState{}, err
	}
	if err := validateRepairCrossDocument(home, draft.Kind, canonical); err != nil {
		return DesktopStartupState{}, err
	}
	if err := publishDocument(path, canonical, info, "repair document"); err != nil {
		return DesktopStartupState{}, err
	}
	return s.DesktopStartupState()
}

func repairDocumentPath(home string, kind RepairDocumentKind) (string, error) {
	base := filepath.Join(home, ".config", "upit")
	switch kind {
	case RepairGlobalConfiguration:
		return filepath.Join(base, "config.json"), nil
	case RepairUploaders:
		return filepath.Join(base, "custom-uploader.json"), nil
	case RepairShorteners:
		return filepath.Join(base, "custom-shortener.json"), nil
	default:
		return "", failure("validation", "unknown repair document", nil)
	}
}

func validateAndCanonicalRepairDocument(kind RepairDocumentKind, data []byte, path string) ([]byte, error) {
	switch kind {
	case RepairGlobalConfiguration:
		var document settings
		if _, err := decodeJSON(data, &document); err != nil {
			return nil, configurationReadFailure(path, err, "repair config.json and try again")
		}
		if document.Version != 2 {
			return nil, configLocation{file: absolutePath(path), path: "$.version"}.fail(nil, "config version = %d, want 2", document.Version)
		}
		if err := validateGlobalStructure(document, configLocation{file: absolutePath(path), path: "$"}); err != nil {
			return nil, err
		}
		return marshalSettings(document)
	case RepairUploaders:
		var document uploaderDocument
		if _, err := decodeJSON(data, &document); err != nil {
			return nil, configurationReadFailure(path, err, "repair custom-uploader.json and try again")
		}
		if document.Version != 2 {
			return nil, configLocation{file: absolutePath(path), path: "$.version"}.fail(nil, "custom uploader version = %d, want 2", document.Version)
		}
		if err := validateUploaderDocument(&document, path); err != nil {
			return nil, err
		}
		return marshalUploaderDocument(document)
	case RepairShorteners:
		var document shortenerDocument
		if _, err := decodeJSON(data, &document); err != nil {
			return nil, configurationReadFailure(path, err, "repair custom-shortener.json and try again")
		}
		if document.Version != 1 {
			return nil, configLocation{file: absolutePath(path), path: "$.version"}.fail(nil, "custom shortener version = %d, want 1", document.Version)
		}
		if err := validateShortenerDocument(&document, path); err != nil {
			return nil, err
		}
		return marshalShortenerDocument(document)
	default:
		return nil, failure("validation", "unknown repair document", nil)
	}
}

func validateRepairCrossDocument(home string, kind RepairDocumentKind, canonical []byte) error {
	switch kind {
	case RepairGlobalConfiguration:
		var global settings
		if _, err := decodeJSON(canonical, &global); err != nil {
			return err
		}
		uploaders, err := loadUploaderConfiguration(home)
		if err != nil {
			return err
		}
		if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, uploaders); err != nil {
			return err
		}
	case RepairUploaders:
		var uploaders uploaderDocument
		if _, err := decodeJSON(canonical, &uploaders); err != nil {
			return err
		}
		global, err := loadGlobalConfiguration(home)
		if err != nil {
			return err
		}
		if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, uploaders); err != nil {
			return err
		}
	case RepairShorteners:
		var shorteners shortenerDocument
		if _, err := decodeJSON(canonical, &shorteners); err != nil {
			return err
		}
		global, err := loadGlobalConfiguration(home)
		if err != nil {
			return err
		}
		if err := validateShortenerSetReferences(home, global, shorteners); err != nil {
			return err
		}
	}
	return nil
}
