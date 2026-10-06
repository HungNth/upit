package windowspayload

import (
	"debug/pe"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// RequiredExecutables are the three executables that must exist in an Active Payload.
var RequiredExecutables = []string{
	"upit.exe",
	"upit-desktop.exe",
	"upit-file-manager.exe",
}

// PayloadInfo contains parsed and verified metadata about an active payload.
type PayloadInfo struct {
	PayloadPath string
	Version     string
}

// CanonicalizePath resolves symlinks and returns the clean, canonical absolute path.
// It fails if the path does not exist or symlinks cannot be resolved.
func CanonicalizePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return filepath.Clean(realPath), nil
	}
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return abs, nil
	}
	return "", fmt.Errorf("failed to evaluate symlinks for %q: %w", abs, err)
}

// LexicalCanonicalPath normalizes a path lexically without requiring it to exist on disk.
func LexicalCanonicalPath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	return filepath.Clean(abs), nil
}

// PathsEqualCaseInsensitive compares two paths for case-insensitive equivalence on Windows.
func PathsEqualCaseInsensitive(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// ValidateCommittedPayload validates an active committed payload directory.
// It enforces:
// - Absolute recorded path
// - Resolves symlinks (canonical)
// - Exists and is a directory
// - Parent directory is exactly filepath.Join(canonicalRoot, "versions")
// - Directory leaf is numeric SemVer X.Y.Z (no .tmp allowed)
// - If expectedVersion is given, directory leaf must match it
// - Contains regular executables upit.exe, upit-desktop.exe, upit-file-manager.exe (rejects symlinks)
// - AMD64 PE architecture
// - Coherent PE product/file version matching version
func ValidateCommittedPayload(payloadPath, expectedRoot, expectedVersion string) (*PayloadInfo, error) {
	if strings.TrimSpace(payloadPath) == "" {
		return nil, errors.New("payload path is empty")
	}
	if !filepath.IsAbs(payloadPath) {
		return nil, fmt.Errorf("payload path must be absolute: %q", payloadPath)
	}

	canonicalPayload, err := CanonicalizePath(payloadPath)
	if err != nil {
		return nil, fmt.Errorf("invalid payload path: %w", err)
	}

	info, err := os.Stat(canonicalPayload)
	if err != nil {
		return nil, fmt.Errorf("payload path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("payload path is not a directory: %s", canonicalPayload)
	}

	dirName := filepath.Base(canonicalPayload)
	parentDir := filepath.Dir(canonicalPayload)

	if expectedRoot != "" {
		canonicalRoot, err := CanonicalizePath(expectedRoot)
		if err != nil {
			return nil, fmt.Errorf("invalid expected root: %w", err)
		}
		expectedVersions := filepath.Join(canonicalRoot, "versions")
		if !PathsEqualCaseInsensitive(parentDir, expectedVersions) {
			return nil, fmt.Errorf("payload %s parent must be exactly %s", canonicalPayload, expectedVersions)
		}
	} else {
		if !strings.EqualFold(filepath.Base(parentDir), "versions") {
			return nil, fmt.Errorf("payload %s must reside inside a 'versions' directory", canonicalPayload)
		}
	}

	if !semverRegex.MatchString(dirName) {
		return nil, fmt.Errorf("payload directory name %q is not a valid version (expected numeric X.Y.Z, no .tmp allowed)", dirName)
	}
	if expectedVersion != "" && dirName != expectedVersion {
		return nil, fmt.Errorf("payload version %q does not match expected %q", dirName, expectedVersion)
	}

	if err := validateExecutables(canonicalPayload, dirName); err != nil {
		return nil, err
	}

	return &PayloadInfo{
		PayloadPath: canonicalPayload,
		Version:     dirName,
	}, nil
}

// ValidateStagedPayload validates a temporary staged payload before it is moved into versions\X.Y.Z.
// It verifies all 3 required regular files exist, have AMD64 PE architecture, and PE version matching expectedVersion.
func ValidateStagedPayload(stagedPath, expectedVersion string) error {
	if strings.TrimSpace(stagedPath) == "" {
		return errors.New("staged payload path is empty")
	}
	canonicalPath, err := CanonicalizePath(stagedPath)
	if err != nil {
		return fmt.Errorf("invalid staged payload path: %w", err)
	}

	info, err := os.Stat(canonicalPath)
	if err != nil {
		return fmt.Errorf("staged payload does not exist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("staged payload is not a directory: %s", canonicalPath)
	}

	return validateExecutables(canonicalPath, expectedVersion)
}

// ValidateLegacyTmpPayload validates an existing prior random .tmp layout under expectedRoot\versions.
// Used only for one-time legacy cutover.
func ValidateLegacyTmpPayload(payloadPath, expectedRoot string) (*PayloadInfo, error) {
	if strings.TrimSpace(payloadPath) == "" {
		return nil, errors.New("legacy payload path is empty")
	}
	if !filepath.IsAbs(payloadPath) {
		return nil, fmt.Errorf("legacy payload path must be absolute: %q", payloadPath)
	}
	canonicalPayload, err := CanonicalizePath(payloadPath)
	if err != nil {
		return nil, fmt.Errorf("invalid legacy payload path: %w", err)
	}

	canonicalRoot, err := CanonicalizePath(expectedRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid expected root: %w", err)
	}
	expectedVersions := filepath.Join(canonicalRoot, "versions")
	parentDir := filepath.Dir(canonicalPayload)
	if !PathsEqualCaseInsensitive(parentDir, expectedVersions) {
		return nil, fmt.Errorf("legacy payload %s parent must be exactly %s", canonicalPayload, expectedVersions)
	}

	dirName := filepath.Base(canonicalPayload)
	if !strings.HasSuffix(strings.ToLower(dirName), ".tmp") {
		return nil, fmt.Errorf("legacy payload %s is not a .tmp directory", canonicalPayload)
	}

	// Verify required regular files exist
	for _, exeName := range RequiredExecutables {
		exePath := filepath.Join(canonicalPayload, exeName)
		fi, err := os.Lstat(exePath)
		if err != nil {
			return nil, fmt.Errorf("required legacy binary %s is missing: %w", exeName, err)
		}
		if !fi.Mode().IsRegular() {
			return nil, fmt.Errorf("required legacy binary %s must be a regular file", exeName)
		}
	}

	return &PayloadInfo{
		PayloadPath: canonicalPayload,
		Version:     dirName,
	}, nil
}

// ValidateFixturePayload validates synthetic unit test fixtures where files are regular but lack PE headers.
func ValidateFixturePayload(payloadPath string) error {
	for _, exeName := range RequiredExecutables {
		exePath := filepath.Join(payloadPath, exeName)
		fi, err := os.Lstat(exePath)
		if err != nil {
			return fmt.Errorf("required fixture binary %s is missing: %w", exeName, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("required fixture binary %s is not a regular file", exeName)
		}
	}
	return nil
}

func validateExecutables(payloadDir, expectedVersion string) error {
	for _, exeName := range RequiredExecutables {
		exePath := filepath.Join(payloadDir, exeName)
		fi, err := os.Lstat(exePath)
		if err != nil {
			return fmt.Errorf("required payload binary %s is missing: %w", exeName, err)
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("required payload binary %s must be a regular file, got mode %s", exeName, fi.Mode())
		}

		canonicalExe, err := CanonicalizePath(exePath)
		if err != nil {
			return fmt.Errorf("failed to canonicalize payload binary %s: %w", exeName, err)
		}
		if !PathsEqualCaseInsensitive(filepath.Dir(canonicalExe), payloadDir) {
			return fmt.Errorf("payload binary %s canonical path %s is outside payload directory", exeName, canonicalExe)
		}

		if err := validatePEArchitecture(canonicalExe); err != nil {
			return fmt.Errorf("invalid architecture for %s: %w", exeName, err)
		}

		if expectedVersion != "" && semverRegex.MatchString(expectedVersion) {
			if err := validatePEVersion(canonicalExe, expectedVersion); err != nil {
				return fmt.Errorf("version check failed for %s: %w", exeName, err)
			}
		}
	}
	return nil
}

func validatePEArchitecture(filePath string) error {
	f, err := pe.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open PE file: %w", err)
	}
	defer f.Close()

	if f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return fmt.Errorf("expected AMD64 architecture, got 0x%x", f.Machine)
	}
	return nil
}

var (
	modVersion                  = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSizeW = modVersion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = modVersion.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = modVersion.NewProc("VerQueryValueW")
)

type vsFixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

func validatePEVersion(filePath string, expectedVersion string) error {
	pPath, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return err
	}

	var handle uintptr
	size, _, _ := procGetFileVersionInfoSizeW.Call(uintptr(unsafe.Pointer(pPath)), uintptr(unsafe.Pointer(&handle)))
	if size == 0 {
		return fmt.Errorf("missing PE version resource in %s", filepath.Base(filePath))
	}

	buf := make([]byte, size)
	ret, _, callErr := procGetFileVersionInfoW.Call(
		uintptr(unsafe.Pointer(pPath)),
		0,
		uintptr(size),
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if ret == 0 {
		return fmt.Errorf("failed to read PE version info: %w", callErr)
	}

	subBlock, _ := windows.UTF16PtrFromString(`\`)
	var pValue unsafe.Pointer
	var valLen uint32
	ret, _, callErr = procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(subBlock)),
		uintptr(unsafe.Pointer(&pValue)),
		uintptr(unsafe.Pointer(&valLen)),
	)
	if ret == 0 || valLen < uint32(unsafe.Sizeof(vsFixedFileInfo{})) || pValue == nil {
		return fmt.Errorf("failed to query root VS_FIXEDFILEINFO: %w", callErr)
	}

	fixedInfo := (*vsFixedFileInfo)(pValue)
	if fixedInfo.Signature != 0xFEEF04BD {
		return fmt.Errorf("invalid VS_FIXEDFILEINFO signature 0x%x", fixedInfo.Signature)
	}
	var major, minor, patch uint16
	n, parseErr := fmt.Sscanf(expectedVersion, "%d.%d.%d", &major, &minor, &patch)
	if n != 3 || parseErr != nil {
		return fmt.Errorf("invalid expected version format %s", expectedVersion)
	}

	fileMajor := uint16(fixedInfo.FileVersionMS >> 16)
	fileMinor := uint16(fixedInfo.FileVersionMS & 0xFFFF)
	filePatch := uint16(fixedInfo.FileVersionLS >> 16)

	prodMajor := uint16(fixedInfo.ProductVersionMS >> 16)
	prodMinor := uint16(fixedInfo.ProductVersionMS & 0xFFFF)
	prodPatch := uint16(fixedInfo.ProductVersionLS >> 16)

	if fileMajor != major || fileMinor != minor || filePatch != patch {
		return fmt.Errorf("file version %d.%d.%d does not match expected %s", fileMajor, fileMinor, filePatch, expectedVersion)
	}
	if prodMajor != major || prodMinor != minor || prodPatch != patch {
		return fmt.Errorf("product version %d.%d.%d does not match expected %s", prodMajor, prodMinor, prodPatch, expectedVersion)
	}

	return nil
}
