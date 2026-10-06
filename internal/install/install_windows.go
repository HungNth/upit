package install

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/HungNth/upit/internal/windowspayload"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	classicVerbKey       = `Software\Classes\*\shell\Upit.Upload`
	classicVerbCommand   = `Software\Classes\*\shell\Upit.Upload\command`
	productKeyPath       = `Software\Upit`
	uninstallKeyPath     = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Upit`
	transactionBackupKey = `Software\UpitInstallerTransactions`
	inactivePayloadsVal  = `InactivePayloads`
	appUserModelID       = "HungNth.Upit"
)

var (
	modShell32                  = windows.NewLazySystemDLL("shell32.dll")
	procSHChangeNotify           = modShell32.NewProc("SHChangeNotify")
	modAdvapi32                 = windows.NewLazySystemDLL("advapi32.dll")
	procRegCopyTreeW             = modAdvapi32.NewProc("RegCopyTreeW")
	procRegDeleteTreeW           = modAdvapi32.NewProc("RegDeleteTreeW")
	modKernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW          = modKernel32.NewProc("Process32FirstW")
	procProcess32NextW           = modKernel32.NewProc("Process32NextW")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")
	modUser32                   = windows.NewLazySystemDLL("user32.dll")
	procMessageBoxW              = modUser32.NewProc("MessageBoxW")
)

func notifyExplorer() {
	procSHChangeNotify.Call(0x08000000, 0, 0, 0)
}

func deleteRegistryTree(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	result, _, _ := procRegDeleteTreeW.Call(uintptr(registry.CURRENT_USER), uintptr(unsafe.Pointer(name)))
	if result != 0 && result != uintptr(windows.ERROR_FILE_NOT_FOUND) {
		return windows.Errno(result)
	}
	return nil
}
func copyRegistryTree(source, destination registry.Key) error {
	result, _, _ := procRegCopyTreeW.Call(uintptr(source), 0, uintptr(destination))
	if result != 0 {
		return windows.Errno(result)
	}
	return nil
}

func compareRegistryTrees(expectedKey, actualKey registry.Key) error {
	expectedValNames, err := expectedKey.ReadValueNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read expected value names: %w", err)
	}
	actualValNames, err := actualKey.ReadValueNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read actual value names: %w", err)
	}

	expValMap := make(map[string]bool)
	for _, name := range expectedValNames {
		expValMap[strings.ToLower(name)] = true
	}
	for _, name := range actualValNames {
		if !expValMap[strings.ToLower(name)] {
			return fmt.Errorf("unexpected value %q found in restored key", name)
		}
	}

	for _, name := range expectedValNames {
		expBuf := make([]byte, 1024)
		n, expValType, err := expectedKey.GetValue(name, expBuf)
		if errors.Is(err, registry.ErrShortBuffer) {
			expBuf = make([]byte, n)
			n, expValType, err = expectedKey.GetValue(name, expBuf)
		}
		if err != nil {
			return fmt.Errorf("failed to get expected value %q: %w", name, err)
		}
		expBytes := expBuf[:n]

		actBuf := make([]byte, 1024)
		an, actValType, err := actualKey.GetValue(name, actBuf)
		if errors.Is(err, registry.ErrShortBuffer) {
			actBuf = make([]byte, an)
			an, actValType, err = actualKey.GetValue(name, actBuf)
		}
		if err != nil {
			return fmt.Errorf("missing or unreadable restored value %q: %w", name, err)
		}
		actBytes := actBuf[:an]

		if expValType != actValType {
			return fmt.Errorf("value %q type mismatch: expected %d, got %d", name, expValType, actValType)
		}
		if string(expBytes) != string(actBytes) {
			return fmt.Errorf("value %q content mismatch", name)
		}
	}

	expectedSubkeys, err := expectedKey.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read expected subkey names: %w", err)
	}
	actualSubkeys, err := actualKey.ReadSubKeyNames(-1)
	if err != nil {
		return fmt.Errorf("failed to read actual subkey names: %w", err)
	}

	expSubMap := make(map[string]bool)
	for _, name := range expectedSubkeys {
		expSubMap[strings.ToLower(name)] = true
	}
	for _, name := range actualSubkeys {
		if !expSubMap[strings.ToLower(name)] {
			return fmt.Errorf("unexpected subkey %q found in restored key", name)
		}
	}

	for _, name := range expectedSubkeys {
		expChild, err := registry.OpenKey(expectedKey, name, registry.READ)
		if err != nil {
			return fmt.Errorf("failed to open expected subkey %q: %w", name, err)
		}
		actChild, err := registry.OpenKey(actualKey, name, registry.READ)
		if err != nil {
			expChild.Close()
			return fmt.Errorf("missing restored subkey %q: %w", name, err)
		}
		cErr := compareRegistryTrees(expChild, actChild)
		expChild.Close()
		actChild.Close()
		if cErr != nil {
			return fmt.Errorf("in subkey %q: %w", name, cErr)
		}
	}

	return nil
}

func verifyRestoredTree(backupPath, targetPath string, hadKey bool) error {
	if !hadKey {
		key, err := registry.OpenKey(registry.CURRENT_USER, targetPath, registry.READ)
		if err == nil {
			key.Close()
			return fmt.Errorf("key %s was expected to be absent after rollback, but still exists", targetPath)
		}
		if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return fmt.Errorf("error verifying absent key %s: %w", targetPath, err)
		}
		return nil
	}

	bKey, err := registry.OpenKey(registry.CURRENT_USER, backupPath, registry.READ)
	if err != nil {
		return fmt.Errorf("failed to open backup key %s for comparison: %w", backupPath, err)
	}
	defer bKey.Close()

	tKey, err := registry.OpenKey(registry.CURRENT_USER, targetPath, registry.READ)
	if err != nil {
		return fmt.Errorf("failed to open restored key %s for comparison: %w", targetPath, err)
	}
	defer tKey.Close()

	return compareRegistryTrees(bKey, tKey)
}

// snapshotState captures the pre-existing state of the system before mutation.
type snapshotState struct {
	hadProductKey       bool
	productBackupKey    string
	previousPayloadPath string
	hadPreviousPayload  bool

	hadVerbKey       bool
	verbBackupKey    string

	hadUninstallKey    bool
	uninstallBackupKey string

	backupRootKey string
	hadShortcut   bool
	shortcutBytes []byte
	shortcutPath  string

	hadLauncher    bool
	launcherBytes  []byte
	launcherPath   string

	hadUninstaller bool
	uninstallerBytes []byte
	uninstallerPath  string
	pathSnap            *pathSnapshot
	hadLegacyPackage    bool
	finalPayloadCreated bool
	finalPayloadPath    string
	recoveryDir         string
}
func getStartMenuShortcutPath() string {
	programsDir := os.Getenv("APPDATA")
	if programsDir == "" {
		return ""
	}
	return filepath.Join(programsDir, "Microsoft", "Windows", "Start Menu", "Programs", "Upit.lnk")
}

func captureSnapshot(installDir string) (*snapshotState, error) {
	snap := &snapshotState{}
	token := rand.Text()
	backupRoot := transactionBackupKey + `\` + token
	snap.backupRootKey = backupRoot

	// 1. Snapshot Product Key (HKCU\Software\Upit)
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
	if err == nil {
		snap.hadProductKey = true
		snap.productBackupKey = backupRoot + `\Product`
		bKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, snap.productBackupKey, registry.ALL_ACCESS)
		if cErr != nil {
			prodKey.Close()
			return nil, fmt.Errorf("failed to create product backup key: %w", cErr)
		}
		cErr = copyRegistryTree(prodKey, bKey)
		bKey.Close()
		if cErr != nil {
			prodKey.Close()
			return nil, fmt.Errorf("failed to copy product tree to backup: %w", cErr)
		}
		val, kind, gErr := prodKey.GetStringValue("PayloadPath")
		if gErr == nil && kind == registry.SZ {
			// Validate previous payload: must be either committed version layout or valid legacy .tmp layout
			normVal := filepath.Clean(val)
			isLegacy := false
			if strings.HasSuffix(strings.ToLower(filepath.Base(normVal)), ".tmp") {
				if _, legErr := windowspayload.ValidateLegacyTmpPayload(normVal, installDir); legErr != nil {
					prodKey.Close()
					return nil, fmt.Errorf("invalid recorded legacy payload %s: %w", normVal, legErr)
				}
				isLegacy = true
			}
			snap.hadPreviousPayload = true
			snap.previousPayloadPath = normVal
			_ = isLegacy
		}
		prodKey.Close()
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil, fmt.Errorf("failed to open product key: %w", err)
	}

	// 2. Snapshot Verb Key
	verbKey, err := registry.OpenKey(registry.CURRENT_USER, classicVerbKey, registry.READ)
	if err == nil {
		snap.hadVerbKey = true
		snap.verbBackupKey = backupRoot + `\Verb`
		bKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, snap.verbBackupKey, registry.ALL_ACCESS)
		if cErr != nil {
			verbKey.Close()
			return nil, fmt.Errorf("failed to create verb backup key: %w", cErr)
		}
		cErr = copyRegistryTree(verbKey, bKey)
		bKey.Close()
		verbKey.Close()
		if cErr != nil {
			return nil, fmt.Errorf("failed to copy verb tree to backup: %w", cErr)
		}
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil, fmt.Errorf("failed to open verb key: %w", err)
	}

	// 3. Snapshot Uninstall Key
	uninstKey, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.READ)
	if err == nil {
		snap.hadUninstallKey = true
		snap.uninstallBackupKey = backupRoot + `\Uninstall`
		bKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, snap.uninstallBackupKey, registry.ALL_ACCESS)
		if cErr != nil {
			uninstKey.Close()
			return nil, fmt.Errorf("failed to create uninstall backup key: %w", cErr)
		}
		cErr = copyRegistryTree(uninstKey, bKey)
		bKey.Close()
		uninstKey.Close()
		if cErr != nil {
			return nil, fmt.Errorf("failed to copy uninstall tree to backup: %w", cErr)
		}
	} else if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return nil, fmt.Errorf("failed to open uninstall key: %w", err)
	}

	// 4. Snapshot Start Menu Shortcut
	shortcutPath := getStartMenuShortcutPath()
	snap.shortcutPath = shortcutPath
	if shortcutPath != "" {
		data, err := os.ReadFile(shortcutPath)
		if err == nil {
			snap.hadShortcut = true
			snap.shortcutBytes = data
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read start menu shortcut: %w", err)
		}
	}

	// 5. Snapshot Launcher
	launcherPath := filepath.Join(installDir, "upit.exe")
	snap.launcherPath = launcherPath
	lData, err := os.ReadFile(launcherPath)
	if err == nil {
		snap.hadLauncher = true
		snap.launcherBytes = lData
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read launcher: %w", err)
	}

	// 6. Snapshot Uninstaller
	uninstallerPath := filepath.Join(installDir, "Uninstall.exe")
	snap.uninstallerPath = uninstallerPath
	uData, err := os.ReadFile(uninstallerPath)
	if err == nil {
		snap.hadUninstaller = true
		snap.uninstallerBytes = uData
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read uninstaller: %w", err)
	}

	// 7. Snapshot PATH
	pSnap, err := capturePathSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to capture PATH snapshot: %w", err)
	}
	snap.pathSnap = pSnap

	// 8. Snapshot Legacy MSIX Package Presence
	legacyPresent, err := checkLegacyPackage(context.Background())
	if err != nil {
		snap.cleanupBackup()
		return nil, fmt.Errorf("cannot snapshot legacy package registration: %w", err)
	}
	snap.hadLegacyPackage = legacyPresent

	return snap, nil
}

func (s *snapshotState) cleanupBackup() {
	if s.backupRootKey != "" {
		_ = deleteRegistryTree(s.backupRootKey)
	}
}
func (s *snapshotState) rollback() error {
	var errs []error

	// 1. Rollback Verb
	if s.hadVerbKey && s.verbBackupKey != "" {
		_ = deleteRegistryTree(classicVerbKey)
		dstKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, classicVerbKey, registry.ALL_ACCESS)
		if cErr != nil {
			errs = append(errs, fmt.Errorf("failed to restore verb key: %w", cErr))
		} else {
			srcKey, oErr := registry.OpenKey(registry.CURRENT_USER, s.verbBackupKey, registry.READ)
			if oErr != nil {
				errs = append(errs, fmt.Errorf("failed to open verb backup key: %w", oErr))
			} else {
				if cpErr := copyRegistryTree(srcKey, dstKey); cpErr != nil {
					errs = append(errs, fmt.Errorf("failed to copy verb backup tree: %w", cpErr))
				}
				srcKey.Close()
			}
			dstKey.Close()
		}
	} else {
		if err := deleteRegistryTree(classicVerbKey); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete verb key during rollback: %w", err))
		}
	}
	notifyExplorer()
	if vErr := verifyRestoredTree(s.verbBackupKey, classicVerbKey, s.hadVerbKey); vErr != nil {
		errs = append(errs, fmt.Errorf("verb rollback verification failed: %w", vErr))
	}

	// 2. Rollback Product Key
	if s.hadProductKey && s.productBackupKey != "" {
		_ = deleteRegistryTree(productKeyPath)
		dstKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
		if cErr != nil {
			errs = append(errs, fmt.Errorf("failed to restore product key: %w", cErr))
		} else {
			srcKey, oErr := registry.OpenKey(registry.CURRENT_USER, s.productBackupKey, registry.READ)
			if oErr != nil {
				errs = append(errs, fmt.Errorf("failed to open product backup key: %w", oErr))
			} else {
				if cpErr := copyRegistryTree(srcKey, dstKey); cpErr != nil {
					errs = append(errs, fmt.Errorf("failed to copy product backup tree: %w", cpErr))
				}
				srcKey.Close()
			}
			dstKey.Close()
		}
	} else {
		if err := deleteRegistryTree(productKeyPath); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete product key during rollback: %w", err))
		}
	}
	if pErr := verifyRestoredTree(s.productBackupKey, productKeyPath, s.hadProductKey); pErr != nil {
		errs = append(errs, fmt.Errorf("product rollback verification failed: %w", pErr))
	}

	// 3. Rollback Uninstall Key
	if s.hadUninstallKey && s.uninstallBackupKey != "" {
		_ = deleteRegistryTree(uninstallKeyPath)
		dstKey, _, cErr := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.ALL_ACCESS)
		if cErr != nil {
			errs = append(errs, fmt.Errorf("failed to restore uninstall key: %w", cErr))
		} else {
			srcKey, oErr := registry.OpenKey(registry.CURRENT_USER, s.uninstallBackupKey, registry.READ)
			if oErr != nil {
				errs = append(errs, fmt.Errorf("failed to open uninstall backup key: %w", oErr))
			} else {
				if cpErr := copyRegistryTree(srcKey, dstKey); cpErr != nil {
					errs = append(errs, fmt.Errorf("failed to copy uninstall backup tree: %w", cpErr))
				}
				srcKey.Close()
			}
			dstKey.Close()
		}
	} else {
		if err := deleteRegistryTree(uninstallKeyPath); err != nil {
			errs = append(errs, fmt.Errorf("failed to delete uninstall key during rollback: %w", err))
		}
	}
	if uErr := verifyRestoredTree(s.uninstallBackupKey, uninstallKeyPath, s.hadUninstallKey); uErr != nil {
		errs = append(errs, fmt.Errorf("uninstall metadata rollback verification failed: %w", uErr))
	}

	// 4. Rollback Start Menu Shortcut
	if s.shortcutPath != "" {
		if s.hadShortcut {
			cur, rErr := os.ReadFile(s.shortcutPath)
			if rErr == nil && bytes.Equal(cur, s.shortcutBytes) {
				// Already identical to snapshot, no write needed (avoids failure if locked)
			} else {
				if wErr := os.WriteFile(s.shortcutPath, s.shortcutBytes, 0600); wErr != nil {
					errs = append(errs, fmt.Errorf("failed to restore shortcut file: %w", wErr))
				} else {
					readBack, vErr := os.ReadFile(s.shortcutPath)
					if vErr != nil || !bytes.Equal(readBack, s.shortcutBytes) {
						errs = append(errs, fmt.Errorf("shortcut verification failed after rollback (vErr=%v)", vErr))
					}
				}
			}
		} else {
			if rmErr := os.Remove(s.shortcutPath); rmErr != nil && !os.IsNotExist(rmErr) {
				errs = append(errs, fmt.Errorf("failed to remove shortcut during rollback: %w", rmErr))
			} else if _, statErr := os.Stat(s.shortcutPath); statErr == nil {
				errs = append(errs, fmt.Errorf("shortcut expected absent after rollback, but still exists"))
			} else if !os.IsNotExist(statErr) {
				errs = append(errs, fmt.Errorf("shortcut absence verification error after rollback: %w", statErr))
			}
		}
	}

	// 5. Rollback Uninstaller
	if s.uninstallerPath != "" {
		if s.hadUninstaller {
			cur, rErr := os.ReadFile(s.uninstallerPath)
			if rErr == nil && bytes.Equal(cur, s.uninstallerBytes) {
				// Already identical to snapshot, no write needed (avoids failure if locked)
			} else {
				if wErr := os.WriteFile(s.uninstallerPath, s.uninstallerBytes, 0700); wErr != nil {
					errs = append(errs, fmt.Errorf("failed to restore uninstaller file: %w", wErr))
				} else {
					readBack, vErr := os.ReadFile(s.uninstallerPath)
					if vErr != nil || !bytes.Equal(readBack, s.uninstallerBytes) {
						errs = append(errs, fmt.Errorf("uninstaller verification failed after rollback (vErr=%v)", vErr))
					}
				}
			}
		} else {
			if rmErr := os.Remove(s.uninstallerPath); rmErr != nil && !os.IsNotExist(rmErr) {
				errs = append(errs, fmt.Errorf("failed to remove uninstaller during rollback: %w", rmErr))
			} else if _, statErr := os.Stat(s.uninstallerPath); statErr == nil {
				errs = append(errs, fmt.Errorf("uninstaller expected absent after rollback, but still exists"))
			} else if !os.IsNotExist(statErr) {
				errs = append(errs, fmt.Errorf("uninstaller absence verification error after rollback: %w", statErr))
			}
		}
	}
	// 6. Rollback Launcher
	if s.launcherPath != "" {
		if s.hadLauncher {
			cur, rErr := os.ReadFile(s.launcherPath)
			if rErr == nil && bytes.Equal(cur, s.launcherBytes) {
				// Already identical to snapshot
			} else {
				if wErr := os.WriteFile(s.launcherPath, s.launcherBytes, 0700); wErr != nil {
					errs = append(errs, fmt.Errorf("failed to restore launcher file: %w", wErr))
				} else {
					readBack, vErr := os.ReadFile(s.launcherPath)
					if vErr != nil || !bytes.Equal(readBack, s.launcherBytes) {
						errs = append(errs, fmt.Errorf("launcher verification failed after rollback (vErr=%v)", vErr))
					}
				}
			}
		} else {
			if rmErr := os.Remove(s.launcherPath); rmErr != nil && !os.IsNotExist(rmErr) {
				errs = append(errs, fmt.Errorf("failed to remove launcher during rollback: %w", rmErr))
			} else if _, statErr := os.Stat(s.launcherPath); statErr == nil {
				errs = append(errs, fmt.Errorf("launcher expected absent after rollback, but still exists"))
			} else if !os.IsNotExist(statErr) {
				errs = append(errs, fmt.Errorf("launcher absence verification error after rollback: %w", statErr))
			}
		}
	}

	// 7. Rollback PATH
	if s.pathSnap != nil {
		if pErr := s.pathSnap.rollback(); pErr != nil {
			errs = append(errs, fmt.Errorf("PATH rollback failed: %w", pErr))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Restore Active Payload from recovery storage if same-version recovery was performed
	if s.recoveryDir != "" && s.finalPayloadPath != "" {
		if _, statErr := os.Stat(s.recoveryDir); statErr == nil {
			_ = os.RemoveAll(s.finalPayloadPath)
			if err := os.Rename(s.recoveryDir, s.finalPayloadPath); err != nil {
				return fmt.Errorf("failed to restore version directory from recovery %s: %w", s.recoveryDir, err)
			}
		}
	}

	// Discard tentative payload only after the prior callable state is proven restored.
	if s.finalPayloadCreated && s.finalPayloadPath != "" && s.recoveryDir == "" {
		if err := os.RemoveAll(s.finalPayloadPath); err != nil {
			return fmt.Errorf("restored previous state but could not remove tentative payload: %w", err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func validatePayloadFiles(payloadDir, version string) error {
	return windowspayload.ValidateStagedPayload(payloadDir, version)
}

func writeClassicVerb(payloadDir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, classicVerbKey, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create classic verb key: %w", err)
	}
	defer key.Close()

	if err = key.SetStringValue("", "Upload with Upit"); err != nil {
		return err
	}
	iconPath := fmt.Sprintf(`"%s",0`, filepath.Join(payloadDir, "upit-desktop.exe"))
	if err = key.SetStringValue("Icon", iconPath); err != nil {
		return err
	}
	if err = key.SetStringValue("MultiSelectModel", "Single"); err != nil {
		return err
	}

	cmdKey, _, err := registry.CreateKey(registry.CURRENT_USER, classicVerbCommand, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create classic verb command key: %w", err)
	}
	defer cmdKey.Close()

	cmdVal := fmt.Sprintf(`"%s" "%%1"`, filepath.Join(payloadDir, "upit-file-manager.exe"))
	if err = cmdKey.SetStringValue("", cmdVal); err != nil {
		return err
	}

	notifyExplorer()
	return nil
}

func writeUninstallMetadata(installDir, version string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, uninstallKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create uninstall key: %w", err)
	}
	defer key.Close()

	if err = key.SetStringValue("DisplayName", "Upit"); err != nil {
		return err
	}
	if version != "" {
		if err = key.SetStringValue("DisplayVersion", version); err != nil {
			return err
		}
	}
	uninstExe := filepath.Join(installDir, "Uninstall.exe")
	if err = key.SetStringValue("UninstallString", fmt.Sprintf(`"%s"`, uninstExe)); err != nil {
		return err
	}
	if err = key.SetDWordValue("NoModify", 1); err != nil {
		return err
	}
	if err = key.SetDWordValue("NoRepair", 1); err != nil {
		return err
	}
	return nil
}

func writeProductMetadata(payloadDir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create product key: %w", err)
	}
	defer key.Close()

	if err = key.SetStringValue("PayloadPath", payloadDir); err != nil {
		return err
	}
	return nil
}

func verifyInstallationRoutes(installDir, payloadDir, version string) error {
	// 1. Verify Payload files
	_, err := windowspayload.ValidateCommittedPayload(payloadDir, installDir, version)
	if err != nil {
		return fmt.Errorf("payload validation failed: %w", err)
	}
	// 2. Verify Product Key (PayloadPath must be registry.SZ and exact)
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
	if err != nil {
		return fmt.Errorf("verification failed: product key not found: %w", err)
	}
	defer prodKey.Close()

	buf := make([]byte, 2048)
	n, kind, err := prodKey.GetValue("PayloadPath", buf)
	if err != nil {
		return fmt.Errorf("verification failed: product PayloadPath not found: %w", err)
	}
	if kind != registry.SZ {
		return fmt.Errorf("verification failed: product PayloadPath kind mismatch: expected REG_SZ (1), got %d", kind)
	}
	pVal, _, _ := prodKey.GetStringValue("PayloadPath")
	_ = n
	if !strings.EqualFold(filepath.Clean(pVal), filepath.Clean(payloadDir)) {
		return fmt.Errorf("verification failed: product PayloadPath mismatch: %s != %s", pVal, payloadDir)
	}

	// Verify PATH ownership metadata in Product Key
	ownedVal, ownedKind, err := prodKey.GetIntegerValue(pathEntryOwnedName)
	if err != nil {
		return fmt.Errorf("verification failed: PathEntryOwned metadata missing: %w", err)
	}
	if ownedKind != registry.DWORD {
		return fmt.Errorf("verification failed: PathEntryOwned kind mismatch: expected DWORD (4), got %d", ownedKind)
	}
	_ = ownedVal // Present and valid DWORD

	// 3. Verify Classic Verb
	verbKey, err := registry.OpenKey(registry.CURRENT_USER, classicVerbKey, registry.READ)
	if err != nil {
		return fmt.Errorf("verification failed: classic verb key not found: %w", err)
	}
	label, _, _ := verbKey.GetStringValue("")
	icon, _, _ := verbKey.GetStringValue("Icon")
	msm, _, _ := verbKey.GetStringValue("MultiSelectModel")
	verbKey.Close()
	if label != "Upload with Upit" || msm != "Single" {
		return fmt.Errorf("verification failed: classic verb values mismatch: label=%q, msm=%q", label, msm)
	}
	expectedIcon := fmt.Sprintf(`"%s",0`, filepath.Join(payloadDir, "upit-desktop.exe"))
	if !strings.EqualFold(icon, expectedIcon) {
		return fmt.Errorf("verification failed: classic verb icon mismatch: got %q, want %q", icon, expectedIcon)
	}

	cmdKey, err := registry.OpenKey(registry.CURRENT_USER, classicVerbCommand, registry.READ)
	if err != nil {
		return fmt.Errorf("verification failed: classic verb command key not found: %w", err)
	}
	cmd, _, _ := cmdKey.GetStringValue("")
	cmdKey.Close()
	expectedCmd := fmt.Sprintf(`"%s" "%%1"`, filepath.Join(payloadDir, "upit-file-manager.exe"))
	if !strings.EqualFold(cmd, expectedCmd) {
		return fmt.Errorf("verification failed: classic verb command mismatch: got %q, want %q", cmd, expectedCmd)
	}

	// 4. Verify Start Menu Shortcut: existence, target path, and AUMID
	shortcutPath := getStartMenuShortcutPath()
	if shortcutPath == "" {
		return fmt.Errorf("verification failed: cannot determine start menu shortcut path")
	}
	if _, err := os.Stat(shortcutPath); err != nil {
		return fmt.Errorf("verification failed: start menu shortcut not found: %w", err)
	}

	expectedDesktop := filepath.Join(payloadDir, "upit-desktop.exe")
	inspectScript := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$wscript = New-Object -ComObject WScript.Shell
$sc = $wscript.CreateShortcut('%s')
$target = $sc.TargetPath
$app = New-Object -ComObject Shell.Application
$folder = $app.NameSpace((Split-Path '%s'))
$item = $folder.ParseName((Split-Path -Leaf '%s'))
$aumid = $item.ExtendedProperty('{9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3} 5')
Write-Output "$target|$aumid"
`, strings.ReplaceAll(shortcutPath, `'`, `''`), strings.ReplaceAll(shortcutPath, `'`, `''`), strings.ReplaceAll(shortcutPath, `'`, `''`))

	scCmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", inspectScript)
	scCmd.Dir = installDir
	scCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	scOut, scErr := scCmd.CombinedOutput()
	if scErr != nil {
		return fmt.Errorf("verification failed: inspect shortcut: %s: %w", strings.TrimSpace(string(scOut)), scErr)
	}
	actualTarget, actualAUMID, found := strings.Cut(strings.TrimSpace(string(scOut)), "|")
	if !found {
		return errors.New("verification failed: shortcut inspection returned malformed state")
	}
	if !strings.EqualFold(filepath.Clean(actualTarget), filepath.Clean(expectedDesktop)) {
		return fmt.Errorf("verification failed: shortcut target mismatch: got %q, want %q", actualTarget, expectedDesktop)
	}
	if actualAUMID != appUserModelID {
		return fmt.Errorf("verification failed: shortcut AUMID mismatch: got %q, want %q", actualAUMID, appUserModelID)
	}

	// 5. Verify Uninstaller executable and Uninstall metadata fields + kinds
	uninstPath := filepath.Join(installDir, "Uninstall.exe")
	if _, err := os.Stat(uninstPath); err != nil {
		return fmt.Errorf("verification failed: uninstaller not found: %w", err)
	}

	uninstKey, err := registry.OpenKey(registry.CURRENT_USER, uninstallKeyPath, registry.READ)
	if err != nil {
		return fmt.Errorf("verification failed: uninstall metadata key not found: %w", err)
	}
	defer uninstKey.Close()

	dName, dKind, err := uninstKey.GetStringValue("DisplayName")
	if err != nil || dKind != registry.SZ || dName != "Upit" {
		return fmt.Errorf("verification failed: DisplayName mismatch: %q (kind=%d, err=%v)", dName, dKind, err)
	}
	if version != "" {
		dVer, vKind, err := uninstKey.GetStringValue("DisplayVersion")
		if err != nil || vKind != registry.SZ || dVer != version {
			return fmt.Errorf("verification failed: DisplayVersion mismatch: %q (kind=%d, err=%v)", dVer, vKind, err)
		}
	}
	uStr, uKind, err := uninstKey.GetStringValue("UninstallString")
	expectedUStr := fmt.Sprintf(`"%s"`, uninstPath)
	if err != nil || uKind != registry.SZ || !strings.EqualFold(uStr, expectedUStr) {
		return fmt.Errorf("verification failed: UninstallString mismatch: %q (kind=%d, err=%v)", uStr, uKind, err)
	}
	noMod, mKind, err := uninstKey.GetIntegerValue("NoModify")
	if err != nil || mKind != registry.DWORD || noMod != 1 {
		return fmt.Errorf("verification failed: NoModify mismatch: %d (kind=%d, err=%v)", noMod, mKind, err)
	}
	noRep, rKind, err := uninstKey.GetIntegerValue("NoRepair")
	if err != nil || rKind != registry.DWORD || noRep != 1 {
		return fmt.Errorf("verification failed: NoRepair mismatch: %d (kind=%d, err=%v)", noRep, rKind, err)
	}

	// Verify obsolete package absent
	legacyRemains, pkgErr := checkLegacyPackage(context.Background())
	if pkgErr != nil {
		return fmt.Errorf("verification failed: cannot inspect obsolete Upit package registration: %w", pkgErr)
	}
	if legacyRemains {
		return fmt.Errorf("verification failed: obsolete legacy package remains registered")
	}

	// 6. Verify Root Launcher exists and executes --help
	launcherPath := filepath.Join(installDir, "upit.exe")
	if _, err := os.Stat(launcherPath); err != nil {
		return fmt.Errorf("verification failed: root launcher upit.exe not found: %w", err)
	}
	launcherCmd := exec.Command(launcherPath, "--help")
	launcherCmd.Dir = installDir
	launcherCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	lOut, lErr := launcherCmd.CombinedOutput()
	if lErr != nil {
		return fmt.Errorf("verification failed: launcher --help failed: %s (%w)", strings.TrimSpace(string(lOut)), lErr)
	}

	// 7. Verify PATH has installDir
	envKey, err := registry.OpenKey(registry.CURRENT_USER, environmentKeyPath, registry.READ)
	if err == nil {
		pathVal, _, _ := envKey.GetStringValue(pathValueName)
		if pathVal == "" {
			pathVal, _, _ = envKey.GetStringValue("Path")
		}
		envKey.Close()
		entries := splitPathEntries(pathVal)
		if !isPathInList(installDir, entries) {
			return fmt.Errorf("verification failed: installDir %s missing from User PATH", installDir)
		}
	}

	return nil
}

