package app

import (
	"context"
	"testing"
)

func TestFileManagerIntegrationLinuxHasNoActions(t *testing.T) {
	s := FileManagerIntegrationService{platform: "linux"}
	state, err := s.Inspect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != IntegrationUnsupported || len(state.Actions) != 0 {
		t.Fatalf("state = %#v, want unsupported without actions", state)
	}
	if _, err := s.Act(context.Background(), "repair"); err == nil {
		t.Fatal("unsupported Repair succeeded")
	}
}
