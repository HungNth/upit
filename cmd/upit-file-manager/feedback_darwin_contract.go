//go:build darwin

package main

import "github.com/HungNth/upit/internal/app"

const (
	darwinActionCancel  = 1
	darwinActionCopy    = 2
	darwinActionRetry   = 3
	darwinActionOpen    = 4
	darwinActionDismiss = -1
)

func actionAvailable(result app.FileManagerUploadResult, kind app.FileManagerActionKind) bool {
	for _, action := range result.Actions {
		if action.Kind == kind {
			return true
		}
	}
	return false
}

func darwinActionKind(action int) (app.FileManagerActionKind, bool) {
	switch action {
	case darwinActionCopy:
		return app.FileManagerActionCopyFinalURL, true
	case darwinActionRetry:
		return app.FileManagerActionRetry, true
	case darwinActionOpen:
		return app.FileManagerActionOpenDesktop, true
	default:
		return "", false
	}
}

func runDarwinLaunch(args []string, notificationLaunch bool) int {
	if notificationLaunch && len(args) == 0 {
		return 0
	}
	return run(args)
}
