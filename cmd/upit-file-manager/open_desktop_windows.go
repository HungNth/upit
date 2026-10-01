//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func openDesktop() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	desktop := filepath.Join(filepath.Dir(executable), "upit-desktop.exe")
	if _, err := os.Stat(desktop); err != nil {
		return err
	}
	if err := exec.Command(desktop).Start(); err != nil {
		return fmt.Errorf("start Upit Desktop: %w", err)
	}
	return nil
}
