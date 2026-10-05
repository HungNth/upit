//go:build !windows

package app

import (
	"context"
	"fmt"
)

func copyWindowsClipboard(ctx context.Context, value string) error {
	return fmt.Errorf("windows clipboard is not supported on non-windows platform")
}
