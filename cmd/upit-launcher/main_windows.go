package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/HungNth/upit/internal/windowspayload"
	"golang.org/x/sys/windows/registry"
)

// launcherFormat defines the internal technical format identifier for replacement logic.
// It can be overridden at build time via -ldflags "-X main.launcherFormat=<version>".
var launcherFormat = "1"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--launcher-format" {
		fmt.Println(launcherFormat)
		os.Exit(0)
	}

	exitCode, err := runLauncher(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "upit launcher error: %v\n", err)
		if exitCode == 0 {
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func runLauncher(args []string) (int, error) {
	// Determine expected installation root
	selfExe, err := os.Executable()
	if err != nil {
		return 1, fmt.Errorf("failed to locate launcher executable: %w", err)
	}
	canonicalSelf, err := windowspayload.CanonicalizePath(selfExe)
	if err != nil {
		return 1, fmt.Errorf("failed to canonicalize launcher path: %w", err)
	}
	installRoot := filepath.Dir(canonicalSelf)

	// Sole authority: HKCU\Software\Upit PayloadPath
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Upit`, registry.QUERY_VALUE)
	if err != nil {
		return 1, fmt.Errorf("active payload metadata is missing or inaccessible (HKCU\\Software\\Upit): %w", err)
	}
	defer k.Close()

	payloadPath, valType, err := k.GetStringValue("PayloadPath")
	if err != nil || valType != registry.SZ || payloadPath == "" {
		return 1, fmt.Errorf("active PayloadPath metadata is missing or not a REG_SZ string in HKCU\\Software\\Upit")
	}
	if !filepath.IsAbs(payloadPath) {
		return 1, fmt.Errorf("active PayloadPath metadata must be an absolute path: %q", payloadPath)
	}

	// Validate committed payload strictly: fails closed, no scanning, full PE validation
	info, err := windowspayload.ValidateCommittedPayload(payloadPath, installRoot, "")
	if err != nil {
		return 1, fmt.Errorf("active payload validation failed for %q: %w", payloadPath, err)
	}

	realCliPath := filepath.Join(info.PayloadPath, "upit.exe")

	// Forward execution to real CLI
	cmd := exec.Command(realCliPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// cmd.Env defaults to nil which inherits os.Environ() without duplicate allocation

	// Stay attached to the current console process tree so console cancellation propagates to the child.
	// In Windows, children sharing the console automatically receive Ctrl+C / Ctrl+Break console events.
	// Registering a buffered signal channel suppresses default Go process termination in the launcher while waiting.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start real CLI %q: %w", realCliPath, err)
	}

	err = cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return int(status.ExitCode), nil
			}
			return exitErr.ExitCode(), nil
		}
		return 1, fmt.Errorf("CLI process execution error: %w", err)
	}

	return 0, nil
}
