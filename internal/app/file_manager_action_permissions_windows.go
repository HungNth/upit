//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"os/user"
	"syscall"

	"golang.org/x/sys/windows"
)

func secureFileManagerActionPath(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("resolve current user for File Manager Upload action ACL: %w", err)
	}
	permission := current.Username + ":(F)"
	cmd := exec.Command("icacls.exe", path, "/inheritance:r", "/grant:r", permission)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secure File Manager Upload action ACL: %w: %s", err, output)
	}
	return nil
}
