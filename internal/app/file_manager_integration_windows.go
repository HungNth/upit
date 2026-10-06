package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/HungNth/upit/internal/windowspayload"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const classicVerbKey = `Software\Classes\*\shell\Upit.Upload`

var explorerAssociationChanged = windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")

type windowsIntegrationAdapter struct {
	executable   func() (string, error)
	registryPath string
	metadataPath string
	legacy       func(bool) (bool, error)
}

func init() { nativeIntegrationAdapter = windowsIntegrationAdapter{executable: os.Executable} }
func (a windowsIntegrationAdapter) key() string {
	if a.registryPath != "" {
		return a.registryPath
	}
	return classicVerbKey
}

func (a windowsIntegrationAdapter) payload() (string, bool, error) {
	executable, err := a.executable()
	if err != nil {
		return "", false, err
	}
	dir := filepath.Dir(executable)
	if a.metadataPath != "" {
		if err := windowspayload.ValidateFixturePayload(dir); err != nil {
			return dir, false, nil
		}
		return dir, true, nil
	}
	return a.inspectActivePayload(dir)
}

func (a windowsIntegrationAdapter) inspectActivePayload(root string) (string, bool, error) {
	if a.metadataPath != "" {
		if err := windowspayload.ValidateFixturePayload(root); err != nil {
			return root, false, nil
		}
		return root, true, nil
	}
	// Production validation: expected install root is derived from a.executable()
	// (e.g. <installRoot>\versions\<oldOrNewVersion>\upit-desktop.exe -> parent of parent is installRoot)
	exePath, err := a.executable()
	if err != nil {
		return root, false, err
	}
	expectedInstallRoot := resolveInstallRoot(exePath)
	_, err = windowspayload.ValidateCommittedPayload(root, expectedInstallRoot, "")
	if err != nil {
		if strings.HasSuffix(strings.ToLower(filepath.Base(root)), ".tmp") {
			if _, legErr := windowspayload.ValidateLegacyTmpPayload(root, expectedInstallRoot); legErr == nil {
				return root, true, nil
			}
		}
		return root, false, nil
	}
	return root, true, nil
}
func resolveInstallRoot(exePath string) string {
	cur := filepath.Dir(exePath)
	for cur != "" {
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		if strings.EqualFold(filepath.Base(cur), "versions") || strings.EqualFold(filepath.Base(cur), "staging") {
			return parent
		}
		cur = parent
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(exePath)))
}


