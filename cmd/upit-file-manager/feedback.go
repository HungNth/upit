package main

import (
	"context"
	"crypto/rand"

	"github.com/HungNth/upit/internal/app"
)

type nativeFeedback interface {
	Begin(context.CancelFunc)
	Progress(app.ManualUploadProgress)
	Complete(app.FileManagerUploadResult) (app.FileManagerActionKind, bool)
	Alert(string)
	Close()
}

// A terminalNotification is always silent and has no activation or recovery action.
// Its payload is separate from the upload result so private values cannot leak.
type terminalNotification struct {
	ID    string
	Title string
	Body  string
}

func completeFeedback(feedback nativeFeedback, result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	if result.Status != app.FileManagerUploadSucceeded || len(result.Warnings) != 0 || len(result.Actions) != 0 {
		return feedback.Complete(result)
	}
	feedback.Close()
	if notifier, ok := feedback.(interface {
		Notify(terminalNotification) error
	}); ok {
		_ = notifier.Notify(terminalNotification{
			ID:    rand.Text()[:16],
			Title: "Upload complete",
			Body:  "Final URL copied to clipboard.",
		})
	}
	return "", false
}
