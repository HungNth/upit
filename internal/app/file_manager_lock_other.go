//go:build !windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type fileManagerUploadLock struct{}

func newFileManagerUploadLock() fileManagerUploadLock {
	return fileManagerUploadLock{}
}

func (fileManagerUploadLock) TryAcquire(homeDir string) (func(), bool, error) {
	directory := filepath.Join(homeDir, ".config", "upit")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, false, fmt.Errorf("create File Manager Upload lock directory: %w", err)
	}
	path := filepath.Join(directory, ".file-manager-upload.lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("create File Manager Upload lock: %w", err)
	}
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, false, fmt.Errorf("write File Manager Upload lock: %w", err)
	}
	return func() {
		_ = file.Close()
		_ = os.Remove(path)
	}, true, nil
}
