//go:build !windows && !darwin

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/HungNth/upit/internal/app"
)

type systemFeedback struct {
	cancel context.CancelFunc
}

func newNativeFeedback() nativeFeedback {
	return &systemFeedback{}
}

func (f *systemFeedback) Begin(cancel context.CancelFunc) {
	f.cancel = cancel
}

func (f *systemFeedback) Progress(update app.ManualUploadProgress) {
	if update.Total > 0 {
		fmt.Fprintf(os.Stderr, "Upit: %s (%d/%d bytes)\n", update.Phase, update.Processed, update.Total)
		return
	}
	fmt.Fprintf(os.Stderr, "Upit: %s\n", update.Phase)
}

func (f *systemFeedback) Complete(result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	fmt.Fprintln(os.Stderr, resultSummary(result))
	return "", false
}

func (f *systemFeedback) Alert(message string) {
	fmt.Fprintln(os.Stderr, message)
}

func (*systemFeedback) Close() {}
