//go:build windows

package app

import (
	"fmt"
	"os/exec"
	"os/user"
)

func secureFileManagerActionPath(path string) error {
	current, err := user.Current()
	if err != nil {
		return fmt.Errorf("resolve current user for File Manager Upload action ACL: %w", err)
	}
	permission := current.Username + ":(F)"
	if output, err := exec.Command("icacls.exe", path, "/inheritance:r", "/grant:r", permission).CombinedOutput(); err != nil {
		return fmt.Errorf("secure File Manager Upload action ACL: %w: %s", err, output)
	}
	return nil
}
