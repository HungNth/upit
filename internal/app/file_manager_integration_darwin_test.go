//go:build darwin && cgo

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMacIntegrationRequiresCompleteMatchingPayload(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Upit.app")
	desktop := filepath.Join(root, "Contents", "MacOS", "upit-desktop")
	adapter := macIntegrationAdapter{executable: func() (string, error) { return desktop, nil }}
	s := FileManagerIntegrationService{platform: "darwin", adapter: adapter}
	state, err := s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationReinstall {
		t.Fatalf("missing payload = %#v, %v", state, err)
	}
	service := filepath.Join(root, "Contents", "Helpers", "UpitFinderService.app")
	worker := filepath.Join(service, "Contents", "Helpers", "UpitFileManager.app")
	for _, b := range []struct{ path, id, exe string }{{root, "com.hungnth.upit.desktop", "upit-desktop"}, {service, "com.hungnth.upit.finder-service", "UpitFinderService"}, {worker, "com.hungnth.upit.file-manager", "upit-file-manager"}} {
		if err = os.MkdirAll(filepath.Join(b.path, "Contents", "MacOS"), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(b.path, "Contents", "MacOS", b.exe), []byte("test executable"), 0700); err != nil {
			t.Fatal(err)
		}
		plist := fmt.Sprintf(`<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleIdentifier</key><string>%s</string><key>CFBundleShortVersionString</key><string>1.0.0</string></dict></plist>`, b.id)
		if err = os.WriteFile(filepath.Join(b.path, "Contents", "Info.plist"), []byte(plist), 0600); err != nil {
			t.Fatal(err)
		}
	}
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationNeedsRepair {
		t.Fatalf("intact unregistered = %#v, %v", state, err)
	}
	if err = os.Remove(filepath.Join(worker, "Contents", "MacOS", "upit-file-manager")); err != nil {
		t.Fatal(err)
	}
	state, err = s.Inspect(t.Context())
	if err != nil || state.Status != IntegrationReinstall {
		t.Fatalf("damaged worker = %#v, %v", state, err)
	}
	if _, err = s.Act(t.Context(), "repair"); err == nil {
		t.Fatal("Repair allowed with damaged payload")
	}
}
