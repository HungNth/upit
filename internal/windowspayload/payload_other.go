//go:build !windows

package windowspayload

import (
	"errors"
	"path/filepath"
	"strings"
)

var RequiredExecutables = []string{
	"upit.exe",
	"upit-desktop.exe",
	"upit-file-manager.exe",
}

type PayloadInfo struct {
	PayloadPath string
	Version     string
}

func ValidateCommittedPayload(payloadPath, expectedRoot, expectedVersion string) (*PayloadInfo, error) {
	return nil, errors.New("payload validation is only supported on Windows")
}

func ValidateStagedPayload(stagedPath, expectedVersion string) error {
	return errors.New("payload validation is only supported on Windows")
}

func ValidateLegacyTmpPayload(payloadPath, expectedRoot string) (*PayloadInfo, error) {
	return nil, errors.New("payload validation is only supported on Windows")
}

func ValidateFixturePayload(payloadPath string) error {
	return nil
}

func CanonicalizePath(p string) (string, error) {
	return p, nil
}

func LexicalCanonicalPath(p string) (string, error) {
	return p, nil
}

func PathsEqualCaseInsensitive(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
