package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func (s Service) SetDefaultUploader(name string) (bool, error) {
	home, global, path, info, err := s.loadGlobalMutationState()
	if err != nil {
		return false, err
	}
	uploaders, err := loadUploaderConfiguration(home)
	if err != nil {
		return false, err
	}
	if _, ok := uploaders.Uploaders[name]; !ok {
		globalLocation := configLocation{file: absolutePath(path), path: "$.defaultUploader"}
		return false, globalLocation.fail(nil, "uploader %q does not exist; choose one listed by config list-uploaders", name)
	}

	wasCurrent := global.DefaultUploader == name
	globalLocation := configLocation{file: absolutePath(path), path: "$"}
	global.DefaultUploader = name
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return false, err
	}
	if wasCurrent {
		return false, nil
	}
	if err := publishGlobalSettings(path, info, global); err != nil {
		return false, err
	}
	return true, nil
}

func (s Service) loadGlobalMutationState() (string, settings, string, os.FileInfo, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return "", settings{}, "", nil, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "config.json")
	info, err := refuseUnsafeMutationTarget(path)
	if err != nil {
		return "", settings{}, "", nil, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return "", settings{}, "", nil, err
	}
	return home, global, path, info, nil
}

func publishGlobalSettings(path string, info os.FileInfo, global settings) error {
	canonical, err := marshalSettings(global)
	if err != nil {
		return err
	}
	return publishConfiguration(path, canonical, info)
}

func (s Service) SetDefaultShortener(name string) (bool, error) {
	home, global, path, info, err := s.loadGlobalMutationState()
	if err != nil {
		return false, err
	}
	shorteners, err := loadShortenerConfiguration(home)
	if err != nil {
		return false, err
	}
	if _, ok := shorteners.Shorteners[name]; !ok {
		globalLocation := configLocation{file: absolutePath(path), path: "$.defaultShortener"}
		return false, globalLocation.fail(nil, "shortener %q does not exist; choose one listed by config list-shorteners", name)
	}

	wasCurrent := global.DefaultShortener == name
	globalLocation := configLocation{file: absolutePath(path), path: "$"}
	global.DefaultShortener = name
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return false, err
	}
	if wasCurrent {
		return false, nil
	}
	if err := publishGlobalSettings(path, info, global); err != nil {
		return false, err
	}
	return true, nil
}

func (s Service) ClearDefaultShortener() (bool, error) {
	_, global, path, info, err := s.loadGlobalMutationState()
	if err != nil {
		return false, err
	}
	wasSelected := global.DefaultShortener != ""
	global.DefaultShortener = ""
	globalLocation := configLocation{file: absolutePath(path), path: "$"}
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return false, err
	}
	if !wasSelected {
		return false, nil
	}
	if err := publishGlobalSettings(path, info, global); err != nil {
		return false, err
	}
	return true, nil
}

func (s Service) SetClipboardCopying(enabled bool) (bool, error) {
	_, global, path, info, err := s.loadGlobalMutationState()
	if err != nil {
		return false, err
	}
	wasEnabled := global.CopyToClipboard
	global.CopyToClipboard = enabled
	globalLocation := configLocation{file: absolutePath(path), path: "$"}
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return false, err
	}
	if wasEnabled == enabled {
		return false, nil
	}
	if err := publishGlobalSettings(path, info, global); err != nil {
		return false, err
	}
	return true, nil
}

func refuseUnsafeMutationTarget(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, failuref("config", err, "Global Configuration %s does not exist; create it before mutating configuration", absolutePath(path))
	}
	if err != nil {
		return nil, failuref("config", err, "inspect Global Configuration %s: %v", absolutePath(path), err)
	}
	linked, err := mutationTargetIsLink(path, info)
	if err != nil {
		return nil, failuref("config", err, "inspect Global Configuration links %s: %v", absolutePath(path), err)
	}
	if linked {
		return nil, failuref("config", nil, "Global Configuration %s is a symlink or reparse-point link; mutation refuses linked targets", absolutePath(path))
	}
	if !info.Mode().IsRegular() {
		return nil, failuref("config", nil, "Global Configuration %s is not a regular file", absolutePath(path))
	}
	return info, nil
}

func marshalSettings(global settings) ([]byte, error) {
	data, err := json.MarshalIndent(global, "", "    ")
	if err != nil {
		return nil, failuref("config", err, "encode Global Configuration: %v", err)
	}
	return append(data, '\n'), nil
}

func publishConfiguration(path string, data []byte, targetInfo os.FileInfo) error {
	directory := filepath.Dir(path)
	staged, err := os.CreateTemp(directory, ".config.json.upit-tmp-*")
	if err != nil {
		return failuref("config", err, "stage Global Configuration in %s: %v", absolutePath(directory), err)
	}
	stagedPath := staged.Name()
	removeStaged := true
	defer func() {
		_ = staged.Close()
		if removeStaged {
			_ = os.Remove(stagedPath)
		}
	}()

	if err := staged.Chmod(targetInfo.Mode().Perm()); err != nil {
		return failuref("config", err, "set staged Global Configuration permissions: %v", err)
	}
	if _, err := staged.Write(data); err != nil {
		return failuref("config", err, "write staged Global Configuration: %v", err)
	}
	if err := staged.Sync(); err != nil {
		return failuref("config", err, "synchronize staged Global Configuration: %v", err)
	}
	if err := staged.Close(); err != nil {
		return failuref("config", err, "close staged Global Configuration: %v", err)
	}

	ambiguous, err := replaceConfigurationFile(stagedPath, path)
	if err != nil {
		if ambiguous {
			removeStaged = false
			return failuref("config", err, "publish Global Configuration failed; staged recovery file retained at %s", absolutePath(stagedPath))
		}
		return failuref("config", err, "publish Global Configuration failed: %v", err)
	}
	removeStaged = false
	return nil
}
