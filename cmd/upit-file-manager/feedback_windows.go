//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/HungNth/upit/internal/app"
	"golang.org/x/sys/windows"
)

const (
	taskDialogNotificationCreated       = 0
	taskDialogNotificationButtonClicked = 2
	taskDialogElementContent            = 0
	taskDialogMessageSetElementText     = 0x0400 + 108
	taskDialogMessageEnableButton       = 0x0400 + 111
	taskDialogMessageClickButton        = 0x0400 + 102
	taskDialogMessageSetProgressPos     = 0x0400 + 106
	taskDialogFlagAllowCancellation     = 0x0008
	taskDialogFlagShowProgressBar       = 0x0200
	taskDialogFlagSizeToContent         = 0x01000000
	taskDialogCommonCloseButton         = 0x0020
	taskDialogCommonCancelButton        = 0x0008
	taskDialogCancelButton              = 2
	taskDialogCloseButton               = 8
	fileManagerCopyButton               = 1001
	fileManagerRetryButton              = 1002
	fileManagerOpenDesktopButton        = 1003
	messageBoxOK                        = 0
	messageBoxOKCancel                  = 1
	messageBoxInformation               = 0x40
	messageBoxError                     = 0x10
)

type taskDialogButton struct {
	id   int32
	text uintptr
}

type taskDialogConfig struct {
	cbSize               uint32
	hwndParent           uintptr
	hInstance            uintptr
	dwFlags              uint32
	dwCommonButtons      uint32
	windowTitle          uintptr
	mainIcon             uintptr
	mainInstruction      uintptr
	content              uintptr
	buttonCount          uint32
	buttons              uintptr
	defaultButton        int32
	radioButtonCount     uint32
	radioButtons         uintptr
	defaultRadioButton   int32
	verificationText     uintptr
	expandedInformation  uintptr
	expandedControlText  uintptr
	collapsedControlText uintptr
	footerIcon           uintptr
	footer               uintptr
	callback             uintptr
	callbackData         uintptr
	width                uint32
}

type windowsFeedback struct {
	cancel    context.CancelFunc
	hwnd      atomic.Uintptr
	ready     chan struct{}
	readyErr  chan error
	readyOnce sync.Once
	buttons   chan int32
}

var (
	taskDialogProc  = windows.NewLazySystemDLL("comctl32.dll").NewProc("TaskDialogIndirect")
	sendMessageProc = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageW")
	feedbackIDs     sync.Map
	feedbackID      atomic.Uintptr
)

func newNativeFeedback() nativeFeedback {
	if os.Getenv("UPIT_FILE_MANAGER_HEADLESS") == "1" {
		return newHeadlessFeedback()
	}
	return &windowsFeedback{}
}

func (f *windowsFeedback) Begin(cancel context.CancelFunc) {
	f.cancel = cancel
	f.ready = make(chan struct{})
	f.readyErr = make(chan error, 1)
	f.buttons = make(chan int32, 1)
	go f.show()
}

func (f *windowsFeedback) show() {
	title := mustUTF16("Upit")
	instruction := mustUTF16("File Manager Upload")
	content := mustUTF16("Preparing upload…")
	copyText := mustUTF16("Copy Final URL")
	retryText := mustUTF16("Retry")
	openDesktopText := mustUTF16("Open Upit Desktop")
	buttons := [...]taskDialogButton{
		{id: fileManagerCopyButton, text: uintptr(unsafe.Pointer(&copyText[0]))},
		{id: fileManagerRetryButton, text: uintptr(unsafe.Pointer(&retryText[0]))},
		{id: fileManagerOpenDesktopButton, text: uintptr(unsafe.Pointer(&openDesktopText[0]))},
	}
	config := taskDialogConfig{
		cbSize:          uint32(unsafe.Sizeof(taskDialogConfig{})),
		dwFlags:         taskDialogFlagAllowCancellation | taskDialogFlagShowProgressBar | taskDialogFlagSizeToContent,
		dwCommonButtons: taskDialogCommonCloseButton | taskDialogCommonCancelButton,
		windowTitle:     uintptr(unsafe.Pointer(&title[0])),
		mainInstruction: uintptr(unsafe.Pointer(&instruction[0])),
		content:         uintptr(unsafe.Pointer(&content[0])),
		buttonCount:     uint32(len(buttons)),
		buttons:         uintptr(unsafe.Pointer(&buttons[0])),
		callback:        windows.NewCallback(taskDialogCallback),
	}
	id := feedbackID.Add(1)
	config.callbackData = id
	feedbackIDs.Store(id, f)
	defer feedbackIDs.Delete(id)
	var selected int32
	hResult, _, _ := taskDialogProc.Call(
		uintptr(unsafe.Pointer(&config)),
		uintptr(unsafe.Pointer(&selected)),
		0,
		0,
	)
	if hResult != 0 {
		f.signalReady(fmt.Errorf("TaskDialogIndirect failed: 0x%x", hResult))
		return
	}
	select {
	case f.buttons <- selected:
	default:
	}
}

