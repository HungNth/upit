//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
	"sync"
	"sync/atomic"
	"syscall"
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

type taskDialogConfigPacked struct {
	data [160]byte
}

func (c *taskDialogConfigPacked) ptr() uintptr {
	return uintptr(unsafe.Pointer(&c.data[0]))
}

func (c *taskDialogConfigPacked) setCbSize(v uint32) {
	*(*uint32)(unsafe.Pointer(&c.data[0])) = v
}

func (c *taskDialogConfigPacked) setDwFlags(v uint32) {
	*(*uint32)(unsafe.Pointer(&c.data[20])) = v
}

func (c *taskDialogConfigPacked) setDwCommonButtons(v uint32) {
	*(*uint32)(unsafe.Pointer(&c.data[24])) = v
}

func (c *taskDialogConfigPacked) setWindowTitle(p uintptr) {
	*(*uintptr)(unsafe.Pointer(&c.data[28])) = p
}

func (c *taskDialogConfigPacked) setMainInstruction(p uintptr) {
	*(*uintptr)(unsafe.Pointer(&c.data[44])) = p
}

func (c *taskDialogConfigPacked) setContent(p uintptr) {
	*(*uintptr)(unsafe.Pointer(&c.data[52])) = p
}

func (c *taskDialogConfigPacked) setButtons(count uint32, buttonsPtr uintptr) {
	*(*uint32)(unsafe.Pointer(&c.data[60])) = count
	*(*uintptr)(unsafe.Pointer(&c.data[64])) = buttonsPtr
}

func (c *taskDialogConfigPacked) setCallback(cb uintptr, data uintptr) {
	*(*uintptr)(unsafe.Pointer(&c.data[140])) = cb
	*(*uintptr)(unsafe.Pointer(&c.data[148])) = data
}

type windowsFeedback struct {
	cancel    context.CancelFunc
	hwnd      atomic.Uintptr
	ready     chan struct{}
	readyErr  chan error
	readyOnce sync.Once
	done      chan struct{}
	closeOnce sync.Once
	buttons   chan int32
}

type actCtxW struct {
	cbSize                 uint32
	dwFlags                uint32
	lpSource               *uint16
	wProcessorArchitecture uint16
	wLangId                uint16
	lpAssemblyDirectory    *uint16
	lpResourceName         *uint16
	lpApplicationName      *uint16
	hModule                uintptr
}

var (
	sendMessageProc = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageW")
	feedbackIDs     sync.Map
	feedbackID      atomic.Uintptr
	taskDialogOnce  sync.Once
	taskDialogAddr  uintptr
	taskDialogErr   error
)

