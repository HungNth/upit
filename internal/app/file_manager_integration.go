package app

import (
	"context"
	"errors"
	"runtime"
)

type IntegrationStatus string

const (
	IntegrationRegistered  IntegrationStatus = "Registered"
	IntegrationNeedsRepair IntegrationStatus = "Needs Repair"
	IntegrationReinstall   IntegrationStatus = "Reinstall Upit"
	IntegrationUnsupported IntegrationStatus = "Not supported on Linux"
)

type FileManagerIntegrationState struct {
	Status   IntegrationStatus `json:"status"`
	Guidance string            `json:"guidance"`
	Actions  []string          `json:"actions"`
}
type integrationAdapter interface {
	inspect(context.Context) (FileManagerIntegrationState, error)
	act(context.Context, string) error
}
type FileManagerIntegrationService struct {
	platform string
	adapter  integrationAdapter
}

var nativeIntegrationAdapter integrationAdapter

func NewFileManagerIntegrationService() FileManagerIntegrationService {
	return FileManagerIntegrationService{platform: runtime.GOOS, adapter: nativeIntegrationAdapter}
}
func (s FileManagerIntegrationService) Inspect(ctx context.Context) (FileManagerIntegrationState, error) {
	platform := s.platform
	if platform == "" {
		platform = runtime.GOOS
	}
	if platform == "linux" {
		return FileManagerIntegrationState{Status: IntegrationUnsupported, Guidance: "File Manager Integration is not supported on Linux. Manual Upload and Configuration Set management remain available.", Actions: []string{}}, nil
	}
	if s.adapter == nil {
		return FileManagerIntegrationState{}, errors.New("File Manager Integration platform adapter is unavailable")
	}
	return s.adapter.inspect(ctx)
}
func (s FileManagerIntegrationService) Act(ctx context.Context, action string) (FileManagerIntegrationState, error) {
	state, err := s.Inspect(ctx)
	if err != nil {
		return state, err
	}
	allowed := false
	for _, a := range state.Actions {
		if a == action {
			allowed = true
			break
		}
	}
	if !allowed {
		return state, errors.New("File Manager Integration action is unavailable")
	}
	if err = s.adapter.act(ctx, action); err != nil {
		return state, err
	}
	state, err = s.Inspect(ctx)
	if err == nil && action == "repair" && state.Status != IntegrationRegistered {
		err = errors.New("registration is still unavailable after Repair; reinstall Upit if the problem persists")
	}
	if err == nil && action == "prepare-removal" && state.Status == IntegrationRegistered {
		err = errors.New("registration is still discoverable; Upit was not prepared for removal")
	}
	return state, err
}

// RunFileManagerIntegrationInstaller is the private installer's current-user lifecycle entry point.
// It never loads the Configuration Set or starts Desktop or an upload.
func RunFileManagerIntegrationInstaller(ctx context.Context, remove bool) error {
	if runtime.GOOS != "windows" || nativeIntegrationAdapter == nil {
		return errors.New("this integration installer lifecycle requires Windows")
	}
	action := "repair"
	if remove {
		action = "remove-registration"
	}
	if err := nativeIntegrationAdapter.act(ctx, action); err != nil {
		return err
	}
	state, err := nativeIntegrationAdapter.inspect(ctx)
	if err != nil {
		return err
	}
	if (!remove && state.Status != IntegrationRegistered) || (remove && state.Status == IntegrationRegistered) {
		return errors.New("Windows integration lifecycle did not reach the expected registration state")
	}
	return nil
}
