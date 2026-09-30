package app

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type ConfigurationView struct {
	DefaultUploader  string
	DefaultShortener string
	CopyToClipboard  bool
}

func (s Service) ConfigurationPath() (string, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return "", failuref("config", err, "resolve user home directory: %v", err)
	}
	return absolutePath(filepath.Join(home, ".config", "upit")), nil
}

func (s Service) ShowConfiguration() (ConfigurationView, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return ConfigurationView{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return ConfigurationView{}, err
	}
	return ConfigurationView{
		DefaultUploader:  global.DefaultUploader,
		DefaultShortener: global.DefaultShortener,
		CopyToClipboard:  global.CopyToClipboard,
	}, nil
}

func (s Service) ListUploaders() ([]string, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return nil, failuref("config", err, "resolve user home directory: %v", err)
	}
	document, err := loadUploaderConfiguration(home)
	if err != nil {
		return nil, err
	}
	return slices.Sorted(maps.Keys(document.Uploaders)), nil
}

func (s Service) ListShorteners() ([]string, bool, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return nil, false, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	document, err := loadShortenerConfiguration(home)
	if err != nil {
		return nil, true, err
	}
	return slices.Sorted(maps.Keys(document.Shorteners)), true, nil
}

func (s Service) ValidateConfiguration() error {
	home, err := s.homeDirectory()
	if err != nil {
		return failuref("config", err, "resolve user home directory: %v", err)
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return err
	}
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

	shorteners, present, err := loadOptionalShortenerConfiguration(home)
	if err != nil {
		return err
	}
	if global.DefaultShortener != "" {
		if !present {
			return globalLocation.child("defaultShortener").fail(nil, "selected default Shortener %q requires custom-shortener.json", global.DefaultShortener)
		}
		if _, ok := shorteners.Shorteners[global.DefaultShortener]; !ok {
			return globalLocation.child("defaultShortener").fail(nil, "default Shortener %q does not exist; choose one listed by config list-shorteners", global.DefaultShortener)
		}
	}
	return nil
}

func validateGlobalStructure(global settings, location configLocation) error {
	if !global.hasDefaultUploader {
		return location.child("defaultUploader").fail(nil, "defaultUploader is required; select a named Uploader")
	}
	if !global.hasCopyToClipboard {
		return location.child("copyToClipboard").fail(nil, "copyToClipboard is required; set it to true or false")
	}
	if global.DefaultUploader == "" {
		return location.child("defaultUploader").fail(nil, "defaultUploader is required; select a named Uploader")
	}
	if !validDefinitionName(global.DefaultUploader) {
		return location.child("defaultUploader").fail(nil, "defaultUploader must name a printable Uploader without leading or trailing Unicode whitespace")
	}
	if global.DefaultShortener != "" && !validDefinitionName(global.DefaultShortener) {
		return location.child("defaultShortener").fail(nil, "defaultShortener must name a printable Shortener without leading or trailing Unicode whitespace")
	}
	return nil
}

func (s Service) homeDirectory() (string, error) {
	if s.HomeDir != nil {
		return s.HomeDir()
	}
	return os.UserHomeDir()
}

func loadGlobalConfiguration(homeDir string) (settings, error) {
	path := filepath.Join(homeDir, ".config", "upit", "config.json")
	var global settings
	if err := readJSON(path, &global); err != nil {
		return settings{}, configurationReadFailure(path, err, "copy examples/config.example.json to this location")
	}
	location := configLocation{file: absolutePath(path), path: "$.version"}
	if global.Version != 2 {
		return settings{}, location.fail(nil, "config version = %d, want 2; received version %d, expected version 2; automatic migration is unavailable; update version to 2 manually; see schemas/config.schema.json", global.Version, global.Version)
	}
	return global, nil
}

func loadUploaderConfiguration(homeDir string) (uploaderDocument, error) {
	path := filepath.Join(homeDir, ".config", "upit", "custom-uploader.json")
	if err := validateCredentialFilePermissions(path); err != nil {
		return uploaderDocument{}, err
	}
	var document uploaderDocument
	if err := readJSON(path, &document); err != nil {
		return uploaderDocument{}, configurationReadFailure(path, err, "copy examples/custom-uploader.example.json to this location")
	}
	location := configLocation{file: absolutePath(path), path: "$.uploaders"}
	if document.Version != 2 {
		return uploaderDocument{}, configLocation{file: absolutePath(path), path: "$.version"}.fail(nil, "custom uploader version = %d, want 2; received version %d, expected version 2; automatic migration is unavailable; update version to 2 manually; see schemas/custom-uploader.schema.json", document.Version, document.Version)
	}
	if len(document.Uploaders) == 0 {
		return uploaderDocument{}, location.fail(nil, "at least one Uploader is required; add a named Uploader")
	}
	for _, name := range slices.Sorted(maps.Keys(document.Uploaders)) {
		candidate := document.Uploaders[name]
		if err := validateUploader(location, name, &candidate); err != nil {
			return uploaderDocument{}, err
		}
		document.Uploaders[name] = candidate
	}
	return document, nil
}

func loadOptionalShortenerConfiguration(homeDir string) (shortenerDocument, bool, error) {
	path := filepath.Join(homeDir, ".config", "upit", "custom-shortener.json")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return shortenerDocument{}, false, nil
	}
	document, err := loadShortenerConfiguration(homeDir)
	return document, true, err
}
