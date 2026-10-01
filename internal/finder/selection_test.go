package finder

import (
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSelectOneFileURLRejectsInvalidSelectionShape(t *testing.T) {
	for name, rawURLs := range map[string][]string{
		"empty":    nil,
		"multiple": {"file:///one", "file:///two"},
		"remote":   {"file://server/share/payload.txt"},
		"missing":  {"https://example.test/payload.txt"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := SelectOneFileURL(rawURLs); err == nil {
				t.Fatal("SelectOneFileURL succeeded, want rejection")
			}
		})
	}
}

func TestSelectOneFileURLAcceptsOneRegularFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Finder file URL paths use POSIX semantics")
	}
	filePath := filepath.Join(t.TempDir(), "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	rawURL := (&url.URL{Scheme: "file", Path: filePath}).String()
	got, err := SelectOneFileURL([]string{rawURL})
	if err != nil {
		t.Fatalf("SelectOneFileURL failed: %v", err)
	}
	if got != filePath {
		t.Fatalf("path = %q, want %q", got, filePath)
	}
}

func TestSelectOneFileURLRejectsDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Finder file URL paths use POSIX semantics")
	}
	directory := filepath.Join(t.TempDir(), "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	rawURL := (&url.URL{Scheme: "file", Path: directory}).String()
	if _, err := SelectOneFileURL([]string{rawURL}); err == nil {
		t.Fatal("SelectOneFileURL accepted a directory")
	}
}
