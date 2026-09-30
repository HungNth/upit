//go:build windows

package app

import (
	"syscall"
	"unsafe"
)

const moveFileReplaceExisting = 0x1

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replaceConfigurationFile(stagedPath, targetPath string) (bool, error) {
	staged, err := syscall.UTF16PtrFromString(stagedPath)
	if err != nil {
		return false, err
	}
	target, err := syscall.UTF16PtrFromString(targetPath)
	if err != nil {
		return false, err
	}
	result, _, callErr := moveFileEx.Call(uintptr(unsafe.Pointer(staged)), uintptr(unsafe.Pointer(target)), moveFileReplaceExisting)
	if result != 0 {
		return false, nil
	}
	if callErr == syscall.Errno(0) {
		callErr = syscall.GetLastError()
	}
	return true, callErr
}
