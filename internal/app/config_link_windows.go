//go:build windows

package app

import (
	"os"
	"syscall"
)

const fileAttributeReparsePoint = 0x400

func mutationTargetIsLink(_ string, info os.FileInfo) (bool, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	attributes, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && attributes.FileAttributes&fileAttributeReparsePoint != 0, nil
}
