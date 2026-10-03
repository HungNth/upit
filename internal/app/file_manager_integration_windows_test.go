package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestWindowsIntegrationMissingOrUnsignedRepairMaterialFailsClosed(t *testing.T) {
	root := t.TempDir()
	s := FileManagerIntegrationService{platform: "windows", adapter: windowsIntegrationAdapter{executable: func() (string, error) { return filepath.Join(root, "upit-desktop.exe"), nil }}}
	for _, artifact := range []bool{false, true} {
		if artifact {
			if err := os.MkdirAll(filepath.Join(root, "repair"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "repair", "Upit.msix"), []byte("unsigned damaged artifact"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		state, err := s.Inspect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if state.Status != IntegrationReinstall || slices.Contains(state.Actions, "repair") {
			t.Fatalf("state = %#v, want reinstall without Repair", state)
		}
		if _, err = s.Act(t.Context(), "repair"); err == nil {
			t.Fatal("Repair accepted unsafe registration inputs")
		}
	}
}
