package finder

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
)

var (
	errExactOneFile = errors.New("exactly one Finder file URL is required")
	errLocalFileURL = errors.New("Finder selection must be a local file URL")
	errRegularFile  = errors.New("Finder selection must be a regular file")
)

// SelectOneFileURL converts and validates one Finder file URL before upload.
func SelectOneFileURL(rawURLs []string) (string, error) {
	if len(rawURLs) != 1 {
		return "", errExactOneFile
	}
	path, err := fileURLPath(rawURLs[0])
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", errRegularFile
	}
	if !info.Mode().IsRegular() {
		return "", errRegularFile
	}
	return path, nil
}

func fileURLPath(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" || parsed.Host != "" {
		return "", errLocalFileURL
	}
	path, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil || path == "" {
		return "", errLocalFileURL
	}
	path = filepath.FromSlash(path)
	if !filepath.IsAbs(path) {
		return "", errLocalFileURL
	}
	return filepath.Clean(path), nil
}