func getTaskDialogProc() (uintptr, error) {
	taskDialogOnce.Do(func() {
		manifest := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
<dependency>
    <dependentAssembly>
        <assemblyIdentity
            type="win32"
            name="Microsoft.Windows.Common-Controls"
            version="6.0.0.0"
            processorArchitecture="*"
            publicKeyToken="6595b64144ccf1df"
            language="*"
        />
    </dependentAssembly>
</dependency>
</assembly>`
		tmpFile, err := os.CreateTemp("", "upit-comctl6-*.manifest")
		if err != nil {
			taskDialogErr = fmt.Errorf("create manifest: %w", err)
			return
		}
		manifestPath := tmpFile.Name()
		if _, err := tmpFile.WriteString(manifest); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(manifestPath)
			taskDialogErr = fmt.Errorf("write manifest: %w", err)
			return
		}
		_ = tmpFile.Close()
		defer os.Remove(manifestPath)

		source, err := windows.UTF16PtrFromString(manifestPath)
		if err != nil {
			taskDialogErr = fmt.Errorf("manifest path utf16: %w", err)
			return
		}

		kernel32 := windows.NewLazySystemDLL("kernel32.dll")
		createActCtx := kernel32.NewProc("CreateActCtxW")
		activateActCtx := kernel32.NewProc("ActivateActCtx")
		deactivateActCtx := kernel32.NewProc("DeactivateActCtx")
		releaseActCtx := kernel32.NewProc("ReleaseActCtx")

		act := actCtxW{
			cbSize:   uint32(unsafe.Sizeof(actCtxW{})),
			lpSource: source,
		}

		hActCtx, _, callErr := createActCtx.Call(uintptr(unsafe.Pointer(&act)))
		if hActCtx == uintptr(windows.InvalidHandle) {
			taskDialogErr = fmt.Errorf("CreateActCtxW failed: %v", callErr)
			return
		}
		defer releaseActCtx.Call(hActCtx)

		var cookie uintptr
		r, _, callErr := activateActCtx.Call(hActCtx, uintptr(unsafe.Pointer(&cookie)))
		if r == 0 {
			taskDialogErr = fmt.Errorf("ActivateActCtx failed: %v", callErr)
			return
		}
		defer deactivateActCtx.Call(0, cookie)

		hComctl, err := windows.LoadLibrary("comctl32.dll")
		if err != nil {
			taskDialogErr = fmt.Errorf("LoadLibrary comctl32.dll failed: %w", err)
			return
		}

		addr, err := windows.GetProcAddress(hComctl, "TaskDialogIndirect")
		if err != nil {
			taskDialogErr = fmt.Errorf("GetProcAddress TaskDialogIndirect failed: %w", err)
			return
		}
		taskDialogAddr = addr
	})
	return taskDialogAddr, taskDialogErr
}

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
	f.done = make(chan struct{})
	f.buttons = make(chan int32, 1)
	go f.show()
}
func (f *windowsFeedback) show() {
	defer close(f.done)
	defer f.hwnd.Store(0)
	title := mustUTF16("Upit")
	instruction := mustUTF16("File Manager Upload")
	content := mustUTF16("Preparing upload…")
	copyText := mustUTF16("Copy Final URL")
	retryText := mustUTF16("Retry")
	openDesktopText := mustUTF16("Open Upit Desktop")
	var buttonBuf [36]byte
	*(*int32)(unsafe.Pointer(&buttonBuf[0])) = fileManagerCopyButton
	*(*uintptr)(unsafe.Pointer(&buttonBuf[4])) = uintptr(unsafe.Pointer(&copyText[0]))
	*(*int32)(unsafe.Pointer(&buttonBuf[12])) = fileManagerRetryButton
	*(*uintptr)(unsafe.Pointer(&buttonBuf[16])) = uintptr(unsafe.Pointer(&retryText[0]))
	*(*int32)(unsafe.Pointer(&buttonBuf[24])) = fileManagerOpenDesktopButton
	*(*uintptr)(unsafe.Pointer(&buttonBuf[28])) = uintptr(unsafe.Pointer(&openDesktopText[0]))

	var config taskDialogConfigPacked
	config.setCbSize(160)
	config.setDwFlags(taskDialogFlagAllowCancellation | taskDialogFlagShowProgressBar | taskDialogFlagSizeToContent)
	config.setDwCommonButtons(taskDialogCommonCloseButton | taskDialogCommonCancelButton)
	config.setWindowTitle(uintptr(unsafe.Pointer(&title[0])))
	config.setMainInstruction(uintptr(unsafe.Pointer(&instruction[0])))
	config.setContent(uintptr(unsafe.Pointer(&content[0])))
	config.setButtons(3, uintptr(unsafe.Pointer(&buttonBuf[0])))

	id := feedbackID.Add(1)
	config.setCallback(windows.NewCallback(taskDialogCallback), id)
	feedbackIDs.Store(id, f)
	defer feedbackIDs.Delete(id)
	var selected int32
	proc, err := getTaskDialogProc()
	if err != nil {
		f.signalReady(fmt.Errorf("TaskDialogIndirect unavailable: %w", err))
		return
	}
	hResult, _, _ := syscall.SyscallN(
		proc,
		config.ptr(),
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
func newWindowsToastCommand(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", windowsToastScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return cmd
}

func (f *windowsFeedback) Notify(notification terminalNotification) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := newWindowsToastCommand(ctx)
	return runWindowsToast(cmd, notification)
}

func (f *windowsFeedback) Alert(message string) {
	showNativeMessage(message, messageBoxOK|messageBoxInformation)
}

func (f *windowsFeedback) Close() {
	f.closeOnce.Do(func() {
		if f.ready != nil {
			_ = f.waitReady()
		}
		hwnd := windows.HWND(f.hwnd.Load())
		if hwnd != 0 {
			sendMessageProc.Call(uintptr(hwnd), taskDialogMessageClickButton, taskDialogCloseButton, 0)
		}
	})
	if f.done != nil {
		<-f.done
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

const windowsToastScript = `
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('HungNth.Upit')
if ($null -eq $notifier -or $notifier.Setting -ne [Windows.UI.Notifications.NotificationSetting]::Enabled) {
    exit 2
}

$xmlText = $env:UPIT_TOAST_XML

$doc = [Windows.Data.Xml.Dom.XmlDocument]::new()
$doc.LoadXml($xmlText)

$toast = [Windows.UI.Notifications.ToastNotification]::new($doc)
$toast.Tag = $env:UPIT_TOAST_TAG

$notifier.Show($toast)
exit 0
`
