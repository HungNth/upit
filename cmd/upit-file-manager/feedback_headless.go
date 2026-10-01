package main

import (
	"context"
	"fmt"
	"os"

	"github.com/HungNth/upit/internal/app"
)

type headlessFeedback struct{}

func newHeadlessFeedback() nativeFeedback {
	return &headlessFeedback{}
}

func (*headlessFeedback) Begin(context.CancelFunc) {}

func (*headlessFeedback) Progress(update app.ManualUploadProgress) {
	if update.Total > 0 {
		fmt.Fprintf(os.Stderr, "Upit: %s (%d/%d bytes)\n", update.Phase, update.Processed, update.Total)
		return
	}
	fmt.Fprintf(os.Stderr, "Upit: %s\n", update.Phase)
}

func (*headlessFeedback) Complete(result app.FileManagerUploadResult) (app.FileManagerActionKind, bool) {
	fmt.Fprintln(os.Stderr, resultSummary(result))
	return "", false
}

func (*headlessFeedback) Alert(message string) {
	fmt.Fprintln(os.Stderr, message)
}

func (*headlessFeedback) Close() {}
