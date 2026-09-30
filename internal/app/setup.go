package app

import (
	"errors"
	"os"
	"path/filepath"
)

func (s Service) CreateInitialConfigurationSet(globalDraft GlobalConfigurationDraft, uploaderDraft UploaderEditorDraft) (DesktopStartupState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return DesktopStartupState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	configurationPath := filepath.Join(home, ".config", "upit")
	info, err := os.Stat(configurationPath)
	if err == nil {
		if !info.IsDir() {
			return DesktopStartupState{}, failure("config", "Configuration Set path is not a directory: "+absolutePath(configurationPath), nil)
		}
		absent, absentErr := configurationSetDocumentsAbsent(configurationPath)
		if absentErr != nil {
			return DesktopStartupState{}, failuref("config", absentErr, "inspect Configuration Set documents in %s: %v", absolutePath(configurationPath), absentErr)
		}
		if !absent {
			return DesktopStartupState{}, failure("config", "Configuration Set already exists; edit the existing documents instead of running Setup", nil)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return DesktopStartupState{}, failuref("config", err, "inspect Configuration Set directory %s: %v", absolutePath(configurationPath), err)
	}

	global := settings{
		Version:             2,
		DefaultUploader:     globalDraft.DefaultUploader,
		DefaultShortener:    globalDraft.DefaultShortener,
		CopyToClipboard:     globalDraft.CopyToClipboard,
		hasVersion:          true,
		hasDefaultUploader:  true,
		hasDefaultShortener: true,
		hasCopyToClipboard:  true,
	}
	candidate, err := uploaderFromEditorDraft(uploaderDraft, nil)
	if err != nil {
		return DesktopStartupState{}, err
	}
	uploaders := uploaderDocument{Version: 2, Uploaders: map[string]uploader{uploaderDraft.Name: candidate}}
	if err := validateUploaderDocument(&uploaders, filepath.Join(configurationPath, "custom-uploader.json")); err != nil {
		return DesktopStartupState{}, err
	}
	if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, uploaders); err != nil {
		return DesktopStartupState{}, err
	}
	globalData, err := marshalSettings(global)
	if err != nil {
		return DesktopStartupState{}, err
	}
	uploaderData, err := marshalUploaderDocument(uploaders)
	if err != nil {
		return DesktopStartupState{}, err
	}

	parent := filepath.Dir(configurationPath)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return DesktopStartupState{}, failuref("config", err, "create Configuration Set parent %s: %v", absolutePath(parent), err)
	}
	stagedDir, err := os.MkdirTemp(parent, ".upit-setup-*")
	if err != nil {
		return DesktopStartupState{}, failuref("config", err, "stage initial Configuration Set in %s: %v", absolutePath(parent), err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(stagedDir)
		}
	}()
	if err := writeSetupDocument(filepath.Join(stagedDir, "config.json"), globalData); err != nil {
		return DesktopStartupState{}, err
	}
	if err := writeSetupDocument(filepath.Join(stagedDir, "custom-uploader.json"), uploaderData); err != nil {
		return DesktopStartupState{}, err
	}
	if info != nil {
		if err := os.Remove(configurationPath); err != nil {
			return DesktopStartupState{}, failuref("config", err, "replace empty Configuration Set directory %s: %v", absolutePath(configurationPath), err)
		}
	}
	if err := os.Rename(stagedDir, configurationPath); err != nil {
		return DesktopStartupState{}, failuref("config", err, "publish initial Configuration Set: %v", err)
	}
	published = true
	return s.DesktopStartupState()
}

func writeSetupDocument(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return failuref("config", err, "create Setup document %s: %v", absolutePath(path), err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return failuref("config", err, "write Setup document %s: %v", absolutePath(path), err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return failuref("config", err, "synchronize Setup document %s: %v", absolutePath(path), err)
	}
	if err := file.Close(); err != nil {
		return failuref("config", err, "close Setup document %s: %v", absolutePath(path), err)
	}
	return nil
}
