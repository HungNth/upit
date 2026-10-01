//go:build windows

package app

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type fileManagerUploadLock struct{}

func newFileManagerUploadLock() fileManagerUploadLock {
	return fileManagerUploadLock{}
}

func (fileManagerUploadLock) TryAcquire(string) (func(), bool, error) {
	name, err := windows.UTF16PtrFromString(`Local\Upit.FileManagerUpload`)
	if err != nil {
		return nil, false, fmt.Errorf("name File Manager Upload mutex: %w", err)
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("create File Manager Upload mutex: %w", err)
	}
	if handle == 0 {
		return nil, false, fmt.Errorf("create File Manager Upload mutex returned an empty handle")
	}
	return func() {
		_ = windows.ReleaseMutex(handle)
		_ = windows.CloseHandle(handle)
	}, true, nil
}
