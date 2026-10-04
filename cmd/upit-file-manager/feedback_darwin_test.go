//go:build darwin

package main

import (
	"testing"

	"github.com/HungNth/upit/internal/app"
)

func TestDarwinActionMappingUsesOpaqueContract(t *testing.T) {
	cases := []struct {
		code int
		kind app.FileManagerActionKind
	}{
		{darwinActionCopy, app.FileManagerActionCopyFinalURL},
		{darwinActionRetry, app.FileManagerActionRetry},
		{darwinActionOpen, app.FileManagerActionOpenDesktop},
	}
	for _, testCase := range cases {
		got, ok := darwinActionKind(testCase.code)
		if !ok || got != testCase.kind {
			t.Errorf("action %d = %q, %t; want %q, true", testCase.code, got, ok, testCase.kind)
		}
	}
	if _, ok := darwinActionKind(darwinActionCancel); ok {
		t.Fatal("Cancel was exposed as a terminal recovery action")
	}
}

func TestDarwinFeedbackAdvertisesOnlyResultActions(t *testing.T) {
	result := app.FileManagerUploadResult{
		Actions: []app.FileManagerAction{{Kind: app.FileManagerActionRetry, Token: "opaque"}},
	}
	if actionAvailable(result, app.FileManagerActionCopyFinalURL) {
		t.Fatal("Copy Final URL was advertised without a shared action")
	}
	if !actionAvailable(result, app.FileManagerActionRetry) {
		t.Fatal("Retry action was not advertised")
	}
	if actionAvailable(result, app.FileManagerActionOpenDesktop) {
		t.Fatal("Open Upit Desktop was advertised without a shared action")
	}
}

func TestDarwinNotificationLaunchDoesNotWeakenArgumentValidation(t *testing.T) {
	t.Setenv("UPIT_FILE_MANAGER_HEADLESS", "1")
	if got := runDarwinLaunch(nil, true); got != 0 {
		t.Fatalf("notification-only activation exit = %d, want 0", got)
	}
	if got := runDarwinLaunch(nil, false); got != 2 {
		t.Fatalf("ordinary no-argument invocation exit = %d, want 2", got)
	}
	if got := runDarwinLaunch([]string{"--unexpected", "one", "two"}, true); got != 2 {
		t.Fatalf("malformed invocation bypassed validation under notification context: %d", got)
	}
}