func runInstall(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	installDir := filepath.Clean(opts.InstallDir)
	stagedPayload := filepath.Clean(opts.StagedPayload)

	// Validate staged payload files before touching anything
	if err := validatePayloadFiles(stagedPayload, opts.Version); err != nil {
		return InstallResult{}, fmt.Errorf("invalid staged payload: %w", err)
	}

	// Capture snapshot of pre-existing state
	snap, err := captureSnapshot(installDir)
	if err != nil {
		return InstallResult{}, fmt.Errorf("failed to capture pre-install snapshot: %w", err)
	}

	retainBackup := false
	defer func() {
		if !retainBackup {
			snap.cleanupBackup()
		}
	}()

	compensate := func(cause error) error {
		// Check legacy package state during compensation
		curLegacy, legErr := checkLegacyPackage(context.Background())
		if legErr != nil || curLegacy != snap.hadLegacyPackage {
			// Registry/file compensation cannot restore changed or unknown package state.
			// Per ADR 0021, retain all recovery payloads and backup, fail clearly, do not claim clean rollback.
			retainBackup = true
			created := snap.finalPayloadCreated
			snap.finalPayloadCreated = false
			rbErr := snap.rollback()
			snap.finalPayloadCreated = created
			return errors.Join(cause, rbErr, fmt.Errorf("legacy package state changed or is unknown; rollback cannot restore the observed route; retaining recovery payloads (%s) and backup (%s)", stagedPayload, snap.backupRootKey))
		}

		rbErr := snap.rollback()
		if rbErr != nil {
			retainBackup = true
			return errors.Join(cause, fmt.Errorf("rollback failed; retain recovery payload %s: %w", stagedPayload, rbErr))
		}
		// Rollback succeeded completely: backup is no longer needed
		snap.cleanupBackup()
		return cause
	}

	// 1. Same-version reinstall vs different-version side-by-side
	finalPayloadDir := stagedPayload
	if opts.Version != "" {
		finalPayloadDir = filepath.Join(installDir, "versions", opts.Version)
		if !strings.EqualFold(stagedPayload, finalPayloadDir) {
			if err := os.MkdirAll(filepath.Dir(finalPayloadDir), 0700); err != nil {
				return InstallResult{}, compensate(fmt.Errorf("failed to create versions directory: %w", err))
			}

			// Check if same-version directory already exists
			if _, statErr := os.Stat(finalPayloadDir); statErr == nil {
				// Same-version reinstall!
				// Must ensure no processes are executing from the Active Payload before swap
				targetDirs := []string{finalPayloadDir}

				// Check launcher format requirement: if new launcher format differs, must also check root launcher
				dstLauncher := filepath.Join(installDir, "upit.exe")
				if opts.StagedLauncher != "" {
					if isLauncherFormatDifferent(dstLauncher, opts.StagedLauncher, installDir) {
						targetDirs = append(targetDirs, dstLauncher)
					}
				}

				if err := promptOrAbortRunningProcesses(targetDirs, opts.Silent); err != nil {
					return InstallResult{}, err // abort nonzero without mutating active state
				}

				// Perform recovery swap: move active versions\X.Y.Z to recovery directory
				token := rand.Text()
				recoveryRoot := filepath.Join(installDir, "recovery")
				_ = os.MkdirAll(recoveryRoot, 0700)
				recoveryDir := filepath.Join(recoveryRoot, token)
				if err := os.Rename(finalPayloadDir, recoveryDir); err != nil {
					return InstallResult{}, fmt.Errorf("failed to move current version directory to recovery storage: %w", err)
				}
				snap.recoveryDir = recoveryDir
				snap.finalPayloadPath = finalPayloadDir

				if err := os.Rename(stagedPayload, finalPayloadDir); err != nil {
					return InstallResult{}, compensate(fmt.Errorf("failed to move staged replacement to %s: %w", finalPayloadDir, err))
				}
			} else if os.IsNotExist(statErr) {
				// Fresh install or different-version side-by-side
				if err := os.Rename(stagedPayload, finalPayloadDir); err != nil {
					return InstallResult{}, compensate(fmt.Errorf("failed to move staged payload to %s: %w", finalPayloadDir, err))
				}
				snap.finalPayloadCreated = true
				snap.finalPayloadPath = finalPayloadDir
			} else {
				return InstallResult{}, compensate(fmt.Errorf("failed to stat final payload path %s: %w", finalPayloadDir, statErr))
			}
		}
	}

	// 2. Publish staged launcher if needed (preserve format 1 launcher if identical)
	if opts.StagedLauncher != "" {
		dstLauncher := filepath.Join(installDir, "upit.exe")
		shouldCopyLauncher := isLauncherFormatDifferent(dstLauncher, opts.StagedLauncher, installDir)
		if shouldCopyLauncher {
			if err := copyFile(opts.StagedLauncher, dstLauncher); err != nil {
				return InstallResult{}, compensate(fmt.Errorf("failed to publish stable launcher: %w", err))
			}
		}
	}

	// 3. Ensure installDir in User PATH
	if _, err := ensureInstallDirInPath(installDir); err != nil {
		return InstallResult{}, compensate(fmt.Errorf("failed to update User PATH: %w", err))
	}

	// 4. Publish staged uninstaller if provided
	if opts.StagedUninstaller != "" {
		dstUninstaller := filepath.Join(installDir, "Uninstall.exe")
		if err := copyFile(opts.StagedUninstaller, dstUninstaller); err != nil {
			return InstallResult{}, compensate(fmt.Errorf("failed to publish uninstaller: %w", err))
		}
	}

	// 5. Publish Classic Verb via staged file manager helper --install-integration
	// This preserves full integration lifecycle and legacy package removal/compensation guarantees.
	helperExe := filepath.Join(finalPayloadDir, "upit-file-manager.exe")
	cmd := exec.CommandContext(ctx, helperExe, "--install-integration")
	cmd.Dir = installDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return InstallResult{}, compensate(fmt.Errorf("failed to run helper integration install: %s (%w)", strings.TrimSpace(string(out)), err))
	}
	// Publish staged shortcut if provided
	if opts.StagedShortcut != "" {
		shortcutPath := getStartMenuShortcutPath()
		if shortcutPath != "" {
			if err := copyFile(opts.StagedShortcut, shortcutPath); err != nil {
				return InstallResult{}, compensate(fmt.Errorf("failed to publish start menu shortcut: %w", err))
			}
		}
	}

	// Record previous payload as InactivePayloads candidate DURING precommit transaction
	// if it is a different payload residing within the installDir.
	if snap.hadPreviousPayload && snap.previousPayloadPath != "" {
		cleanPrev := filepath.Clean(snap.previousPayloadPath)
		cleanInstallRoot := strings.ToLower(filepath.Clean(installDir)) + string(filepath.Separator)
		if !strings.EqualFold(cleanPrev, finalPayloadDir) && strings.HasPrefix(strings.ToLower(cleanPrev), cleanInstallRoot) {
			if err := recordInactivePayload(cleanPrev); err != nil {
				return InstallResult{}, compensate(fmt.Errorf("failed to record inactive payload metadata: %w", err))
			}
		}
	}

	// Publish Uninstall metadata
	if err := writeUninstallMetadata(installDir, opts.Version); err != nil {
		return InstallResult{}, compensate(fmt.Errorf("failed to write uninstall metadata: %w", err))
	}
	// Verify all published routes
	if err := verifyInstallationRoutes(installDir, finalPayloadDir, opts.Version); err != nil {
		return InstallResult{}, compensate(fmt.Errorf("installation route verification failed: %w", err))
	}

	// Transaction committed successfully!
	// Attempt cleanup of recorded inactive payload candidates (including previous payload).
	cleanInstallRoot := strings.ToLower(filepath.Clean(installDir)) + string(filepath.Separator)
	cleanActive := strings.ToLower(filepath.Clean(finalPayloadDir))

	inactives, err := getInactivePayloads()
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnostic: failed to read InactivePayloads metadata: %v\n", err)
	}

	deferredCleanup := false
	for _, p := range inactives {
		cleanP := filepath.Clean(p)
		lowerP := strings.ToLower(cleanP)
		if lowerP != cleanActive && strings.HasPrefix(lowerP, cleanInstallRoot) {
			if err := os.RemoveAll(cleanP); err != nil {
				// File or directory is locked: remain recorded, signal deferred cleanup outcome
				deferredCleanup = true
				fmt.Fprintf(os.Stderr, "diagnostic: inactive payload at %s is locked; deferred cleanup retained\n", cleanP)
			} else {
				// Successfully removed: unregister from InactivePayloads
				if uErr := unregisterInactivePayload(cleanP); uErr != nil {
					// Removal from journal failed: do NOT undo committed route, report deferred cleanup
					deferredCleanup = true
					fmt.Fprintf(os.Stderr, "diagnostic: deleted %s but failed to unregister from InactivePayloads: %v\n", cleanP, uErr)
				}
			}
		}
	}

	// Clean up recovery storage on successful commit
	if snap.recoveryDir != "" {
		_ = os.RemoveAll(snap.recoveryDir)
	}

	return InstallResult{DeferredCleanup: deferredCleanup}, nil
}

