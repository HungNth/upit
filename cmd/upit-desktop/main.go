package main

import (
	"embed"
	"log"
	"sync/atomic"

	"github.com/HungNth/upit/internal/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

type desktopService struct {
	core       app.Service
	dirty      atomic.Bool
	allowClose atomic.Bool
	close      func()
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

	window = desktop.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "upit-desktop-main",
		Title:            "Upit Desktop",
		Width:            1120,
		Height:           720,
		MinWidth:         900,
		MinHeight:        600,
		URL:              "/",
		InitialPosition:  application.WindowCentered,
		BackgroundColour: application.NewRGB(15, 23, 42),
	})
	service.close = window.Close
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
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
