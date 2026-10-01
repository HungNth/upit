package main

import (
	"strings"
	"testing"

	"github.com/HungNth/upit/internal/app"
)

func TestActionTokenSelectsRequestedOpaqueAction(t *testing.T) {
	result := app.FileManagerUploadResult{
		Status: app.FileManagerUploadFailed,
		Actions: []app.FileManagerAction{
			{Kind: app.FileManagerActionRetry, Token: "retry-token"},
		},
	}
	token, ok := actionToken(result, app.FileManagerActionRetry)
	if !ok || token != "retry-token" {
		t.Fatalf("actionToken = %q, %t; want retry-token, true", token, ok)
	}
	if _, ok := actionToken(result, app.FileManagerActionCopyFinalURL); ok {
		t.Fatal("missing Copy Final URL action was selected")
	}
}

func TestResultSummaryDoesNotRenderPrivateValues(t *testing.T) {
	privateValues := []string{
		`C:\Users\private\payload.txt`,
		"https://upload.example.test/secret",
		"https://files.example.test/final",
	}
	results := []app.FileManagerUploadResult{
		{Status: app.FileManagerUploadSucceeded},
		{Status: app.FileManagerUploadSucceeded, Warnings: []string{"Final URL was not copied to the clipboard"}},
		{Status: app.FileManagerUploadFailed, Failure: &app.FileManagerUploadFailure{Stage: "response", Message: "private diagnostic"}},
		{Status: app.FileManagerUploadCanceled, Failure: &app.FileManagerUploadFailure{Canceled: true}},
	}
	for _, result := range results {
		summary := resultSummary(result)
		for _, privateValue := range privateValues {
			if strings.Contains(summary, privateValue) {
				t.Fatalf("summary %q contains private value %q", summary, privateValue)
			}
		}
	}
}

func TestResultSummaryIdentifiesConfigurationRecovery(t *testing.T) {
	result := app.FileManagerUploadResult{
		Status:  app.FileManagerUploadFailed,
		Failure: &app.FileManagerUploadFailure{Stage: "validation"},
		Actions: []app.FileManagerAction{{Kind: app.FileManagerActionOpenDesktop, Token: "opaque"}},
	}
	if got := resultSummary(result); got != "Configuration Set is unavailable or invalid." {
		t.Fatalf("summary = %q, want configuration recovery message", got)
	}
}

func TestResultSummaryIdentifiesCancellation(t *testing.T) {
	result := app.FileManagerUploadResult{Status: app.FileManagerUploadCanceled}
	if got := resultSummary(result); got != "Upload canceled." {
		t.Fatalf("summary = %q, want cancellation message", got)
	}
}

func TestResultExitCodeMapsTerminalStatuses(t *testing.T) {
	cases := []struct {
		status app.FileManagerUploadStatus
		want   int
	}{
		{app.FileManagerUploadSucceeded, 0},
		{app.FileManagerUploadCanceled, 130},
		{app.FileManagerUploadFailed, 1},
		{app.FileManagerUploadRejected, 1},
	}
	for _, testCase := range cases {
		if got := resultExitCode(app.FileManagerUploadResult{Status: testCase.status}); got != testCase.want {
			t.Errorf("status %q exit code = %d, want %d", testCase.status, got, testCase.want)
		}
	}
}
