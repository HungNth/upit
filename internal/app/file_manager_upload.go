package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

type FileManagerUploadStatus string

const (
	FileManagerUploadSucceeded FileManagerUploadStatus = "succeeded"
	FileManagerUploadFailed    FileManagerUploadStatus = "failed"
	FileManagerUploadCanceled  FileManagerUploadStatus = "canceled"
	FileManagerUploadRejected  FileManagerUploadStatus = "rejected"
)

type FileManagerActionKind string

const (
	FileManagerActionCopyFinalURL FileManagerActionKind = "copy-final-url"
	FileManagerActionRetry        FileManagerActionKind = "retry"
	FileManagerActionOpenDesktop  FileManagerActionKind = "open-desktop"
)

type FileManagerAction struct {
	Kind  FileManagerActionKind `json:"kind"`
	Token string                `json:"token"`
}

type FileManagerUploadFailure struct {
	Stage      string `json:"stage"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitzero"`
	Canceled   bool   `json:"canceled"`
}

type FileManagerUploadResult struct {
	Status   FileManagerUploadStatus   `json:"status"`
	Warnings []string                  `json:"warnings,omitzero"`
	Failure  *FileManagerUploadFailure `json:"failure,omitzero"`
	Actions  []FileManagerAction       `json:"actions,omitzero"`
}

type FileManagerActionResult struct {
	Kind   FileManagerActionKind
	Upload *FileManagerUploadResult
}

type FileManagerUploadService struct {
	core           Service
	mu             sync.Mutex
	active         bool
	actionLifetime time.Duration
	uploadLock     fileManagerUploadLock
}

type fileManagerActionState struct {
	kind      FileManagerActionKind
	filePath  string
	finalURL  string
	expiresAt time.Time
}

func NewFileManagerUploadService(core Service) *FileManagerUploadService {
	return &FileManagerUploadService{
		core:           core,
		actionLifetime: 10 * time.Minute,
		uploadLock:     newFileManagerUploadLock(),
	}
}

func (s *FileManagerUploadService) Upload(ctx context.Context, paths []string, progress ManualUploadProgressFunc) FileManagerUploadResult {
	return s.upload(ctx, paths, progress, true)
}

func (s *FileManagerUploadService) upload(ctx context.Context, paths []string, progress ManualUploadProgressFunc, allowRetry bool) FileManagerUploadResult {
	if !s.begin() {
		return FileManagerUploadResult{
			Status: FileManagerUploadRejected,
			Failure: &FileManagerUploadFailure{
				Stage:   "validation",
				Message: "a File Manager Upload is already active",
			},
		}
	}
	defer s.end()

	if len(paths) != 1 {
		return rejectedFileManagerUpload("exactly one regular file is required")
	}
	selection, err := s.core.SelectManualUploadFile(paths[0])
	if err != nil {
		return rejectedFileManagerUpload("selected path is not a regular file")
	}
	expected, err := snapshotFileManagerUpload(paths[0], selection)
	if err != nil {
		return rejectedFileManagerUpload("selected path changed before upload")
	}

	homeDir, err := s.homeDirectory()
	if err != nil {
		return s.failed(ctx, paths[0], failure("config", "resolve user home directory", err), allowRetry)
	}
	release, acquired, err := s.uploadLock.TryAcquire(homeDir)
	if err != nil {
		return s.failed(ctx, paths[0], failure("config", "acquire File Manager Upload lock", err), allowRetry)
	}
	if !acquired {
		return rejectedFileManagerUpload("a File Manager Upload is already active")
	}
	defer release()

	emitManualUploadProgress(progress, ManualUploadProgress{Phase: "preparing"})
	outcome, err := s.core.uploadWithSnapshot(ctx, UploadOptions{
		FilePath:  paths[0],
		Clipboard: ClipboardFromConfig,
	}, &expected, progress)
	if err != nil {
		return s.failed(ctx, paths[0], err, allowRetry)
	}

	result := FileManagerUploadResult{
		Status:   FileManagerUploadSucceeded,
		Warnings: sanitizeFileManagerWarnings(outcome.Warnings),
	}
	if !outcome.Copied {
		if action, actionErr := s.addAction(fileManagerActionState{
			kind:      FileManagerActionCopyFinalURL,
			finalURL:  outcome.Result.FinalURL,
			expiresAt: time.Now().Add(s.actionLifetime),
		}); actionErr == nil {
			result.Actions = append(result.Actions, action)
		} else {
			result.Warnings = append(result.Warnings, "Final URL copy action unavailable")
		}
	}
	return result
}

