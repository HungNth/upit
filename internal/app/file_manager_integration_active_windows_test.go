package app

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsRepairUsesActiveInstalledPayload(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"upit.exe", "upit-desktop.exe", "upit-file-manager.exe"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	prefix := `Software\UpitTests\` + rand.Text()
	t.Cleanup(func() {
		if err := deleteRegistryTree(prefix); err != nil {
			t.Error(err)
		}
	})
	executable := filepath.Join(root, "upit-desktop.exe")
	a := windowsIntegrationAdapter{executable: func() (string, error) { return executable, nil }, registryPath: prefix + `\verb`, metadataPath: prefix + `\product`, legacy: func(bool) (bool, error) { return false, nil }}
	if err := a.act(t.Context(), "install"); err != nil {
		t.Fatal(err)
	}
	// A Desktop instance still running from a previous payload must neither classify
	// the new installation as broken nor rewrite the command to its own old helper.
	executable = filepath.Join(t.TempDir(), "upit-desktop.exe")
	s := FileManagerIntegrationService{platform: "windows", adapter: a}
	state, err := s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationRegistered {
		t.Fatalf("active state=%#v err=%v", state, err)
	}
	if err = a.removeVerb(); err != nil {
		t.Fatal(err)
	}
	state, err = s.Act(t.Context(), "repair")
	if err != nil || state.Status != IntegrationRegistered {
		t.Fatalf("repair=%#v err=%v", state, err)
	}
	if !a.matches(root) {
		t.Fatal("Repair targeted the stale Desktop payload")
	}
}

func TestWindowsIntegrationRequiresInstalledPayloadMetadata(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"upit.exe", "upit-desktop.exe", "upit-file-manager.exe"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	prefix := `Software\UpitTests\` + rand.Text()
	t.Cleanup(func() {
		if err := deleteRegistryTree(prefix); err != nil {
			t.Error(err)
		}
	})
	a := windowsIntegrationAdapter{executable: func() (string, error) { return filepath.Join(root, "upit-desktop.exe"), nil }, registryPath: prefix + `\verb`, metadataPath: prefix + `\product`, legacy: func(bool) (bool, error) { return false, nil }}
	s := FileManagerIntegrationService{platform: "windows", adapter: a}
	state, err := s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationReinstall {
		t.Fatalf("uninstalled state=%#v err=%v", state, err)
	}
	if _, err = s.Act(t.Context(), "repair"); err == nil {
		t.Fatal("Repair registered an uninstalled collection of binaries")
	}
}