func getInactivePayloads() ([]string, error) {
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.READ)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open product key to read InactivePayloads: %w", err)
	}
	defer prodKey.Close()

	val, kind, err := prodKey.GetStringsValue(inactivePayloadsVal)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read InactivePayloads: %w", err)
	}
	if kind != registry.MULTI_SZ {
		return nil, fmt.Errorf("InactivePayloads kind mismatch: expected MULTI_SZ, got %d", kind)
	}
	return val, nil
}

func recordInactivePayload(p string) error {
	prodKey, _, err := registry.CreateKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to open product key to write InactivePayloads: %w", err)
	}
	defer prodKey.Close()

	existing, _, err := prodKey.GetStringsValue(inactivePayloadsVal)
	if err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return fmt.Errorf("failed to read existing InactivePayloads: %w", err)
	}
	normP := strings.ToLower(filepath.Clean(p))
	for _, item := range existing {
		if strings.ToLower(filepath.Clean(item)) == normP {
			return nil // already recorded
		}
	}
	existing = append(existing, filepath.Clean(p))
	if err := prodKey.SetStringsValue(inactivePayloadsVal, existing); err != nil {
		return fmt.Errorf("failed to write InactivePayloads: %w", err)
	}
	return nil
}

func unregisterInactivePayload(p string) error {
	prodKey, err := registry.OpenKey(registry.CURRENT_USER, productKeyPath, registry.ALL_ACCESS)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil
		}
		return fmt.Errorf("failed to open product key to unregister InactivePayload: %w", err)
	}
	defer prodKey.Close()

	existing, _, err := prodKey.GetStringsValue(inactivePayloadsVal)
	if err != nil {
		if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return nil
		}
		return fmt.Errorf("failed to read InactivePayloads: %w", err)
	}
	normP := strings.ToLower(filepath.Clean(p))
	var remaining []string
	for _, item := range existing {
		if strings.ToLower(filepath.Clean(item)) != normP {
			remaining = append(remaining, item)
		}
	}
	if len(remaining) == 0 {
		if err := prodKey.DeleteValue(inactivePayloadsVal); err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
			return fmt.Errorf("failed to delete empty InactivePayloads value: %w", err)
		}
		return nil
	}
	if err := prodKey.SetStringsValue(inactivePayloadsVal, remaining); err != nil {
		return fmt.Errorf("failed to update InactivePayloads: %w", err)
	}
	return nil
}

