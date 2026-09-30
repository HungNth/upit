package main

import (
	"context"
	"embed"
	"log"
	"sync"
	"sync/atomic"

	"github.com/HungNth/upit/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

type desktopService struct {
	core                     app.Service
	dirty                    atomic.Bool
	allowClose               atomic.Bool
	manualUploadActive       atomic.Bool
	manualUploadMu           sync.Mutex
	manualUploadCancel       context.CancelFunc
	manualUploadDone         chan struct{}
	emitManualUploadProgress app.ManualUploadProgressFunc
	close                    func()
}

func (s *desktopService) StartupState() (app.DesktopStartupState, error) {
	return s.core.DesktopStartupState()
}

func (s *desktopService) LoadGlobalConfigurationEditor() (app.GlobalConfigurationEditorState, error) {
	return s.core.LoadGlobalConfigurationEditor()
}

func (s *desktopService) SaveGlobalConfiguration(draft app.GlobalConfigurationDraft) (app.GlobalConfigurationEditorState, error) {
	return s.core.SaveGlobalConfiguration(draft)
}

func (s *desktopService) LoadUploaderEditor(name string) (app.UploaderEditorState, error) {
	return s.core.LoadUploaderEditor(name)
}

func (s *desktopService) SaveUploaderEditor(draft app.UploaderEditorDraft) (app.UploaderEditorState, error) {
	return s.core.SaveUploaderEditor(draft)
}

func (s *desktopService) RenameUploader(draft app.UploaderRenameDraft) (app.UploaderEditorState, error) {
	return s.core.RenameUploader(draft)
}

func (s *desktopService) DeleteUploader(draft app.UploaderDeleteDraft) (app.UploaderEditorState, error) {
	return s.core.DeleteUploader(draft)
}

func (s *desktopService) LoadShortenerEditor(name string) (app.ShortenerEditorState, error) {
	return s.core.LoadShortenerEditor(name)
}

func (s *desktopService) SaveShortenerEditor(draft app.ShortenerEditorDraft) (app.ShortenerEditorState, error) {
	return s.core.SaveShortenerEditor(draft)
}

func (s *desktopService) RenameShortener(draft app.ShortenerRenameDraft) (app.ShortenerEditorState, error) {
	return s.core.RenameShortener(draft)
}

func (s *desktopService) DeleteShortener(draft app.ShortenerDeleteDraft) (app.ShortenerEditorState, error) {
	return s.core.DeleteShortener(draft)
}

func (s *desktopService) CreateInitialConfigurationSet(global app.GlobalConfigurationDraft, uploader app.UploaderEditorDraft) (app.DesktopStartupState, error) {
	return s.core.CreateInitialConfigurationSet(global, uploader)
}

func (s *desktopService) SetGlobalConfigurationDirty(dirty bool) {
	s.dirty.Store(dirty)
}

func (s *desktopService) LoadRepairDocument(kind app.RepairDocumentKind) (app.RepairDocumentState, error) {
	return s.core.LoadRepairDocument(kind)
}

func (s *desktopService) SaveRepairDocument(draft app.RepairDocumentDraft) (app.DesktopStartupState, error) {
	return s.core.SaveRepairDocument(draft)
}

func (s *desktopService) ConfirmClose() {
	s.allowClose.Store(true)
	if s.close != nil {
		s.close()
	}
}

func (s *desktopService) ChooseManualUploadFile(ctx context.Context) (app.ManualUploadSelection, error) {
	dialog := application.Get().Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		SetTitle("Select a file for Manual Upload")
	if window, ok := ctx.Value(application.WindowKey).(application.Window); ok {
		dialog.AttachToWindow(window)
	}
	filePath, err := dialog.PromptForSingleSelection()
	if err != nil || filePath == "" {
		return app.ManualUploadSelection{}, err
	}
	return s.core.SelectManualUploadFile(filePath)
}

func (s *desktopService) PrepareManualUploadFile(filePath string) (app.ManualUploadSelection, error) {
	return s.core.SelectManualUploadFile(filePath)
}

func (s *desktopService) StartManualUpload(ctx context.Context, options app.ManualUploadOptions) app.ManualUploadResult {
	if s.dirty.Load() {
		return app.ManualUploadResult{Failure: &app.ManualUploadFailure{
			Stage:   "validation",
			Message: "save or discard Configuration Set edits before starting Manual Upload",
		}}
	}
	if !s.manualUploadActive.CompareAndSwap(false, true) {
		return app.ManualUploadResult{Failure: &app.ManualUploadFailure{
			Stage:   "validation",
			Message: "a Manual Upload is already active",
		}}
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.manualUploadMu.Lock()
	s.manualUploadCancel = cancel
	s.manualUploadDone = done
	s.manualUploadMu.Unlock()
	defer func() {
		cancel()
		s.manualUploadMu.Lock()
		if s.manualUploadDone == done {
			s.manualUploadCancel = nil
			s.manualUploadDone = nil
		}
		s.manualUploadMu.Unlock()
		s.manualUploadActive.Store(false)
		close(done)
	}()
	return s.core.ManualUploadWithProgress(runCtx, options, s.emitManualUploadProgress)
}

func (s *desktopService) CancelManualUpload(ctx context.Context) bool {
	s.manualUploadMu.Lock()
	cancel := s.manualUploadCancel
	done := s.manualUploadDone
	s.manualUploadMu.Unlock()
	if cancel == nil || done == nil {
		return false
	}
	cancel()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *desktopService) CopyManualUploadFinalURL(ctx context.Context, finalURL string) error {
	return s.core.CopyManualUploadFinalURL(ctx, finalURL)
}

func (s *desktopService) RetryClose() {
	if s.close != nil {
		s.close()
	}
}

type windowFocusTarget interface {
	Restore()
	Focus()
}

func restoreAndFocus(window windowFocusTarget) {
	if window == nil {
		return
	}
	window.Restore()
	window.Focus()
}

func main() {
	core := app.Service{}
	service := &desktopService{core: core}
	var window application.Window

	desktop := application.New(application.Options{
		Name:        "upit-desktop",
		Description: "Upit Desktop",
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.hungnth.upit.desktop",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				restoreAndFocus(window)
			},
		},
	})
	service.emitManualUploadProgress = func(progress app.ManualUploadProgress) {
		desktop.Event.Emit("desktop:manual-upload-progress", progress)
	}

	window = desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "upit-desktop-main",
		Title:            "Upit Desktop",
		Width:            1120,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		URL:              "/",
		InitialPosition:  application.WindowCentered,
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGB(9, 9, 11),
	})
	service.close = window.Close
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		desktop.Event.Emit("desktop:manual-upload-files-dropped", event.Context().DroppedFiles())
	})
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if service.manualUploadActive.Load() {
			event.Cancel()
			desktop.Event.Emit("desktop:manual-upload-close-requested")
			return
		}
		if !service.dirty.Load() || service.allowClose.Load() {
			return
		}
		event.Cancel()
		desktop.Event.Emit("desktop:close-requested")
	})

	if err := desktop.Run(); err != nil {
		log.Fatal(err)
	}
}
