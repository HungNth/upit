package app

import (
	"context"
	"errors"
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
	FilePath          string
	Uploader          string
	Shortener         string
	DisableShortening bool
	Clipboard         ClipboardOverride
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

	shortenerName := options.Shortener
	if options.DisableShortening {
		shortenerName = ""
	} else if shortenerName == "" {
		shortenerName = global.DefaultShortener
	}
	var selectedShortener *shortener
	if shortenerName != "" {
		shorteners, err := loadShortenerConfiguration(home)
		if err != nil {
			return Outcome{}, err
		}
		candidate, ok := shorteners.Shorteners[shortenerName]
		if !ok {
			return Outcome{}, failuref("validation", nil, "shortener %q does not exist", shortenerName)
		}
		selectedShortener = &candidate
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
	warnings := []string(nil)
	if selectedShortener != nil {
		finalURL, shortenErr := shortenURL(ctx, &client, result.OriginalURL, *selectedShortener)
		if shortenErr != nil {
			if errors.Is(shortenErr, context.Canceled) && errors.Is(ctx.Err(), context.Canceled) {
				return Outcome{}, failure("shortener", "URL shortening canceled", shortenErr)
			}
			warnings = append(warnings, fmt.Sprintf("shorten URL: %v", shortenErr))
		} else {
			result.FinalURL = finalURL
		}
	}

	copyResult := global.CopyToClipboard
	switch options.Clipboard {
	case ClipboardEnabled:
		copyResult = true
	case ClipboardDisabled:
		copyResult = false
	}
	outcome := Outcome{Result: result, Warnings: warnings}
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
