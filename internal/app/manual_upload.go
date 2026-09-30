package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ManualUploadSelection describes one regular file available for Manual Upload.
type ManualUploadSelection struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// ManualUploadOptions contains the one-invocation choices available in Manual Upload.
type ManualUploadOptions struct {
	FilePath          string `json:"filePath"`
	Uploader          string `json:"uploader"`
	Shortener         string `json:"shortener"`
	DisableShortening bool   `json:"disableShortening"`
	Clipboard         string `json:"clipboard"`
	Timeout           string `json:"timeout"`
}

// ManualUploadFailure is a redacted failure suitable for desktop presentation.
type ManualUploadFailure struct {
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitzero"`
	Canceled   bool   `json:"canceled"`
}

// ManualUploadResult is the terminal result of one Manual Upload.
type ManualUploadResult struct {
	Success     bool                 `json:"success"`
	OriginalURL string               `json:"originalURL"`
	FinalURL    string               `json:"finalURL"`
	Warnings    []string             `json:"warnings"`
	Failure     *ManualUploadFailure `json:"failure"`
}

// ManualUploadProgress describes observable work that contains no configuration or response values.
type ManualUploadProgress struct {
	Phase     string `json:"phase"`
	Processed int64  `json:"processed"`
	Total     int64  `json:"total"`
}

// ManualUploadProgressFunc receives ordered, privacy-safe Manual Upload updates.
type ManualUploadProgressFunc func(ManualUploadProgress)

// SelectManualUploadFile validates one path before it enters Manual Upload state.
func (s Service) SelectManualUploadFile(filePath string) (ManualUploadSelection, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return ManualUploadSelection{}, failure("validation", "selected file cannot be inspected", err)
	}
	if !info.Mode().IsRegular() {
		return ManualUploadSelection{}, failure("validation", "upload path must be a regular file", nil)
	}
	return ManualUploadSelection{Path: filePath, Name: filepath.Base(filePath), Size: info.Size()}, nil
}

// ManualUpload runs one upload after validating the current Configuration Set and options.
func (s Service) ManualUpload(ctx context.Context, options ManualUploadOptions) ManualUploadResult {
	return s.ManualUploadWithProgress(ctx, options, nil)
}

// ManualUploadWithProgress runs one upload and reports privacy-safe lifecycle updates.
func (s Service) ManualUploadWithProgress(ctx context.Context, options ManualUploadOptions, progress ManualUploadProgressFunc) ManualUploadResult {
	emitManualUploadProgress(progress, ManualUploadProgress{Phase: "preparing"})
	if _, err := s.SelectManualUploadFile(options.FilePath); err != nil {
		return failedManualUpload(ctx, options.FilePath, err)
	}
	startup, err := s.DesktopStartupState()
	if err != nil {
		return failedManualUpload(ctx, options.FilePath, err)
	}
	if startup.Mode != DesktopStartupNormal {
		return ManualUploadResult{Failure: &ManualUploadFailure{
			Stage:   "validation",
			Message: "Manual Upload requires a valid Configuration Set",
		}}
	}
	clipboard, err := manualClipboardOverride(options.Clipboard)
	if err != nil {
		return failedManualUpload(ctx, options.FilePath, err)
	}
	if options.Timeout != "" {
		timeout, err := time.ParseDuration(options.Timeout)
		if err != nil || timeout <= 0 {
			return ManualUploadResult{Failure: &ManualUploadFailure{
				Stage:   "validation",
				Message: "timeout must be a positive Go duration",
			}}
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	outcome, err := s.upload(ctx, UploadOptions{
		FilePath:          options.FilePath,
		Uploader:          options.Uploader,
		Shortener:         options.Shortener,
		DisableShortening: options.DisableShortening,
		Clipboard:         clipboard,
	}, progress)
	if err != nil {
		return failedManualUpload(ctx, options.FilePath, err)
	}
	return ManualUploadResult{
		Success:     true,
		OriginalURL: outcome.Result.OriginalURL,
		FinalURL:    outcome.Result.FinalURL,
		Warnings:    outcome.Warnings,
	}
}

func manualClipboardOverride(value string) (ClipboardOverride, error) {
	switch value {
	case "", "default":
		return ClipboardFromConfig, nil
	case "enabled":
		return ClipboardEnabled, nil
	case "disabled":
		return ClipboardDisabled, nil
	default:
		return ClipboardFromConfig, failuref("validation", nil, "clipboard override %q is invalid", value)
	}
}

func failedManualUpload(ctx context.Context, filePath string, err error) ManualUploadResult {
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return ManualUploadResult{Failure: &ManualUploadFailure{
			Stage:    "canceled",
			Message:  "Manual Upload canceled",
			Canceled: true,
		}}
	}
	var uploadFailure *Failure
	if errors.As(err, &uploadFailure) {
		return ManualUploadResult{Failure: &ManualUploadFailure{
			Stage:      uploadFailure.Stage,
			Message:    redactManualUploadPath(uploadFailure.Message, filePath),
			StatusCode: uploadFailure.StatusCode,
		}}
	}
	return ManualUploadResult{Failure: &ManualUploadFailure{
		Stage:   "upload",
		Message: redactManualUploadPath(fmt.Sprintf("Manual Upload failed: %v", err), filePath),
	}}
}

func redactManualUploadPath(message, filePath string) string {
	if filePath == "" {
		return message
	}
	return strings.ReplaceAll(message, filePath, "selected file")
}

// CopyManualUploadFinalURL copies one validated Final URL for an explicit desktop action.
func (s Service) CopyManualUploadFinalURL(ctx context.Context, finalURL string) error {
	if !isHTTPURL(finalURL) {
		return failure("validation", "Final URL must be an absolute HTTP(S) URL", nil)
	}
	copier := s.Clipboard
	if copier == nil {
		copier = SystemClipboard{}
	}
	if err := copier.Copy(ctx, finalURL); err != nil {
		return failuref("clipboard", err, "copy Final URL: %v", err)
	}
	return nil
}
