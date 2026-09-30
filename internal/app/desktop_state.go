package app

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type DesktopStartupMode string

const (
	DesktopStartupNormal DesktopStartupMode = "normal"
	DesktopStartupSetup  DesktopStartupMode = "setup"
	DesktopStartupRepair DesktopStartupMode = "repair"
)

type DesktopStartupState struct {
	Mode              DesktopStartupMode `json:"mode"`
	ConfigurationPath string             `json:"configurationPath"`
	DefaultUploader   string             `json:"defaultUploader"`
	DefaultShortener  string             `json:"defaultShortener"`
	CopyToClipboard   bool               `json:"copyToClipboard"`
	Uploaders         []string           `json:"uploaders"`
	Shorteners        []string           `json:"shorteners"`
	ShortenersPresent bool               `json:"shortenersPresent"`
	Diagnostic        string             `json:"diagnostic"`
}

type validatedConfigurationSet struct {
	global            settings
	uploaders         uploaderDocument
	shorteners        shortenerDocument
	shortenersPresent bool
}

func (s Service) DesktopStartupState() (DesktopStartupState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return DesktopStartupState{}, failuref("config", err, "resolve user home directory: %v", err)
	}

	configurationPath := absolutePath(filepath.Join(home, ".config", "upit"))
	state := DesktopStartupState{
		ConfigurationPath: configurationPath,
	}
	info, err := os.Stat(configurationPath)
	if errors.Is(err, os.ErrNotExist) {
		state.Mode = DesktopStartupSetup
		return state, nil
	}
	if err != nil {
		state.Mode = DesktopStartupRepair
		state.Diagnostic = failuref("config", err, "inspect Configuration Set directory %s: %v", configurationPath, err).Error()
		return state, nil
	}
	if !info.IsDir() {
		state.Mode = DesktopStartupRepair
		state.Diagnostic = failure("config", "Configuration Set path is not a directory: "+configurationPath, nil).Error()
		return state, nil
	}
	absent, err := configurationSetDocumentsAbsent(configurationPath)
	if err != nil {
		state.Mode = DesktopStartupRepair
		state.Diagnostic = failuref("config", err, "inspect Configuration Set documents in %s: %v", configurationPath, err).Error()
		return state, nil
	}
	if absent {
		state.Mode = DesktopStartupSetup
		return state, nil
	}

	loaded, err := loadValidatedConfigurationSet(home)
	if err != nil {
		state.Mode = DesktopStartupRepair
		state.Diagnostic = err.Error()
		return state, nil
	}

	state.Mode = DesktopStartupNormal
	state.DefaultUploader = loaded.global.DefaultUploader
	state.DefaultShortener = loaded.global.DefaultShortener
	state.CopyToClipboard = loaded.global.CopyToClipboard
	state.Uploaders = slices.Sorted(maps.Keys(loaded.uploaders.Uploaders))
	state.ShortenersPresent = loaded.shortenersPresent
	if loaded.shortenersPresent {
		state.Shorteners = slices.Sorted(maps.Keys(loaded.shorteners.Shorteners))
	}
	return state, nil
}

func loadValidatedConfigurationSet(homeDir string) (validatedConfigurationSet, error) {
	global, err := loadGlobalConfiguration(homeDir)
	if err != nil {
		return validatedConfigurationSet{}, err
	}
	return validateConfigurationSetWithGlobal(homeDir, global)
}

func validateConfigurationSetWithGlobal(homeDir string, global settings) (validatedConfigurationSet, error) {
	globalPath := filepath.Join(homeDir, ".config", "upit", "config.json")
	globalLocation := configLocation{file: absolutePath(globalPath), path: "$"}
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return validatedConfigurationSet{}, err
	}
	uploaders, err := loadUploaderConfiguration(homeDir)
	if err != nil {
		return validatedConfigurationSet{}, err
	}
	shorteners, present, err := validateConfigurationSetWithUploaderDocument(homeDir, global, uploaders)
	if err != nil {
		return validatedConfigurationSet{}, err
	}
	return validatedConfigurationSet{
		global:            global,
		uploaders:         uploaders,
		shorteners:        shorteners,
		shortenersPresent: present,
	}, nil
}

func configurationSetDocumentsAbsent(configurationPath string) (bool, error) {
	for _, name := range []string{"config.json", "custom-uploader.json", "custom-shortener.json"} {
		_, err := os.Stat(filepath.Join(configurationPath, name))
		if err == nil {
			return false, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	return true, nil
}
