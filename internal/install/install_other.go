//go:build !windows

package install

import (
	"context"
	"errors"
)

var errWindowsOnly = errors.New("install lifecycle is only supported on Windows")

func runInstall(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	return InstallResult{}, errWindowsOnly
}

func runUninstall(ctx context.Context, opts UninstallOptions) error {
	return errWindowsOnly
}

func checkRunningProcesses(installDir string) ([]ProcessInfo, error) {
	return nil, errWindowsOnly
}