func (a windowsIntegrationAdapter) activePayload() (string, bool, error) {
	path := a.metadataPath
	if path == "" {
		path = `Software\Upit`
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.READ)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer key.Close()
	root, kind, err := key.GetStringValue("PayloadPath")
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if kind != registry.SZ || !filepath.IsAbs(root) {
		return "", false, nil
	}
	return a.inspectActivePayload(root)
}
func (a windowsIntegrationAdapter) legacyPackage(ctx context.Context, remove bool) (bool, error) {
	if a.legacy != nil {
		return a.legacy(remove)
	}
	script := `$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; @(Get-AppxPackage -Name 'HungNth.Upit').Count`
	if remove {
		script = `$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; $p=@(Get-AppxPackage -Name 'HungNth.Upit'); foreach($item in $p){try{Remove-AppxPackage -Package $item.PackageFullName}catch{}}; @(Get-AppxPackage -Name 'HungNth.Upit').Count`
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, errors.New("cannot inspect obsolete Upit package registration")
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || count < 0 {
		return false, errors.New("unexpected obsolete Upit package state")
	}
	return count != 0, nil
}
func (a windowsIntegrationAdapter) matches(root string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, a.key(), registry.READ)
	if err != nil {
		return false
	}
	defer k.Close()
	info, err := k.Stat()
	if err != nil || info.ValueCount != 3 || info.SubKeyCount != 1 {
		return false
	}
	for name, want := range map[string]string{"": "Upload with Upit", "Icon": `"` + filepath.Join(root, "upit-desktop.exe") + `",0`, "MultiSelectModel": "Single"} {
		got, kind, err := k.GetStringValue(name)
		if err != nil || kind != registry.SZ || got != want {
			return false
		}
	}
	command, err := registry.OpenKey(k, "command", registry.READ)
	if err != nil {
		return false
	}
	defer command.Close()
	info, err = command.Stat()
	if err != nil || info.ValueCount != 1 || info.SubKeyCount != 0 {
		return false
	}
	got, kind, err := command.GetStringValue("")
	return err == nil && kind == registry.SZ && got == `"`+filepath.Join(root, "upit-file-manager.exe")+`" "%1"`
}
func (a windowsIntegrationAdapter) inspect(ctx context.Context) (FileManagerIntegrationState, error) {
	root, intact, err := a.activePayload()
	if err != nil {
		return FileManagerIntegrationState{}, err
	}
	state := FileManagerIntegrationState{Status: IntegrationReinstall, Guidance: "Reinstall Upit. Verify Upload with Upit under Show more options in File Explorer; registration does not prove live menu visibility.", Actions: []string{"open-file-manager"}}
	if !intact {
		return state, nil
	}
	legacy, err := a.legacyPackage(ctx, false)
	if err != nil {
		return state, err
	}
	state.Guidance = "Verify Upload with Upit under Show more options in File Explorer. Registration does not prove live menu visibility."
	if a.matches(root) && !legacy {
		state.Status = IntegrationRegistered
	} else {
		state.Status = IntegrationNeedsRepair
		state.Actions = append(state.Actions, "repair")
	}
	return state, nil
}
func notifyExplorer() {
	explorerAssociationChanged.Call(0x08000000, 0, 0, 0)
}
func (a windowsIntegrationAdapter) removeVerb() error {
	if err := deleteRegistryTree(a.key()); err != nil {
		return fmt.Errorf("remove Upit Classic Verb: %w", err)
	}
	notifyExplorer()
	k, err := registry.OpenKey(registry.CURRENT_USER, a.key(), registry.READ)
	if err == nil {
		k.Close()
		return errors.New("Upit Classic Verb remains registered")
	}
	if !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return err
	}
	return nil
}
func (a windowsIntegrationAdapter) act(ctx context.Context, action string) error {
	if action == "open-file-manager" {
		return exec.CommandContext(ctx, "explorer.exe").Start()
	}
	if action == "install" {
		return a.install(ctx)
	}
	if action == "remove-registration" {
		if err := a.removeVerb(); err != nil {
			return err
		}
		remains, err := a.legacyPackage(ctx, true)
		if err != nil {
			return err
		}
		if remains {
			return errors.New("obsolete Upit package remains registered")
		}
		if err := removeStartMenuShortcut(); err != nil {
			return err
		}
		return nil
	}
	if action != "repair" {
		return errors.New("unknown integration action")
	}
	root, intact, err := a.activePayload()
	if err != nil {
		return err
	}
	if !intact {
		return errors.New("required Upit payload is missing; reinstall Upit")
	}
	if err = a.writeVerb(root); err != nil {
		return err
	}
	remains, err := a.legacyPackage(ctx, true)
	if err != nil {
		return err
	}
	if remains {
		return errors.New("obsolete Upit package cleanup failed")
	}
	return ensureStartMenuShortcut(filepath.Join(root, "upit-desktop.exe"))
}

func (a windowsIntegrationAdapter) writeVerb(root string) error {
	if err := deleteRegistryTree(a.key()); err != nil {
		return err
	}
	k, _, err := registry.CreateKey(registry.CURRENT_USER, a.key(), registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer k.Close()
	for name, value := range map[string]string{"": "Upload with Upit", "Icon": `"` + filepath.Join(root, "upit-desktop.exe") + `",0`, "MultiSelectModel": "Single"} {
		if err = k.SetStringValue(name, value); err != nil {
			return err
		}
	}
	command, _, err := registry.CreateKey(k, "command", registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	err = command.SetStringValue("", `"`+filepath.Join(root, "upit-file-manager.exe")+`" "%1"`)
	command.Close()
	if err != nil {
		return err
	}
	notifyExplorer()
	if !a.matches(root) {
		return errors.New("Classic Verb verification failed")
	}
	return nil
}
