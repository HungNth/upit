package install

import (
	"context"
)

// InstallOptions contains the parameters for an installation or update operation.
type InstallOptions struct {
	InstallDir        string
	StagedPayload     string
	StagedLauncher    string
	StagedUninstaller string
	StagedShortcut    string
	Version           string
	Silent            bool
}

// InstallResult contains the result of a successful installation.
type InstallResult struct {
	DeferredCleanup bool
}

// UninstallOptions contains the parameters for an uninstallation operation.
type UninstallOptions struct {
	InstallDir string
	Silent     bool
}

// ProcessInfo holds information about a running process that matches an installed executable.
type ProcessInfo struct {
	PID       uint32
	ImagePath string
}

// RunInstall executes the installation transaction.
func RunInstall(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	return runInstall(ctx, opts)
}

// RunUninstall executes the uninstallation transaction.
func RunUninstall(ctx context.Context, opts UninstallOptions) error {
	return runUninstall(ctx, opts)
}

// CheckRunningProcesses finds any running processes executing from the specified directory.
func CheckRunningProcesses(installDir string) ([]ProcessInfo, error) {
	return checkRunningProcesses(installDir)
}
