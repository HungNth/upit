package main

import (
	"context"

	"github.com/HungNth/upit/internal/app"
)

type nativeFeedback interface {
	Begin(context.CancelFunc)
	Progress(app.ManualUploadProgress)
	Complete(app.FileManagerUploadResult) (app.FileManagerActionKind, bool)
	Alert(string)
	Close()
}
