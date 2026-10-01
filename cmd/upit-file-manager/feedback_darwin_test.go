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
