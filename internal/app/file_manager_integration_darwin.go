//go:build darwin && cgo

package app

/*
#cgo LDFLAGS: -framework AppKit -framework CoreServices
#include <stdbool.h>
#include <stdlib.h>
bool upitIntegrationRegistered(const char *, const char *);
void upitIntegrationRefresh(void);
*/
import "C"

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

const launchServicesRegister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

type macIntegrationAdapter struct{ executable func() (string, error) }

func init() { nativeIntegrationAdapter = macIntegrationAdapter{executable: os.Executable} }
func (a macIntegrationAdapter) bundle() (string, error) {
	p, err := a.executable()
	if err != nil {
		return "", err
	}
	p, err = filepath.EvalSymlinks(p)
	if err != nil {
		return "", err
	}
	if filepath.Base(filepath.Dir(p)) != "MacOS" || filepath.Base(filepath.Dir(filepath.Dir(p))) != "Contents" {
		return "", errors.New("Desktop is not installed in Upit.app")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(p)))
	if filepath.Base(root) != "Upit.app" {
		return "", errors.New("Desktop is not installed in Upit.app")
	}
	return root, nil
}
func registeredMacBundle(identifier, path string) bool {
	id := C.CString(identifier)
	defer C.free(unsafe.Pointer(id))
	p := C.CString(path)
	defer C.free(unsafe.Pointer(p))
	return bool(C.upitIntegrationRegistered(id, p))
}
func (a macIntegrationAdapter) inspect(ctx context.Context) (FileManagerIntegrationState, error) {
	state := FileManagerIntegrationState{Status: IntegrationReinstall, Guidance: "Reinstall Upit by dragging Upit.app into Applications. Verify Upload with Upit in Finder; registration does not prove menu visibility. Enable it in System Settings → Keyboard → Keyboard Shortcuts → Services → Files and Folders.", Actions: []string{"open-settings", "open-file-manager"}}
	root, err := a.bundle()
	if err != nil {
		return state, nil
	}
	service := filepath.Join(root, "Contents", "Helpers", "UpitFinderService.app")
	worker := filepath.Join(service, "Contents", "Helpers", "UpitFileManager.app")
	bundles := []struct{ path, id, executable string }{{root, "com.hungnth.upit.desktop", "upit-desktop"}, {service, "com.hungnth.upit.finder-service", "UpitFinderService"}, {worker, "com.hungnth.upit.file-manager", "upit-file-manager"}}
	var version string
	for _, b := range bundles {
		info, e := os.Stat(filepath.Join(b.path, "Contents", "MacOS", b.executable))
		if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			return state, nil
		}
		for _, field := range []string{"CFBundleIdentifier", "CFBundleShortVersionString"} {
			out, e := exec.CommandContext(ctx, "/usr/libexec/PlistBuddy", "-c", "Print :"+field, filepath.Join(b.path, "Contents", "Info.plist")).Output()
			if e != nil {
				return state, nil
			}
			value := strings.TrimSpace(string(out))
			if field == "CFBundleIdentifier" && value != b.id {
				return state, nil
			}
			if field == "CFBundleShortVersionString" {
				if value == "" {
					return state, nil
				}
				if version == "" {
					version = value
				} else if value != version {
					return state, nil
				}
			}
		}
	}
	if _, err = os.Stat(launchServicesRegister); err != nil {
		return state, nil
	}
	state.Actions = append(state.Actions, "prepare-removal")
	state.Guidance = "Verify Upload with Upit in Finder Services or Quick Actions. Registration does not prove menu visibility. If needed, enable it in System Settings → Keyboard → Keyboard Shortcuts → Services → Files and Folders."
	if registeredMacBundle("com.hungnth.upit.desktop", root) && registeredMacBundle("com.hungnth.upit.finder-service", service) {
		state.Status = IntegrationRegistered
	} else {
		state.Status = IntegrationNeedsRepair
		state.Actions = append(state.Actions, "repair")
	}
	return state, nil
}
func (a macIntegrationAdapter) act(ctx context.Context, action string) error {
	switch action {
	case "open-settings":
		return exec.CommandContext(ctx, "/usr/bin/open", "x-apple.systempreferences:com.apple.Keyboard-Settings.extension").Run()
	case "open-file-manager":
		return exec.CommandContext(ctx, "/usr/bin/open", "-a", "Finder").Run()
	case "repair", "prepare-removal":
		root, err := a.bundle()
		if err != nil {
			return err
		}
		service := filepath.Join(root, "Contents", "Helpers", "UpitFinderService.app")
		flag := "-f"
		paths := []string{root, service}
		if action == "prepare-removal" {
			flag = "-u"
			paths = []string{service, root}
		}
		for _, p := range paths {
			if err = exec.CommandContext(ctx, launchServicesRegister, flag, p).Run(); err != nil {
				return err
			}
		}
		C.upitIntegrationRefresh()
		if action == "prepare-removal" && (registeredMacBundle("com.hungnth.upit.desktop", root) || registeredMacBundle("com.hungnth.upit.finder-service", service)) {
			return errors.New("Upit registration is still discoverable; the application was not prepared for removal")
		}
		return nil
	default:
		return errors.New("unknown macOS integration action")
	}
}
