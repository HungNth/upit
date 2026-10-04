//go:build windows

package main

import (
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsProgressWindowClosesBeforeTerminalNotification(t *testing.T) {
	feedback := &windowsFeedback{}
	feedback.Begin(func() {})
	if err := feedback.waitReady(); err != nil {
		t.Fatal(err)
	}
	hwnd := feedback.hwnd.Load()
	isWindow := windows.NewLazySystemDLL("user32.dll").NewProc("IsWindow")
	if visible, _, _ := isWindow.Call(hwnd); visible == 0 {
		t.Fatal("native progress window did not open")
	}
	closed := make(chan struct{})
	go func() {
		feedback.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("progress closure required user acknowledgment")
	}
	if visible, _, _ := isWindow.Call(hwnd); visible != 0 {
		t.Fatal("Close returned while the native progress window remained open")
	}
	feedback.Close()
}
