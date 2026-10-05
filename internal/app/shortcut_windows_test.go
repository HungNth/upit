package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateShortcutWithAUMID(t *testing.T) {
	tempDir := t.TempDir()
	shortcutPath := filepath.Join(tempDir, "TestApp.lnk")
	targetPath := os.Args[0]

	err := createShortcutWithAUMID(shortcutPath, targetPath, UpitAUMID)
	if err != nil {
		t.Fatalf("createShortcutWithAUMID failed: %v", err)
	}

	if _, err := os.Stat(shortcutPath); err != nil {
		t.Fatalf("shortcut file was not created: %v", err)
	}

	readBack, err := readShortcutAUMID(shortcutPath)
	if err != nil {
		t.Fatalf("readShortcutAUMID failed: %v", err)
	}

	if readBack != UpitAUMID {
		t.Fatalf("AUMID mismatch: got %q, want %q", readBack, UpitAUMID)
	}
}