func checkLegacyPackage(ctx context.Context) (bool, error) {
	script := `$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; @(Get-AppxPackage -Name 'HungNth.Upit').Count`
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("failed to query legacy package: %w", err)
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "0" {
		return false, nil
	}
	return true, nil
}

func runUninstall(ctx context.Context, opts UninstallOptions) error {
	installDir := filepath.Clean(opts.InstallDir)

	// Step 1: Remove File Manager Integration via staged helper --uninstall-integration first
	// (fail-closed if integration removal fails)
	helperExe := filepath.Join(installDir, "upit-file-manager.exe")
	if _, err := os.Stat(helperExe); err != nil {
		vDirs, _ := filepath.Glob(filepath.Join(installDir, "versions", "*", "upit-file-manager.exe"))
		if len(vDirs) > 0 {
			helperExe = vDirs[0]
		}
	}
	cmd := exec.CommandContext(ctx, helperExe, "--uninstall-integration")
	cmd.Dir = installDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unregister file manager integration: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	// Double check that Classic Verb key is really gone
	vKey, err := registry.OpenKey(registry.CURRENT_USER, classicVerbKey, registry.READ)
	if err == nil {
		vKey.Close()
		return fmt.Errorf("classic verb still present after helper uninstall-integration")
	}

	// Step 2: Remove Start Menu shortcut
	shortcutPath := getStartMenuShortcutPath()
	if shortcutPath != "" {
		_ = os.Remove(shortcutPath)
	}

	// Step 3: Remove stable launcher
	_ = os.Remove(filepath.Join(installDir, "upit.exe"))

	// Step 4: Remove installDir from User PATH only if installer-owned
	_ = removeInstallDirFromPath(installDir)

	// Step 5: Read and remove recorded InactivePayloads while product metadata is still present
	cleanInstallRoot := strings.ToLower(filepath.Clean(installDir)) + string(filepath.Separator)
	if inactives, err := getInactivePayloads(); err == nil {
		for _, p := range inactives {
			cleanP := filepath.Clean(p)
			if strings.HasPrefix(strings.ToLower(cleanP), cleanInstallRoot) {
				_ = os.RemoveAll(cleanP)
			}
		}
	}

	// Step 6: Remove Uninstall metadata
	if err := deleteRegistryTree(uninstallKeyPath); err != nil {
		return fmt.Errorf("failed to remove uninstall metadata: %w", err)
	}

	// Step 7: Clean up transaction backup keys scoped to this installDir
	cleanTransactionBackups()

	// Step 8: Remove Product key (HKCU\Software\Upit)
	if err := deleteRegistryTree(productKeyPath); err != nil {
		return fmt.Errorf("failed to remove product key: %w", err)
	}

	// Step 9: Remove staging, recovery, versions directories and install root files
	_ = os.RemoveAll(filepath.Join(installDir, "staging"))
	_ = os.RemoveAll(filepath.Join(installDir, "recovery"))
	_ = os.RemoveAll(filepath.Join(installDir, "versions"))
	_ = os.Remove(filepath.Join(installDir, "Uninstall.exe"))
	_ = os.Remove(installDir) // removes if empty

	return nil
}

