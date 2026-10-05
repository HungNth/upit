//go:build windows

package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	openClipboardProc    = user32.NewProc("OpenClipboard")
	closeClipboardProc   = user32.NewProc("CloseClipboard")
	emptyClipboardProc   = user32.NewProc("EmptyClipboard")
	setClipboardDataProc = user32.NewProc("SetClipboardData")
	globalAllocProc      = kernel32.NewProc("GlobalAlloc")
	globalFreeProc       = kernel32.NewProc("GlobalFree")
	globalLockProc       = kernel32.NewProc("GlobalLock")
	globalUnlockProc     = kernel32.NewProc("GlobalUnlock")
	rtlMoveMemoryProc    = kernel32.NewProc("RtlMoveMemory")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

func copyWindowsClipboard(ctx context.Context, value string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	utf16, err := windows.UTF16FromString(value)
	if err != nil {
		return fmt.Errorf("encode clipboard text: %w", err)
	}
	byteCount := uintptr(len(utf16) * 2)

	var opened bool
	for range 20 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		r, _, _ := openClipboardProc.Call(0)
		if r != 0 {
			opened = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !opened {
		return errors.New("open clipboard timed out or failed")
	}
	defer closeClipboardProc.Call()

	if r, _, err := emptyClipboardProc.Call(); r == 0 {
		return fmt.Errorf("empty clipboard: %w", err)
	}

	hMem, _, err := globalAllocProc.Call(gmemMoveable, byteCount)
	if hMem == 0 {
		return fmt.Errorf("allocate clipboard memory: %w", err)
	}

	ptr, _, err := globalLockProc.Call(hMem)
	if ptr == 0 {
		globalFreeProc.Call(hMem)
		return fmt.Errorf("lock clipboard memory: %w", err)
	}

	rtlMoveMemoryProc.Call(ptr, uintptr(unsafe.Pointer(&utf16[0])), byteCount)
	runtime.KeepAlive(utf16)
	globalUnlockProc.Call(hMem)

	if r, _, err := setClipboardDataProc.Call(cfUnicodeText, hMem); r == 0 {
		globalFreeProc.Call(hMem)
		return fmt.Errorf("set clipboard data: %w", err)
	}
	return nil
}
