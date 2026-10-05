package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var registryCopyTree = windows.NewLazySystemDLL("advapi32.dll").NewProc("RegCopyTreeW")
var registryDeleteTree = windows.NewLazySystemDLL("advapi32.dll").NewProc("RegDeleteTreeW")

func deleteRegistryTree(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	result, _, _ := registryDeleteTree.Call(uintptr(registry.CURRENT_USER), uintptr(unsafe.Pointer(name)))
	if result != 0 && result != uintptr(windows.ERROR_FILE_NOT_FOUND) {
		return windows.Errno(result)
	}
	return nil
}
func copyRegistryTree(source, destination registry.Key) error {
	result, _, _ := registryCopyTree.Call(uintptr(source), 0, uintptr(destination))
	if result != 0 {
		return windows.Errno(result)
	}
	return nil
}

// Installation snapshots the entire previous verb, not just expected values: compensation
// must restore the observed route even when its values or subkeys are stale.
func (a windowsIntegrationAdapter) install(ctx context.Context) error {
	root, intact, err := a.payload()
	if err != nil {
		return err
	}
	if !intact {
		return errors.New("required Upit payload is missing; installation was not activated")
	}
	metadataPath := a.metadataPath
	if metadataPath == "" {
		metadataPath = `Software\Upit`
	}
	metadata, hadMetadata, err := registry.CreateKey(registry.CURRENT_USER, metadataPath, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	removeMetadata := !hadMetadata
	defer func() {
		metadata.Close()
		if removeMetadata {
			deleteRegistryTree(metadataPath)
		}
	}()
	previous, kind, err := metadata.GetStringValue("PayloadPath")
	hadPrevious := err == nil
	if err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return err
	}
	if hadPrevious && kind != registry.SZ {
		return errors.New("invalid previous Upit payload metadata")
	}
	backupPath := metadataPath + `\IntegrationRollback-` + rand.Text()
	backup, _, err := registry.CreateKey(registry.CURRENT_USER, backupPath, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	old, err := registry.OpenKey(registry.CURRENT_USER, a.key(), registry.READ)
	hadVerb := err == nil
	if err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		backup.Close()
		deleteRegistryTree(backupPath)
		return err
	}
	if hadVerb {
		err = copyRegistryTree(old, backup)
		old.Close()
		if err != nil {
			backup.Close()
			deleteRegistryTree(backupPath)
			return err
		}
	}
	retainBackup := false
	removeMetadata = false
	defer func() {
		backup.Close()
		if !retainBackup {
			deleteRegistryTree(backupPath)
		}
	}()
	compensate := func(cause error) error {
		restoreErr := deleteRegistryTree(a.key())
		if restoreErr == nil && hadVerb {
			restored, _, createErr := registry.CreateKey(registry.CURRENT_USER, a.key(), registry.ALL_ACCESS)
			if createErr != nil {
				restoreErr = createErr
			} else {
				restoreErr = copyRegistryTree(backup, restored)
				restored.Close()
			}
		}
		var metadataErr error
		if hadPrevious {
			metadataErr = metadata.SetStringValue("PayloadPath", previous)
		} else {
			metadataErr = metadata.DeleteValue("PayloadPath")
			if errors.Is(metadataErr, windows.ERROR_FILE_NOT_FOUND) {
				metadataErr = nil
			}
		}
		notifyExplorer()
		if restoreErr != nil || metadataErr != nil {
			retainBackup = true
			return errors.Join(cause, fmt.Errorf("registration compensation failed; retain recovery payload and reinstall Upit: %w", errors.Join(restoreErr, metadataErr)))
		}
		removeMetadata = !hadMetadata
		return cause
	}
	if err = a.writeVerb(root); err != nil {
		return compensate(err)
	}
	if err = ensureStartMenuShortcut(filepath.Join(root, "upit-desktop.exe")); err != nil {
		return compensate(err)
	}
	if err = metadata.SetStringValue("PayloadPath", root); err != nil {
		return compensate(err)
	}
	remains, err := a.legacyPackage(ctx, true)
	if err != nil {
		// The old package may already have been removed. Do not discard the verified
		// callable replacement, previous payload, or snapshot while state is unknown.
		retainBackup = true
		return errors.New("obsolete package state is unknown; cutover unresolved, recovery payload retained")
	}
	if remains {
		return compensate(errors.New("obsolete Upit package remains; previous registration restored, installation aborted"))
	}
	return nil
}