func cleanTransactionBackups() {
	bkKey, err := registry.OpenKey(registry.CURRENT_USER, transactionBackupKey, registry.READ)
	if err != nil {
		return
	}
	subkeys, err := bkKey.ReadSubKeyNames(-1)
	bkKey.Close()
	if err != nil {
		return
	}
	for _, sub := range subkeys {
		_ = deleteRegistryTree(transactionBackupKey + `\` + sub)
	}
	_ = deleteRegistryTree(transactionBackupKey)
}

type processEntry32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [windows.MAX_PATH]uint16
}

func checkRunningProcesses(installDir string) ([]ProcessInfo, error) {
	cleanInstallDir := strings.ToLower(filepath.Clean(installDir)) + string(filepath.Separator)

	hSnap, _, err := procCreateToolhelp32Snapshot.Call(0x00000002, 0) // TH32CS_SNAPPROCESS
	if hSnap == uintptr(windows.InvalidHandle) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(windows.Handle(hSnap))

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, nil
	}

	var results []ProcessInfo
	currentPID := windows.GetCurrentProcessId()

	for {
		if entry.ProcessID != currentPID && entry.ProcessID != 0 && entry.ProcessID != 4 {
			hProc, oErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if oErr == nil {
				var buf [windows.MAX_PATH]uint16
				size := uint32(len(buf))
				r, _, _ := procQueryFullProcessImageNameW.Call(
					uintptr(hProc),
					0,
					uintptr(unsafe.Pointer(&buf[0])),
					uintptr(unsafe.Pointer(&size)),
				)
				windows.CloseHandle(hProc)
				if r != 0 {
					imagePath := windows.UTF16ToString(buf[:size])
					cleanImage := strings.ToLower(filepath.Clean(imagePath))
					if strings.HasPrefix(cleanImage, cleanInstallDir) {
						results = append(results, ProcessInfo{
							PID:       entry.ProcessID,
							ImagePath: imagePath,
						})
					}
				}
			}
		}

		ret, _, _ = procProcess32NextW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return results, nil
}

func isLauncherFormatDifferent(installedLauncher, stagedLauncher, installDir string) bool {
	if _, err := os.Stat(installedLauncher); err != nil {
		return true // not installed yet
	}
	stagedFmt := getLauncherFormat(stagedLauncher, filepath.Dir(stagedLauncher))
	installedFmt := getLauncherFormat(installedLauncher, installDir)

	if stagedFmt == "" {
		stagedFmt = "1"
	}
	if installedFmt == "" {
		installedFmt = "1"
	}
	return stagedFmt != installedFmt
}

func getLauncherFormat(exePath, dir string) string {
	chkCmd := exec.Command(exePath, "--launcher-format")
	chkCmd.Dir = dir
	chkCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := chkCmd.CombinedOutput()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func promptOrAbortRunningProcesses(paths []string, silent bool) error {
	for {
		var running []ProcessInfo
		for _, p := range paths {
			procs, err := checkRunningProcessesForPath(p)
			if err != nil {
				return fmt.Errorf("process check error: %w", err)
			}
			running = append(running, procs...)
		}

		if len(running) == 0 {
			return nil
		}

		if silent {
			var procDetails []string
			for _, r := range running {
				procDetails = append(procDetails, fmt.Sprintf("PID %d: %s", r.PID, r.ImagePath))
			}
			return fmt.Errorf("cannot replace while processes are executing: %s", strings.Join(procDetails, "; "))
		}

		// Interactive: prompt Retry / Cancel
		msg := "Upit files are currently in use by running processes:\n"
		for _, r := range running {
			msg += fmt.Sprintf(" - %s (PID %d)\n", filepath.Base(r.ImagePath), r.PID)
		}
		msg += "\nPlease close these processes and click Retry, or Cancel to abort."

		pMsg, _ := windows.UTF16PtrFromString(msg)
		pTitle, _ := windows.UTF16PtrFromString("Upit - Files in Use")
		ret, _, _ := procMessageBoxW.Call(
			0,
			uintptr(unsafe.Pointer(pMsg)),
			uintptr(unsafe.Pointer(pTitle)),
			0x00000005|0x00000030, // MB_RETRYCANCEL | MB_ICONWARNING
		)
		if ret == 4 { // IDRETRY
			continue
		}
		return errors.New("installation cancelled by user due to running processes")
	}
}

func checkRunningProcessesForPath(targetPath string) ([]ProcessInfo, error) {
	cleanTarget := strings.ToLower(filepath.Clean(targetPath))
	realTarget, evalErr := filepath.EvalSymlinks(cleanTarget)
	if evalErr == nil {
		cleanTarget = strings.ToLower(filepath.Clean(realTarget))
	}

	hSnap, _, err := procCreateToolhelp32Snapshot.Call(0x00000002, 0)
	if hSnap == uintptr(windows.InvalidHandle) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed: %w", err)
	}
	defer windows.CloseHandle(windows.Handle(hSnap))

	var entry processEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32FirstW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, nil
	}

	var results []ProcessInfo
	currentPID := windows.GetCurrentProcessId()

	for {
		if entry.ProcessID != currentPID && entry.ProcessID != 0 && entry.ProcessID != 4 {
			hProc, oErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ProcessID)
			if oErr == nil {
				var buf [windows.MAX_PATH]uint16
				size := uint32(len(buf))
				r, _, _ := procQueryFullProcessImageNameW.Call(
					uintptr(hProc),
					0,
					uintptr(unsafe.Pointer(&buf[0])),
					uintptr(unsafe.Pointer(&size)),
				)
				windows.CloseHandle(hProc)
				if r != 0 {
					imagePath := windows.UTF16ToString(buf[:size])
					cleanImage := strings.ToLower(filepath.Clean(imagePath))
					realImage, rEvalErr := filepath.EvalSymlinks(cleanImage)
					if rEvalErr == nil {
						cleanImage = strings.ToLower(filepath.Clean(realImage))
					}
					isMatch := false
					if strings.EqualFold(cleanImage, cleanTarget) {
						isMatch = true
					} else {
						cleanTargetDir := cleanTarget + string(filepath.Separator)
						if strings.HasPrefix(cleanImage, cleanTargetDir) {
							isMatch = true
						}
					}
					if isMatch {
						results = append(results, ProcessInfo{
							PID:       entry.ProcessID,
							ImagePath: imagePath,
						})
					}
				}
			}
		}

		ret, _, _ = procProcess32NextW.Call(hSnap, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return results, nil
}
