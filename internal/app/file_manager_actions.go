package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type diskFileManagerActionStore struct{}

type persistedFileManagerAction struct {
	Kind      FileManagerActionKind `json:"kind"`
	FilePath  string                `json:"filePath,omitzero"`
	FinalURL  string                `json:"finalURL,omitzero"`
	ExpiresAt time.Time             `json:"expiresAt"`
}

func (diskFileManagerActionStore) Put(homeDir string, state fileManagerActionState) (FileManagerAction, error) {
	directory := fileManagerActionDirectory(homeDir)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return FileManagerAction{}, fmt.Errorf("create File Manager Upload action directory: %w", err)
	}
	if err := secureFileManagerActionPath(directory); err != nil {
		return FileManagerAction{}, err
	}
	if err := purgeFileManagerActions(directory, time.Now()); err != nil {
		return FileManagerAction{}, err
	}
	token, err := newFileManagerActionToken()
	if err != nil {
		return FileManagerAction{}, err
	}
	data, err := json.Marshal(persistedFileManagerAction{
		Kind:      state.kind,
		FilePath:  state.filePath,
		FinalURL:  state.finalURL,
		ExpiresAt: state.expiresAt,
	})
	if err != nil {
		return FileManagerAction{}, fmt.Errorf("encode File Manager Upload action: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".action-*")
	if err != nil {
		return FileManagerAction{}, fmt.Errorf("stage File Manager Upload action: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return FileManagerAction{}, fmt.Errorf("protect File Manager Upload action: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return FileManagerAction{}, fmt.Errorf("write File Manager Upload action: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return FileManagerAction{}, fmt.Errorf("sync File Manager Upload action: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return FileManagerAction{}, fmt.Errorf("close File Manager Upload action: %w", err)
	}
	actionPath := fileManagerActionPath(directory, token)
	if err := os.Rename(temporaryPath, actionPath); err != nil {
		return FileManagerAction{}, fmt.Errorf("publish File Manager Upload action: %w", err)
	}
	if err := secureFileManagerActionPath(actionPath); err != nil {
		_ = os.Remove(actionPath)
		return FileManagerAction{}, err
	}
	return FileManagerAction{Kind: state.kind, Token: token}, nil
}
func (diskFileManagerActionStore) Purge(homeDir string, now time.Time) error {
	return purgeFileManagerActions(fileManagerActionDirectory(homeDir), now)
}

var fileManagerActionConsumeMu sync.Mutex

func (diskFileManagerActionStore) Consume(homeDir, token string) (fileManagerActionState, bool, error) {
	if !validFileManagerActionToken(token) {
		return fileManagerActionState{}, false, nil
	}
	fileManagerActionConsumeMu.Lock()
	defer fileManagerActionConsumeMu.Unlock()
	directory := fileManagerActionDirectory(homeDir)
	if err := purgeFileManagerActions(directory, time.Now()); err != nil {
		return fileManagerActionState{}, false, err
	}
	path := fileManagerActionPath(directory, token)
	claimed, err := os.CreateTemp(directory, ".consumed-*")
	if errors.Is(err, os.ErrNotExist) {
		return fileManagerActionState{}, false, nil
	}
	if err != nil {
		return fileManagerActionState{}, false, fmt.Errorf("claim File Manager Upload action: %w", err)
	}
	claimedPath := claimed.Name()
	if err := claimed.Close(); err != nil {
		_ = os.Remove(claimedPath)
		return fileManagerActionState{}, false, fmt.Errorf("close claimed File Manager Upload action: %w", err)
	}
	defer os.Remove(claimedPath)
	if err := os.Remove(claimedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fileManagerActionState{}, false, fmt.Errorf("prepare File Manager Upload action claim: %w", err)
	}
	if err := os.Rename(path, claimedPath); errors.Is(err, os.ErrNotExist) {
		return fileManagerActionState{}, false, nil
	} else if err != nil {
		return fileManagerActionState{}, false, fmt.Errorf("claim File Manager Upload action: %w", err)
	}
	data, err := os.ReadFile(claimedPath)
	if err != nil {
		return fileManagerActionState{}, false, fmt.Errorf("read claimed File Manager Upload action: %w", err)
	}
	var persisted persistedFileManagerAction
	if err := json.Unmarshal(data, &persisted); err != nil {
		return fileManagerActionState{}, false, fmt.Errorf("decode File Manager Upload action: %w", err)
	}
	if time.Now().After(persisted.ExpiresAt) {
		return fileManagerActionState{}, false, nil
	}
	return fileManagerActionState{
		kind:      persisted.Kind,
		filePath:  persisted.FilePath,
		finalURL:  persisted.FinalURL,
		expiresAt: persisted.ExpiresAt,
	}, true, nil
}

func fileManagerActionDirectory(homeDir string) string {
	return filepath.Join(homeDir, ".config", "upit", ".file-manager-actions")
}

func fileManagerActionPath(directory, token string) string {
	return filepath.Join(directory, token+".json")
}

func validFileManagerActionToken(token string) bool {
	if len(token) != 32 || strings.ToLower(token) != token {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func purgeFileManagerActions(directory string, now time.Time) error {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read File Manager Upload action directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read File Manager Upload action: %w", err)
		}
		var persisted persistedFileManagerAction
		if err := json.Unmarshal(data, &persisted); err != nil || now.After(persisted.ExpiresAt) {
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				return fmt.Errorf("remove expired File Manager Upload action: %w", removeErr)
			}
		}
	}
	return nil
}

func newFileManagerActionToken() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", fmt.Errorf("create File Manager Upload action token: %w", err)
	}
	return hex.EncodeToString(data[:]), nil
}
