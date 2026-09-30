//go:build !windows

package app

import "os"

func replaceConfigurationFile(stagedPath, targetPath string) (bool, error) {
	return false, os.Rename(stagedPath, targetPath)
}