func (s *FileManagerUploadService) Dispatch(ctx context.Context, token string, progress ManualUploadProgressFunc) (FileManagerActionResult, error) {
	state, ok, err := s.takeAction(token)
	if err != nil {
		return FileManagerActionResult{}, err
	}
	if !ok {
		return FileManagerActionResult{}, errors.New("File Manager Upload action is missing or expired")
	}
	switch state.kind {
	case FileManagerActionCopyFinalURL:
		if err := s.core.CopyManualUploadFinalURL(ctx, state.finalURL); err != nil {
			return FileManagerActionResult{}, err
		}
		return FileManagerActionResult{Kind: state.kind}, nil
	case FileManagerActionRetry:
		retryResult := s.upload(ctx, []string{state.filePath}, progress, false)
		return FileManagerActionResult{Kind: state.kind, Upload: &retryResult}, nil
	default:
		return FileManagerActionResult{Kind: state.kind}, nil
	}
}

func (s *FileManagerUploadService) failed(ctx context.Context, filePath string, err error, allowRetry bool) FileManagerUploadResult {
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return FileManagerUploadResult{
			Status: FileManagerUploadCanceled,
			Failure: &FileManagerUploadFailure{
				Stage:    "canceled",
				Message:  "File Manager Upload canceled",
				Canceled: true,
			},
		}
	}
	result := FileManagerUploadResult{
		Status: FileManagerUploadFailed,
		Failure: &FileManagerUploadFailure{
			Stage:   "upload",
			Message: "File Manager Upload failed",
		},
	}
	var uploadFailure *Failure
	if errors.As(err, &uploadFailure) {
		result.Failure.Stage = uploadFailure.Stage
		result.Failure.StatusCode = uploadFailure.StatusCode
		if isFileManagerConfigurationFailure(uploadFailure) {
			result.Failure.Message = "Configuration Set is unavailable or invalid"
			if action, actionErr := s.addAction(fileManagerActionState{
				kind:      FileManagerActionOpenDesktop,
				expiresAt: time.Now().Add(s.actionLifetime),
			}); actionErr == nil {
				result.Actions = append(result.Actions, action)
			} else {
				result.Warnings = append(result.Warnings, "Configuration recovery action unavailable")
			}
			return result
		}
	}
	if allowRetry {
		if action, actionErr := s.addAction(fileManagerActionState{
			kind:      FileManagerActionRetry,
			filePath:  filePath,
			expiresAt: time.Now().Add(s.actionLifetime),
		}); actionErr == nil {
			result.Actions = append(result.Actions, action)
		} else {
			result.Warnings = append(result.Warnings, "Retry action unavailable")
		}
	}
	return result
}

func isFileManagerConfigurationFailure(failure *Failure) bool {
	if failure.Stage == "config" {
		return true
	}
	if failure.Stage != "validation" {
		return false
	}
	for _, marker := range []string{"upload file", "upload path", "selected file", "UTF-8"} {
		if strings.Contains(failure.Message, marker) {
			return false
		}
	}
	return true
}

func sanitizeFileManagerWarnings(warnings []string) []string {
	result := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		switch {
		case strings.HasPrefix(warning, "shorten URL:"):
			result = append(result, "URL shortening was unavailable; the Original URL was retained")
		case strings.HasPrefix(warning, "copy to clipboard:"):
			result = append(result, "Final URL was not copied to the clipboard")
		default:
			result = append(result, "Upload completed with a warning")
		}
	}
	return result
}

func (s *FileManagerUploadService) addAction(state fileManagerActionState) (FileManagerAction, error) {
	homeDir, err := s.homeDirectory()
	if err != nil {
		return FileManagerAction{}, err
	}
	return diskFileManagerActionStore{}.Put(homeDir, state)
}

func (s *FileManagerUploadService) takeAction(token string) (fileManagerActionState, bool, error) {
	homeDir, err := s.homeDirectory()
	if err != nil {
		return fileManagerActionState{}, false, err
	}
	return diskFileManagerActionStore{}.Consume(homeDir, token)
}
func (s *FileManagerUploadService) DiscardActions(result FileManagerUploadResult) error {
	if len(result.Actions) == 0 {
		return nil
	}
	homeDir, err := s.homeDirectory()
	if err != nil {
		return err
	}
	store := diskFileManagerActionStore{}
	for _, action := range result.Actions {
		if _, _, err := store.Consume(homeDir, action.Token); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileManagerUploadService) homeDirectory() (string, error) {
	if s.core.HomeDir != nil {
		return s.core.HomeDir()
	}
	return os.UserHomeDir()
}

func snapshotFileManagerUpload(path string, selection ManualUploadSelection) (uploadFileSnapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return uploadFileSnapshot{}, err
	}
	if !info.Mode().IsRegular() || info.Size() != selection.Size {
		return uploadFileSnapshot{}, errors.New("selected file changed")
	}
	return uploadFileSnapshot{info: info}, nil
}

func (s *FileManagerUploadService) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		return false
	}
	s.active = true
	return true
}

func (s *FileManagerUploadService) end() {
	s.mu.Lock()
	s.active = false
	s.mu.Unlock()
}

func rejectedFileManagerUpload(message string) FileManagerUploadResult {
	return FileManagerUploadResult{
		Status: FileManagerUploadRejected,
		Failure: &FileManagerUploadFailure{
			Stage:   "validation",
			Message: message,
		},
	}
}
