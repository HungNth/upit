package app

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func TestWindowsInstallCompensatesFailedLegacyRemoval(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"upit.exe", "upit-desktop.exe", "upit-file-manager.exe"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("payload"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	prefix := `Software\UpitTests\` + rand.Text()
	a := windowsIntegrationAdapter{executable: func() (string, error) { return filepath.Join(root, "upit-desktop.exe"), nil }, registryPath: prefix + `\verb`, metadataPath: prefix + `\product`, legacy: func(bool) (bool, error) { return true, nil }}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, a.key()+`\command`, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	if err = k.SetStringValue("", `"C:\previous helper.exe" "%1"`); err != nil {
		t.Fatal(err)
	}
	k.Close()
	product, _, err := registry.CreateKey(registry.CURRENT_USER, a.metadataPath, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	if err = product.SetStringValue("PayloadPath", `C:\previous payload`); err != nil {
		t.Fatal(err)
	}
	product.Close()
	t.Cleanup(func() { deleteRegistryTree(prefix) })
	if err = a.act(t.Context(), "install"); err == nil {
		t.Fatal("install claimed success with remaining legacy package")
	}
	k, err = registry.OpenKey(registry.CURRENT_USER, a.key()+`\command`, registry.READ)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := k.GetStringValue("")
	k.Close()
	if err != nil || got != `"C:\previous helper.exe" "%1"` {
		t.Fatalf("previous command not restored: %q %v", got, err)
	}
	product, err = registry.OpenKey(registry.CURRENT_USER, a.metadataPath, registry.READ)
	if err != nil {
		t.Fatal(err)
	}
	got, _, err = product.GetStringValue("PayloadPath")
	product.Close()
	if err != nil || got != `C:\previous payload` {
		t.Fatalf("previous active payload not restored: %q %v", got, err)
	}
	a.legacy = func(bool) (bool, error) { return false, errors.New("unknown package state") }
	if err = a.act(t.Context(), "install"); err == nil {
		t.Fatal("unknown state claimed successful cutover")
	}
	if !a.matches(root) {
		t.Fatal("unknown package state lost callable new route")
	}
	a.legacy = func(bool) (bool, error) { return false, nil }
	if err = a.act(t.Context(), "install"); err != nil {
		t.Fatal(err)
	}
	state, err := a.inspect(t.Context())
	if err != nil || state.Status != IntegrationRegistered {
		t.Fatalf("state=%#v err=%v", state, err)
	}
}

func TestWindowsFirstInstallFailureLeavesNoActiveRegistration(t *testing.T) {
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
	a := windowsIntegrationAdapter{executable: func() (string, error) { return filepath.Join(root, "upit-desktop.exe"), nil }, registryPath: prefix + `\verb`, metadataPath: prefix + `\product`, legacy: func(bool) (bool, error) { return true, nil }}
	if err := a.act(t.Context(), "install"); err == nil {
		t.Fatal("failed migration claimed success")
	}
	for _, path := range []string{a.key(), a.metadataPath} {
		key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
		if err == nil {
			key.Close()
			t.Fatalf("failed first install left active registry state at %s", path)
		}
		if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			t.Fatalf("cannot prove absent registration: %v", err)
		}
	}
}
