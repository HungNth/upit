//go:build windows

package app

import (
	"context"
	"testing"
)

func TestNativeWindowsClipboardCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := copyWindowsClipboard(ctx, "should-not-copy"); err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}
