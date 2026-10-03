package main

import "testing"

func TestDesktopServiceConfirmCloseAllowsWindowClose(t *testing.T) {
	service := &desktopService{}
	closed := false
	service.close = func() { closed = true }
	service.SetGlobalConfigurationDirty(true)

	service.ConfirmClose()

	if !closed || !service.allowClose.Load() {
		t.Fatalf("closed = %v, allowClose = %v, want both true", closed, service.allowClose.Load())
	}
}
