//go:build darwin

package main

/*
#cgo darwin LDFLAGS: -framework Cocoa -framework UserNotifications
#include <stdint.h>
#include <stdlib.h>

void upitFeedbackBegin(void);
void upitFeedbackProgress(const char *phase, int64_t processed, int64_t total);
void upitFeedbackComplete(const char *summary, int copy, int retry, int openDesktop);
int upitFeedbackAlertWithActions(const char *summary, int copy, int retry, int openDesktop);
void upitFeedbackAlert(const char *message);
int upitFeedbackTakeAction(void);
int upitFeedbackNotificationsAvailable(void);
void upitFeedbackClose(void);
void upitFeedbackRunEventLoop(void);
void upitFeedbackStopEventLoop(void);
void upitFeedbackPrepareEventLoop(void);
*/
import "C"

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/HungNth/upit/internal/app"
)

type darwinFeedback struct {
	cancel        context.CancelFunc
	actions       chan int
	done          chan struct{}
	terminal      atomic.Bool
	closeFeedback func()
}

func newNativeFeedback() nativeFeedback {
	if os.Getenv("UPIT_FILE_MANAGER_HEADLESS") == "1" {
		return newHeadlessFeedback()
	}
	return &darwinFeedback{}
}
func runWithPlatformLifecycle(args []string) int {
	C.upitFeedbackPrepareEventLoop()
	result := make(chan int, 1)
	go func() {
		code := run(args)
		C.upitFeedbackStopEventLoop()
		result <- code
	}()
	C.upitFeedbackRunEventLoop()
	return <-result
}

func (f *darwinFeedback) Begin(cancel context.CancelFunc) {
	f.cancel = cancel
	f.actions = make(chan int, 2)
	f.done = make(chan struct{})
	f.closeFeedback = sync.OnceFunc(func() {
		close(f.done)
		C.upitFeedbackClose()
	})
	C.upitFeedbackBegin()
	go f.watchActions()
}

func (f *darwinFeedback) watchActions() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-f.done:
			return
		case <-ticker.C:
			action := int(C.upitFeedbackTakeAction())
			switch action {
			case 0:
				continue
			case darwinActionCancel:
				if f.cancel != nil {
					f.cancel()
				}
			case darwinActionDismiss:
				if !f.terminal.Load() {
					continue
				}
			}
			select {
			case f.actions <- action:
			default:
			}
		}
	}
}

func (f *darwinFeedback) Progress(update app.ManualUploadProgress) {
	phase := C.CString(update.Phase)
	defer C.free(unsafe.Pointer(phase))
	C.upitFeedbackProgress(phase, C.int64_t(update.Processed), C.int64_t(update.Total))
}

func (f *darwinFeedback) Complete(result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	f.terminal.Store(true)
	copyAction := actionAvailable(result, app.FileManagerActionCopyFinalURL)
	retryAction := actionAvailable(result, app.FileManagerActionRetry)
	openAction := actionAvailable(result, app.FileManagerActionOpenDesktop)
	summary := C.CString(resultSummary(result))
	defer C.free(unsafe.Pointer(summary))
	if C.upitFeedbackNotificationsAvailable() == 0 {
		if !copyAction && !retryAction && !openAction {
			f.Alert(resultSummary(result))
			return "", false
		}
		action := int(C.upitFeedbackAlertWithActions(summary, cInt(copyAction), cInt(retryAction), cInt(openAction)))
		return darwinActionKind(action)
	}
	C.upitFeedbackComplete(summary, cInt(copyAction), cInt(retryAction), cInt(openAction))
	if !copyAction && !retryAction && !openAction {
		return "", false
	}
	select {
	case action := <-f.actions:
		return darwinActionKind(action)
	case <-time.After(10 * time.Minute):
		return "", false
	}
}

func (f *darwinFeedback) Alert(message string) {
	value := C.CString(message)
	defer C.free(unsafe.Pointer(value))
	C.upitFeedbackAlert(value)
}

func (f *darwinFeedback) Close() {
	if f.closeFeedback != nil {
		f.closeFeedback()
	}
}
func cInt(value bool) C.int {
	if value {
		return 1
	}
	return 0
}
