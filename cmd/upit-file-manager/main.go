package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/HungNth/upit/internal/app"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	runner := app.NewFileManagerUploadService(app.Service{})
	switch {
	case len(args) == 1:
		return runUpload(ctx, runner, args[0])
	case len(args) == 2 && args[0] == "--action":
		return runAction(ctx, runner, args[1])
	default:
		newNativeFeedback().Alert("Upit accepts exactly one selected file.")
		return 2
	}
}

func runUpload(ctx context.Context, runner *app.FileManagerUploadService, filePath string) int {
	cancelCtx, cancel := context.WithCancel(ctx)
	feedback := newNativeFeedback()
	defer feedback.Close()
	feedback.Begin(cancel)
	result := runner.Upload(cancelCtx, []string{filePath}, feedback.Progress)
	kind, selected := feedback.Complete(result)
	if !selected {
		cancel()
		_ = runner.DiscardActions(result)
		return resultExitCode(result)
	}
	cancel()
	return runSelectedAction(ctx, runner, result, kind)
}

func runAction(ctx context.Context, runner *app.FileManagerUploadService, token string) int {
	cancelCtx, cancel := context.WithCancel(ctx)
	feedback := newNativeFeedback()
	defer feedback.Close()
	feedback.Begin(cancel)
	action, err := runner.Dispatch(cancelCtx, token, feedback.Progress)
	if err != nil {
		feedback.Close()
		cancel()
		feedback.Alert("The Upit action is unavailable or expired.")
		return 1
	}
	if action.Upload == nil {
		feedback.Close()
		cancel()
		return finishAction(feedback, action.Kind)
	}
	kind, selected := feedback.Complete(*action.Upload)
	if !selected {
		cancel()
		_ = runner.DiscardActions(*action.Upload)
		return resultExitCode(*action.Upload)
	}
	cancel()
	return runSelectedAction(ctx, runner, *action.Upload, kind)
}

func runSelectedAction(ctx context.Context, runner *app.FileManagerUploadService, result app.FileManagerUploadResult, kind app.FileManagerActionKind) int {
	token, ok := actionToken(result, kind)
	if !ok {
		return resultExitCode(result)
	}
	switch kind {
	case app.FileManagerActionOpenDesktop, app.FileManagerActionCopyFinalURL, app.FileManagerActionRetry:
		return runAction(ctx, runner, token)
	default:
		return resultExitCode(result)
	}
}

func finishAction(feedback nativeFeedback, kind app.FileManagerActionKind) int {
	switch kind {
	case app.FileManagerActionCopyFinalURL:
		feedback.Alert("Final URL copied to the clipboard.")
		return 0
	case app.FileManagerActionOpenDesktop:
		if err := openDesktop(); err != nil {
			feedback.Alert("Upit Desktop could not be opened.")
			return 1
		}
		return 0
	default:
		return 0
	}
}

func actionToken(result app.FileManagerUploadResult, kind app.FileManagerActionKind) (string, bool) {
	for _, action := range result.Actions {
		if action.Kind == kind {
			return action.Token, true
		}
	}
	return "", false
}

func resultExitCode(result app.FileManagerUploadResult) int {
	if result.Status == app.FileManagerUploadSucceeded {
		return 0
	}
	if result.Status == app.FileManagerUploadCanceled {
		return 130
	}
	return 1
}

func openDesktop() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	desktop := filepath.Join(filepath.Dir(executable), "upit-desktop.exe")
	if _, err := os.Stat(desktop); err != nil {
		return err
	}
	if err := exec.Command(desktop).Start(); err != nil {
		return fmt.Errorf("start Upit Desktop: %w", err)
	}
	return nil
}

func resultSummary(result app.FileManagerUploadResult) string {
	if result.Status == app.FileManagerUploadSucceeded {
		if len(result.Warnings) > 0 {
			return "Upload completed with a warning."
		}
		return "Upload complete."
	}
	if result.Status == app.FileManagerUploadCanceled || (result.Failure != nil && result.Failure.Canceled) {
		return "Upload canceled."
	}
	if result.Failure != nil && result.Failure.Message == "a File Manager Upload is already active" {
		return "Another File Manager Upload is already active."
	}
	if result.Failure != nil && result.Failure.Stage == "config" || actionHasKind(result, app.FileManagerActionOpenDesktop) {
		return "Configuration Set is unavailable or invalid."
	}
	return "Upload failed."
}

func actionHasKind(result app.FileManagerUploadResult, kind app.FileManagerActionKind) bool {
	for _, action := range result.Actions {
		if action.Kind == kind {
			return true
		}
	}
	return false
}
