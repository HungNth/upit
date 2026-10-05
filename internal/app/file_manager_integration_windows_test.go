package app

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestWindowsIntegrationClassicVerbLifecycle(t *testing.T) {
	root := t.TempDir()
	prefix := `Software\UpitTests\` + rand.Text()
	key := prefix + `\verb`
	metadataPath := prefix + `\product`
	t.Cleanup(func() {
		if err := deleteRegistryTree(prefix); err != nil {
			t.Error(err)
		}
	})
	metadata, _, err := registry.CreateKey(registry.CURRENT_USER, metadataPath, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	if err = metadata.SetStringValue("PayloadPath", root); err != nil {
		t.Fatal(err)
	}
	metadata.Close()
	for _, name := range []string{"upit.exe", "upit-desktop.exe", "upit-file-manager.exe"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	legacy := false
	a := windowsIntegrationAdapter{executable: func() (string, error) { return filepath.Join(root, "upit-desktop.exe"), nil }, registryPath: key, metadataPath: metadataPath, legacy: func(remove bool) (bool, error) {
		if remove {
			legacy = false
		}
		return legacy, nil
	}}
	s := FileManagerIntegrationService{platform: "windows", adapter: a}
	state, err := s.Inspect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != IntegrationNeedsRepair {
		t.Fatalf("state: %#v", state)
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.READ); err == nil {
		k.Close()
		t.Fatal("inspection created registration")
	}
	state, err = s.Act(t.Context(), "repair")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != IntegrationRegistered {
		t.Fatalf("state: %#v", state)
	}
	k, err := registry.OpenKey(registry.CURRENT_USER, key+`\command`, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	command, _, err := k.GetStringValue("")
	if err != nil {
		t.Fatal(err)
	}
	want := `"` + filepath.Join(root, "upit-file-manager.exe") + `" "%1"`
	if command != want {
		t.Fatalf("command=%q want %q", command, want)
	}
	if err = k.SetStringValue("", `"C:\stale.exe" "%1"`); err != nil {
		t.Fatal(err)
	}
	k.Close()
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationNeedsRepair {
		t.Fatalf("stale state=%#v err=%v", state, err)
	}
	if _, err = s.Act(t.Context(), "repair"); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"": "Wrong menu label", "Icon": "stale.ico", "MultiSelectModel": "Player", "ExplorerCommandHandler": "obsolete handler"} {
		verb, err := registry.OpenKey(registry.CURRENT_USER, key, registry.ALL_ACCESS)
		if err != nil {
			t.Fatal(err)
		}
		if err = verb.SetStringValue(name, value); err != nil {
			t.Fatal(err)
		}
		verb.Close()
		state, err = s.Inspect(t.Context())
		if err != nil || state.Status != IntegrationNeedsRepair {
			t.Fatalf("wrong %s state=%#v err=%v", name, state, err)
		}
		if _, err = s.Act(t.Context(), "repair"); err != nil {
			t.Fatal(err)
		}
	}
	legacy = true
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationNeedsRepair {
		t.Fatalf("legacy state=%#v err=%v", state, err)
	}
	if _, err = s.Act(t.Context(), "repair"); err != nil {
		t.Fatal(err)
	}
	if err = a.act(t.Context(), "remove-registration"); err != nil {
		t.Fatal(err)
	}
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationNeedsRepair {
		t.Fatalf("removed state=%#v err=%v", state, err)
	}
	if err = os.Remove(filepath.Join(root, "upit-file-manager.exe")); err != nil {
		t.Fatal(err)
	}
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationReinstall || slices.Contains(state.Actions, "repair") {
		t.Fatalf("missing state=%#v err=%v", state, err)
	}
}
