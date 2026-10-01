//go:build darwin

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
	desktop, err := findDesktopBundleExecutable(filepath.Dir(executable))
	if err != nil {
		return err
	}
	if err := exec.Command("open", desktop).Start(); err != nil {
		return fmt.Errorf("start Upit Desktop: %w", err)
	}
	return nil
}

func findDesktopBundleExecutable(start string) (string, error) {
	for directory := start; ; directory = filepath.Dir(directory) {
		candidates := []string{
			filepath.Join(directory, "upit-desktop"),
			filepath.Join(directory, "MacOS", "upit-desktop"),
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				return candidate, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return "", fmt.Errorf("Upit Desktop executable not found")
}
