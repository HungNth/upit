package app

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type GlobalConfigurationDraft struct {
	Revision         string `json:"revision"`
	DefaultUploader  string `json:"defaultUploader"`
	DefaultShortener string `json:"defaultShortener"`
	CopyToClipboard  bool   `json:"copyToClipboard"`
}

type GlobalConfigurationEditorState struct {
	ConfigurationPath string   `json:"configurationPath"`
	Revision          string   `json:"revision"`
	DefaultUploader   string   `json:"defaultUploader"`
	DefaultShortener  string   `json:"defaultShortener"`
	CopyToClipboard   bool     `json:"copyToClipboard"`
	Uploaders         []string `json:"uploaders"`
	Shorteners        []string `json:"shorteners"`
	ShortenersPresent bool     `json:"shortenersPresent"`
}

func (s Service) LoadGlobalConfigurationEditor() (GlobalConfigurationEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return GlobalConfigurationEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	loaded, err := loadValidatedConfigurationSet(home)
	if err != nil {
		return GlobalConfigurationEditorState{}, err
	}
	path := filepath.Join(home, ".config", "upit", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return GlobalConfigurationEditorState{}, failuref("config", err, "read Global Configuration %s: %v", absolutePath(path), err)
	}
	return globalConfigurationEditorState(path, configurationRevision(data), loaded), nil
}

func (s Service) SaveGlobalConfiguration(draft GlobalConfigurationDraft) (GlobalConfigurationEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return GlobalConfigurationEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return GlobalConfigurationEditorState{}, failuref("config", err, "read Global Configuration %s: %v", absolutePath(path), err)
	}
	if draft.Revision == "" || configurationRevision(data) != draft.Revision {
		return GlobalConfigurationEditorState{}, failure("config", "Global Configuration changed on disk; refresh before saving", nil)
	}

	info, err := refuseUnsafeMutationTarget(path)
	if err != nil {
		return GlobalConfigurationEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return GlobalConfigurationEditorState{}, err
	}
	global.DefaultUploader = draft.DefaultUploader
	global.DefaultShortener = draft.DefaultShortener
	global.CopyToClipboard = draft.CopyToClipboard
	if _, err := validateConfigurationSetWithGlobal(home, global); err != nil {
		return GlobalConfigurationEditorState{}, err
	}
	if err := publishGlobalSettings(path, info, global); err != nil {
		return GlobalConfigurationEditorState{}, err
	}
	return s.LoadGlobalConfigurationEditor()
}

func globalConfigurationEditorState(path, revision string, loaded validatedConfigurationSet) GlobalConfigurationEditorState {
	state := GlobalConfigurationEditorState{
		ConfigurationPath: absolutePath(path),
		Revision:          revision,
		DefaultUploader:   loaded.global.DefaultUploader,
		DefaultShortener:  loaded.global.DefaultShortener,
		CopyToClipboard:   loaded.global.CopyToClipboard,
		Uploaders:         slices.Sorted(maps.Keys(loaded.uploaders.Uploaders)),
		ShortenersPresent: loaded.shortenersPresent,
	}
	if loaded.shortenersPresent {
		state.Shorteners = slices.Sorted(maps.Keys(loaded.shorteners.Shorteners))
	}
	return state
}

func configurationRevision(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