func (f *windowsFeedback) Progress(update app.ManualUploadProgress) {
	hwnd := windows.HWND(f.hwnd.Load())
	if hwnd == 0 {
		return
	}
	text := update.Phase
	if update.Total > 0 {
		text = fmt.Sprintf("%s (%d of %d bytes)", update.Phase, update.Processed, update.Total)
	}
	sendDialogText(hwnd, text)
	if update.Total > 0 {
		position := uintptr(min(100, update.Processed*100/max(1, update.Total)))
		sendMessageProc.Call(uintptr(hwnd), taskDialogMessageSetProgressPos, position, 0)
	}
}

func (f *windowsFeedback) Complete(result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	if err := f.waitReady(); err != nil {
		return f.fallback(result)
	}
	hwnd := windows.HWND(f.hwnd.Load())
	sendDialogText(hwnd, resultSummary(result))
	sendMessageProc.Call(uintptr(hwnd), taskDialogMessageEnableButton, taskDialogCancelButton, 0)
	for _, action := range result.Actions {
		sendMessageProc.Call(uintptr(hwnd), taskDialogButtonID(action.Kind), 1, 0)
	}
	button := <-f.buttons
	return actionKindForButton(button)
}

func (f *windowsFeedback) Alert(message string) {
	showNativeMessage(message, messageBoxOK|messageBoxInformation)
}

func (f *windowsFeedback) Close() {
	hwnd := windows.HWND(f.hwnd.Load())
	if hwnd != 0 {
		sendMessageProc.Call(uintptr(hwnd), taskDialogMessageClickButton, taskDialogCloseButton, 0)
	}
}

func (f *windowsFeedback) waitReady() error {
	<-f.ready
	select {
	case err := <-f.readyErr:
		return err
	default:
		return nil
	}
}

func (f *windowsFeedback) signalReady(err error) {
	f.readyOnce.Do(func() {
		if err != nil {
			f.readyErr <- err
		}
		close(f.ready)
	})
}

func (f *windowsFeedback) handle(hwnd uintptr, notification uint, wParam uintptr) uintptr {
	if notification == taskDialogNotificationCreated {
		f.hwnd.Store(hwnd)
		f.signalReady(nil)
		// TDM_ENABLE_BUTTON uses lParam=0 to disable; Complete enables only returned actions.
		for _, id := range []int32{fileManagerCopyButton, fileManagerRetryButton, fileManagerOpenDesktopButton} {
			sendMessageProc.Call(hwnd, taskDialogMessageEnableButton, uintptr(id), 0)
		}
		return 0
	}
	if notification != taskDialogNotificationButtonClicked {
		return 0
	}
	button := int32(wParam)
	if button == taskDialogCancelButton || button == taskDialogCloseButton {
		if f.cancel != nil {
			f.cancel()
		}
	}
	select {
	case f.buttons <- button:
	default:
	}
	return 0
}

func taskDialogCallback(hwnd uintptr, notification uint, wParam uintptr, _ uintptr, refData uintptr) uintptr {
	value, ok := feedbackIDs.Load(refData)
	if !ok {
		return 0
	}
	return value.(*windowsFeedback).handle(hwnd, notification, wParam)
}

func sendDialogText(hwnd windows.HWND, text string) {
	value := mustUTF16(text)
	sendMessageProc.Call(uintptr(hwnd), taskDialogMessageSetElementText, taskDialogElementContent, uintptr(unsafe.Pointer(&value[0])))
}

func taskDialogButtonID(kind app.FileManagerActionKind) uintptr {
	switch kind {
	case app.FileManagerActionCopyFinalURL:
		return fileManagerCopyButton
	case app.FileManagerActionRetry:
		return fileManagerRetryButton
	case app.FileManagerActionOpenDesktop:
		return fileManagerOpenDesktopButton
	default:
		return 0
	}
}

func actionKindForButton(button int32) (app.FileManagerActionKind, bool) {
	switch button {
	case fileManagerCopyButton:
		return app.FileManagerActionCopyFinalURL, true
	case fileManagerRetryButton:
		return app.FileManagerActionRetry, true
	case fileManagerOpenDesktopButton:
		return app.FileManagerActionOpenDesktop, true
	default:
		return "", false
	}
}

func (f *windowsFeedback) fallback(result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	if len(result.Actions) == 1 {
		kind := result.Actions[0].Kind
		label := fallbackActionLabel(kind)
		caption := mustUTF16("Upit — " + label)
		text := mustUTF16(resultSummary(result) + " Select OK to " + label + ".")
		choice, _ := windows.MessageBox(0, &text[0], &caption[0], messageBoxOKCancel|messageBoxInformation)
		if choice == 1 {
			return kind, true
		}
		return "", false
	}
	showNativeMessage(resultSummary(result), messageBoxOK|messageBoxError)
	return "", false
}

func fallbackActionLabel(kind app.FileManagerActionKind) string {
	switch kind {
	case app.FileManagerActionCopyFinalURL:
		return "Copy Final URL"
	case app.FileManagerActionRetry:
		return "Retry"
	case app.FileManagerActionOpenDesktop:
		return "Open Upit Desktop"
	default:
		return "Continue"
	}
}

func showNativeMessage(message string, flags uint32) {
	text := mustUTF16(message)
	caption := mustUTF16("Upit")
	_, _ = windows.MessageBox(0, &text[0], &caption[0], flags)
}

func mustUTF16(value string) []uint16 {
	result, err := windows.UTF16FromString(value)
	if err != nil {
		return []uint16{0}
	}
	return result
}
