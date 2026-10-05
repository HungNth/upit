package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Clipboard interface {
	Copy(ctx context.Context, value string) error
}

type SystemClipboard struct{}

func (SystemClipboard) Copy(ctx context.Context, value string) error {
	switch runtime.GOOS {
	case "darwin":
		return runClipboardCommand(ctx, value, "pbcopy")
	case "windows":
		return copyWindowsClipboard(ctx, value)
	case "linux":
		return copyLinuxClipboard(ctx, value)
	default:
		return fmt.Errorf("clipboard is not supported on %s", runtime.GOOS)
	}
}

func copyLinuxClipboard(ctx context.Context, value string) error {
	var lastErr error
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			if err := runClipboardCommand(ctx, value, "wl-copy"); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		if err := runClipboardCommand(ctx, value, "xclip", "-selection", "clipboard"); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		if err := runClipboardCommand(ctx, value, "xsel", "--clipboard", "--input"); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("no clipboard command found (tried wl-copy, xclip, and xsel)")
}

func runClipboardCommand(ctx context.Context, value, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdin = strings.NewReader(value)
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return nil
}
