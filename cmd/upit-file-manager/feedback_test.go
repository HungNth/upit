package main

import (
	"context"
	"errors"
	"runtime"
	"testing"

	"github.com/HungNth/upit/internal/app"
)

// The recorder substitutes only the operating-system presentation boundary.
type feedbackSurface struct {
	progressOpen           bool
	interactive            bool
	notifications          []terminalNotification
	deliveryError          error
	progressAtNotification bool
}

func (*feedbackSurface) Begin(context.CancelFunc)          {}
func (*feedbackSurface) Progress(app.ManualUploadProgress) {}
func (f *feedbackSurface) Complete(app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	f.interactive = true
	return app.FileManagerActionRetry, true
}
func (f *feedbackSurface) Notify(notification terminalNotification) error {
	f.progressAtNotification = f.progressOpen
	f.notifications = append(f.notifications, notification)
	return f.deliveryError
}
func (*feedbackSurface) Alert(string) { panic("unexpected modal fallback") }
func (f *feedbackSurface) Close()     { f.progressOpen = false }

func TestCleanSuccessClosesProgressAndFinishesWithoutInteraction(t *testing.T) {
	result := app.FileManagerUploadResult{Status: app.FileManagerUploadSucceeded}
	surface := &feedbackSurface{progressOpen: true}
	kind, selected := completeFeedback(surface, result)
	if surface.progressOpen || surface.progressAtNotification || surface.interactive || selected || kind != "" {
		t.Fatalf("clean success left progress/interaction: %+v; action %q, %t", surface, kind, selected)
	}
	if runtime.GOOS == "windows" {
		if len(surface.notifications) != 0 {
			t.Fatal("Windows clean success attempted notification delivery")
		}
		if resultExitCode(result) != 0 {
			t.Fatal("silent success changed exit status")
		}
		return
	}
	if len(surface.notifications) != 1 {
		t.Fatalf("terminal events = %d, want exactly one", len(surface.notifications))
	}
	notification := surface.notifications[0]
	if notification.Title != "Upload complete" || notification.Body != "Final URL copied to clipboard." {
		t.Fatalf("notification = %+v", notification)
	}
	if resultExitCode(result) != 0 {
		t.Fatal("presentation changed successful exit status")
	}
}

func TestCleanSuccessDeliveryFailureNeverFallsBackToInteraction(t *testing.T) {
	for _, deliveryError := range []error{errors.ErrUnsupported, errors.New("notification API failed")} {
		t.Run(deliveryError.Error(), func(t *testing.T) {
			surface := &feedbackSurface{progressOpen: true, deliveryError: deliveryError}
			result := app.FileManagerUploadResult{Status: app.FileManagerUploadSucceeded}
			kind, selected := completeFeedback(surface, result)
			if surface.progressOpen || surface.interactive || selected || kind != "" || resultExitCode(result) != 0 {
				t.Fatalf("delivery failure changed clean-success outcome: %+v; action %q, %t", surface, kind, selected)
			}
		})
	}
}

func TestNonCleanOutcomesRetainInteractiveFeedback(t *testing.T) {
	cases := map[string]app.FileManagerUploadResult{
		"warning":                {Status: app.FileManagerUploadSucceeded, Warnings: []string{"shortener failed"}},
		"copy recovery":          {Status: app.FileManagerUploadSucceeded, Actions: []app.FileManagerAction{{Kind: app.FileManagerActionCopyFinalURL, Token: "private-token"}}},
		"failure":                {Status: app.FileManagerUploadFailed, Failure: &app.FileManagerUploadFailure{Stage: "request", Message: "private URL or credential"}},
		"cancel":                 {Status: app.FileManagerUploadCanceled},
		"rejection":              {Status: app.FileManagerUploadRejected},
		"configuration recovery": {Status: app.FileManagerUploadFailed, Actions: []app.FileManagerAction{{Kind: app.FileManagerActionOpenDesktop, Token: "private-token"}}},
	}
	for name, result := range cases {
		t.Run(name, func(t *testing.T) {
			surface := &feedbackSurface{progressOpen: true}
			_, selected := completeFeedback(surface, result)
			if !surface.interactive || !selected || len(surface.notifications) != 0 {
				t.Fatalf("non-clean outcome was hidden by clean-success policy: %+v", surface)
			}
		})
	}
}

func TestSequentialCleanSuccessesDoNotReplaceEarlierEvents(t *testing.T) {
	surface := &feedbackSurface{}
	result := app.FileManagerUploadResult{Status: app.FileManagerUploadSucceeded}
	completeFeedback(surface, result)
	completeFeedback(surface, result)
	if runtime.GOOS == "windows" {
		if len(surface.notifications) != 0 {
			t.Fatal("Windows sequential successes emitted notifications")
		}
		return
	}
	if len(surface.notifications) != 2 || surface.notifications[0].ID == surface.notifications[1].ID {
		t.Fatalf("sequential successes did not create distinct terminal events: %+v", surface.notifications)
	}
}
