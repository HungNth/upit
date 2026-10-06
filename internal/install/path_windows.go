package install

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	environmentKeyPath = `Environment`
	pathValueName      = `PATH`
	pathEntryOwnedName = `PathEntryOwned`
)

var (
	procSendMessageTimeoutW = modUser32.NewProc("SendMessageTimeoutW")
	procRegSetValueExW      = modAdvapi32.NewProc("RegSetValueExW")
)

func broadcastEnvironmentChange() {
	envPtr, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	var result uintptr
	procSendMessageTimeoutW.Call(
		0xffff, // HWND_BROADCAST
		0x001a, // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(envPtr)),
		0x0002, // SMTO_ABORTIFHUNG
		5000,
		uintptr(unsafe.Pointer(&result)),
	)
}

func normalizePathForComparison(p string) string {
	cleaned := filepath.Clean(strings.TrimSpace(p))
	cleaned = strings.TrimRight(cleaned, `/\`)
	return strings.ToLower(cleaned)
}

func isPathInList(targetPath string, pathEntries []string) bool {
	normTarget := normalizePathForComparison(targetPath)
	for _, entry := range pathEntries {
		if normalizePathForComparison(entry) == normTarget {
			return true
		}
	}
	return false
}

func splitPathEntries(pathVal string) []string {
	if pathVal == "" {
		return nil
	}
	return strings.Split(pathVal, ";")
}

func joinPathEntries(entries []string) string {
	return strings.Join(entries, ";")
}

type pathSnapshot struct {
	hadPathKey    bool
	hadPathValue  bool
	pathValName   string
	originalBytes []byte
	originalKind  uint32
	hadOwnedValue bool
	originalOwned uint32
}

func capturePathSnapshot() (*pathSnapshot, error) {
	snap := &pathSnapshot{}

	envKey, err := registry.OpenKey(registry.CURRENT_USER, environmentKeyPath, registry.READ)
	if err == nil {
		snap.hadPathKey = true
		for _, name := range []string{pathValueName, "Path"} {
			buf := make([]byte, 2048)
			n, kind, gErr := envKey.GetValue(name, buf)
			if errors.Is(gErr, registry.ErrShortBuffer) {
				buf = make([]byte, n)
				n, kind, gErr = envKey.GetValue(name, buf)
			}
			if gErr == nil {
				snap.hadPathValue = true
				snap.pathValName = name
				snap.originalBytes = buf[:n]
				snap.originalKind = kind
				break
			} else if !errors.Is(gErr, windows.ERROR_FILE_NOT_FOUND) {
				envKey.Close()
				return nil, fmt.Errorf("failed to read %s value from Environment key: %w", name, gErr)
			}
		}
		envKey.Close()
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil, fmt.Errorf("failed to open Environment key: %w", err)
	}

	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
	if err == nil {
		owned, _, gErr := prodKey.GetIntegerValue(pathEntryOwnedName)
		if gErr == nil {
			snap.hadOwnedValue = true
			snap.originalOwned = uint32(owned)
		} else if !errors.Is(gErr, windows.ERROR_FILE_NOT_FOUND) {
			prodKey.Close()
			return nil, fmt.Errorf("failed to read PathEntryOwned from product key: %w", gErr)
		}
		prodKey.Close()
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil, fmt.Errorf("failed to open product key for PATH snapshot: %w", err)
	}

	return snap, nil
}

func (s *pathSnapshot) rollback() error {
	var errs []error

	// Rollback PATH value under HKCU\Environment
	envKey, _, err := registry.CreateKey(registry.CURRENT_USER, environmentKeyPath, registry.ALL_ACCESS)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to open Environment key for PATH rollback: %w", err))
	} else {
		defer envKey.Close()
		if s.hadPathValue {
			valName := s.pathValName
			if valName == "" {
				valName = pathValueName
			}
			// Check if current value already matches snapshot bytes and kind
			curBuf := make([]byte, 2048)
			cn, curKind, curErr := envKey.GetValue(valName, curBuf)
			if errors.Is(curErr, registry.ErrShortBuffer) {
				curBuf = make([]byte, cn)
				cn, curKind, curErr = envKey.GetValue(valName, curBuf)
			}
			if curErr == nil && curKind == s.originalKind && bytes.Equal(curBuf[:cn], s.originalBytes) {
				// Already identical, skip write
			} else {
				vName, _ := windows.UTF16PtrFromString(valName)
				var pData unsafe.Pointer
				if len(s.originalBytes) > 0 {
					pData = unsafe.Pointer(&s.originalBytes[0])
				}
				ret, _, _ := procRegSetValueExW.Call(
					uintptr(envKey),
					uintptr(unsafe.Pointer(vName)),
					0,
					uintptr(s.originalKind),
					uintptr(pData),
					uintptr(len(s.originalBytes)),
				)
				if ret != 0 {
					errs = append(errs, fmt.Errorf("failed to restore original PATH value: %w", windows.Errno(ret)))
				} else {
					// Verify observed equality after write
					vBuf := make([]byte, 2048)
					vn, vKind, vErr := envKey.GetValue(valName, vBuf)
					if errors.Is(vErr, registry.ErrShortBuffer) {
						vBuf = make([]byte, vn)
						vn, vKind, vErr = envKey.GetValue(valName, vBuf)
					}
					if vErr != nil || vKind != s.originalKind || !bytes.Equal(vBuf[:vn], s.originalBytes) {
						errs = append(errs, fmt.Errorf("PATH verification failed after rollback (vErr=%v, vKind=%d, expKind=%d)", vErr, vKind, s.originalKind))
					}
				}
			}
		} else {
			// Environment originally had no PATH value: remove any newly added PATH value
			_ = envKey.DeleteValue(pathValueName)
			_ = envKey.DeleteValue("Path")
			for _, name := range []string{pathValueName, "Path"} {
				chkBuf := make([]byte, 1)
				_, _, chkErr := envKey.GetValue(name, chkBuf)
				if chkErr == nil {
					errs = append(errs, fmt.Errorf("%s value still exists in Environment after rollback removal", name))
				} else if !errors.Is(chkErr, windows.ERROR_FILE_NOT_FOUND) {
					errs = append(errs, fmt.Errorf("error verifying absent %s value: %w", name, chkErr))
				}
			}
			if !s.hadPathKey {
				// If HKCU\Environment originally did not exist at all, and is now empty of other values, we can clean it
				valNames, _ := envKey.ReadValueNames(-1)
				subNames, _ := envKey.ReadSubKeyNames(-1)
				if len(valNames) == 0 && len(subNames) == 0 {
					_ = registry.DeleteKey(registry.CURRENT_USER, environmentKeyPath)
				}
			}
		}
	}

	// Rollback PathEntryOwned under Product key
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
	if err == nil {
		if s.hadOwnedValue {
			if sErr := prodKey.SetDWordValue(pathEntryOwnedName, s.originalOwned); sErr != nil {
				errs = append(errs, fmt.Errorf("failed to restore PathEntryOwned: %w", sErr))
			}
		} else {
			if dErr := prodKey.DeleteValue(pathEntryOwnedName); dErr != nil && !errors.Is(dErr, windows.ERROR_FILE_NOT_FOUND) {
				errs = append(errs, fmt.Errorf("failed to delete PathEntryOwned on rollback: %w", dErr))
			}
		}
		prodKey.Close()
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		errs = append(errs, fmt.Errorf("failed to open product key for PathEntryOwned rollback: %w", err))
	}

	broadcastEnvironmentChange()

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// ensureInstallDirInPath adds installDir to User PATH if not already present.
// It persists PathEntryOwned in HKCU\Software\Upit and broadcasts environment change if mutated.
func ensureInstallDirInPath(installDir string) (bool, error) {
	canonicalDir := filepath.Clean(installDir)

	envKey, _, err := registry.CreateKey(registry.CURRENT_USER, environmentKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return false, fmt.Errorf("failed to open HKCU\\Environment: %w", err)
	}
	defer envKey.Close()

	val, kind, err := envKey.GetStringValue(pathValueName)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		val, kind, err = envKey.GetStringValue("Path")
	}
	if err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return false, fmt.Errorf("failed to read User PATH: %w", err)
	}

	entries := splitPathEntries(val)
	if isPathInList(canonicalDir, entries) {
		// Equivalent entry already present.
		// If Upit already owns it (or ownership was previously recorded as 1), KEEP IT OWNED (sticky).
		// Only if ownership was not previously recorded, record 0 (user-owned).
		prodKey, pErr := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
		hadPriorOwnership := false
		if pErr == nil {
			ownedVal, _, gErr := prodKey.GetIntegerValue(pathEntryOwnedName)
			if gErr == nil && ownedVal == 1 {
				hadPriorOwnership = true
			}
			prodKey.Close()
		}
		if !hadPriorOwnership {
			if setErr := recordPathOwnership(0); setErr != nil {
				return false, setErr
			}
		}
		return false, nil
	}

	// Add exactly one canonical entry
	entries = append(entries, canonicalDir)
	newPathVal := joinPathEntries(entries)

	if kind == 0 {
		kind = registry.EXPAND_SZ
	}
	if kind == registry.EXPAND_SZ {
		err = envKey.SetExpandStringValue(pathValueName, newPathVal)
	} else {
		err = envKey.SetStringValue(pathValueName, newPathVal)
	}
	if err != nil {
		return false, fmt.Errorf("failed to write updated User PATH: %w", err)
	}

	// Record ownership
	if err := recordPathOwnership(1); err != nil {
		return false, err
	}

	broadcastEnvironmentChange()
	return true, nil
}

func recordPathOwnership(owned uint32) error {
	prodKey, _, err := registry.CreateKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to open product key to record PATH ownership: %w", err)
	}
	defer prodKey.Close()

	if err := prodKey.SetDWordValue(pathEntryOwnedName, owned); err != nil {
		return fmt.Errorf("failed to write PathEntryOwned: %w", err)
	}
	return nil
}

// removeInstallDirFromPath removes installDir from User PATH only if PathEntryOwned == 1.
func removeInstallDirFromPath(installDir string) error {
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
	if err != nil {
		// No product metadata, do not remove
		return nil
	}
	owned, _, err := prodKey.GetIntegerValue(pathEntryOwnedName)
	prodKey.Close()
	if err != nil || owned != 1 {
		// Upit did not own the entry or ownership unknown: preserve it
		return nil
	}

	envKey, err := registry.OpenKey(registry.CURRENT_USER, environmentKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return nil
	}
	defer envKey.Close()

	val, kind, err := envKey.GetStringValue(pathValueName)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		val, kind, err = envKey.GetStringValue("Path")
	}
	if err != nil {
		return nil
	}

	entries := splitPathEntries(val)
	normTarget := normalizePathForComparison(installDir)
	var newEntries []string
	removedOne := false

	for _, entry := range entries {
		if !removedOne && normalizePathForComparison(entry) == normTarget {
			removedOne = true
			continue // skip this one occurrence
		}
		newEntries = append(newEntries, entry)
	}

	if !removedOne {
		return nil
	}

	newPathVal := joinPathEntries(newEntries)
	if len(newEntries) == 0 {
		_ = envKey.DeleteValue(pathValueName)
		_ = envKey.DeleteValue("Path")
	} else {
		if kind == registry.EXPAND_SZ {
			_ = envKey.SetExpandStringValue(pathValueName, newPathVal)
		} else {
			_ = envKey.SetStringValue(pathValueName, newPathVal)
		}
	}

	broadcastEnvironmentChange()
	return nil
}
