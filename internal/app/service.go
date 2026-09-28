package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
)

type ClipboardOverride uint8

const (
	ClipboardFromConfig ClipboardOverride = iota
	ClipboardEnabled
	ClipboardDisabled
)

type UploadOptions struct {
	FilePath  string
	Uploader  string
	Clipboard ClipboardOverride
}

type Outcome struct {
	Result   Result
	Warnings []string
}

type Service struct {
	HomeDir   func() (string, error)
	Client    *http.Client
	Clipboard Clipboard
}

func (s Service) Upload(ctx context.Context, options UploadOptions) (Outcome, error) {
	homeDir := s.HomeDir
	if homeDir == nil {
		homeDir = os.UserHomeDir
	}
	home, err := homeDir()
	if err != nil {
		return Outcome{}, failuref("config", err, "resolve user home directory: %v", err)
	}

	global, uploaders, err := loadConfiguration(home)
	if err != nil {
		return Outcome{}, err
	}
	uploaderName := options.Uploader
	if uploaderName == "" {
		uploaderName = global.DefaultUploader
	}
	selected, ok := uploaders.Uploaders[uploaderName]
	if !ok {
		return Outcome{}, failuref("validation", nil, "uploader %q does not exist", uploaderName)
	}

	client := *http.DefaultClient
	if s.Client != nil {
		client = *s.Client
	}
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	result, err := uploadFile(ctx, &client, options.FilePath, selected)
	if err != nil {
		return Outcome{}, err
	}

	copyResult := global.CopyToClipboard
	switch options.Clipboard {
	case ClipboardEnabled:
		copyResult = true
	case ClipboardDisabled:
		copyResult = false
	}
	outcome := Outcome{Result: result}
	if !copyResult {
		return outcome, nil
	}
	copier := s.Clipboard
	if copier == nil {
		copier = SystemClipboard{}
	}
	if err := copier.Copy(ctx, result.FinalURL); err != nil {
		outcome.Warnings = append(outcome.Warnings, fmt.Sprintf("copy to clipboard: %v", err))
	}
	return outcome, nil
}
