package main

import (
	"reflect"
	"testing"
)

type focusWindowSpy struct {
	calls []string
}

func (w *focusWindowSpy) Restore() {
	w.calls = append(w.calls, "restore")
}

func (w *focusWindowSpy) Focus() {
	w.calls = append(w.calls, "focus")
}

func TestRestoreAndFocusRestoresBeforeFocusing(t *testing.T) {
	window := &focusWindowSpy{}

	restoreAndFocus(window)

	if want := []string{"restore", "focus"}; !reflect.DeepEqual(window.calls, want) {
		t.Fatalf("calls = %#v, want %#v", window.calls, want)
	}
}

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
