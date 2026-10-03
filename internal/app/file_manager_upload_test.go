package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileManagerUploadRejectsAnythingExceptOneRegularFile(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(home, "directory")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")

	runner := NewFileManagerUploadService(Service{HomeDir: func() (string, error) { return home, nil }})
	for _, paths := range [][]string{
		{},
		{filePath, filePath},
		{directory},
		{filepath.Join(home, "missing.txt")},
	} {
		result := runner.Upload(t.Context(), paths, nil)
		if result.Status != FileManagerUploadRejected {
			t.Errorf("paths %v: status = %q, want %q", paths, result.Status, FileManagerUploadRejected)
		}
		if result.Failure == nil {
			t.Errorf("paths %v: failure = nil, want validation failure", paths)
		}
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("endpoint requests = %d, want 0", got)
	}
}

func TestFileManagerUploadUsesDefaultsAndAlwaysCopies(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	var copied []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		fmt.Fprint(w, "https://files.example.test/result")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")

	runner := NewFileManagerUploadService(Service{
		HomeDir: func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(_ context.Context, value string) error {
			copied = append(copied, value)
			return nil
		}),
	})
	var progress []ManualUploadProgress
	result := runner.Upload(t.Context(), []string{filePath}, func(update ManualUploadProgress) {
		progress = append(progress, update)
	})
	if result.Status != FileManagerUploadSucceeded || result.Failure != nil {
		t.Fatalf("result = %#v, want success", result)
	}
	if len(copied) != 1 || copied[0] != "https://files.example.test/result" {
		t.Fatalf("copied = %#v, want automatic Final URL copy despite disabled preference", copied)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("actions = %#v, want no recovery action after successful copy", result.Actions)
	}
	if !slices.ContainsFunc(progress, func(update ManualUploadProgress) bool { return update.Phase == "preparing" }) ||
		!slices.ContainsFunc(progress, func(update ManualUploadProgress) bool { return update.Phase == "response" }) {
		t.Fatalf("progress = %#v, want preparation and response phases", progress)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{filePath, server.URL, "https://files.example.test/result"} {
		if strings.Contains(string(serialized), private) {
			t.Fatalf("serialized result contains private value %q: %s", private, serialized)
		}
	}

}

func TestFileManagerUploadRetryUsesFreshConfigurationExactlyOnce(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	var firstRequests atomic.Int32
	var secondRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			firstRequests.Add(1)
			http.Error(w, "private provider diagnostic", http.StatusBadGateway)
		case "/second":
			secondRequests.Add(1)
			fmt.Fprint(w, "https://files.example.test/retried")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "first", "second")

	runner := NewFileManagerUploadService(Service{
		HomeDir:   func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(context.Context, string) error { return nil }),
	})
	failed := runner.Upload(t.Context(), []string{filePath}, nil)
	if failed.Status != FileManagerUploadFailed || len(failed.Actions) != 1 || failed.Actions[0].Kind != FileManagerActionRetry {
		t.Fatalf("failed = %#v, want one Retry action", failed)
	}
	if strings.Contains(failed.Failure.Message, "private provider diagnostic") || strings.Contains(failed.Failure.Message, server.URL) {
		t.Fatalf("failure = %#v, leaks private diagnostic", failed.Failure)
	}

	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "second",
  "defaultShortener": "",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	retried, err := runner.Dispatch(t.Context(), failed.Actions[0].Token, nil)
	if err != nil {
		t.Fatalf("dispatch Retry: %v", err)
	}
	if retried.Upload == nil || retried.Upload.Status != FileManagerUploadSucceeded {
		t.Fatalf("retry result = %#v, want success", retried)
	}
	if got := firstRequests.Load(); got != 1 {
		t.Fatalf("first endpoint requests = %d, want 1", got)
	}
	if got := secondRequests.Load(); got != 1 {
		t.Fatalf("second endpoint requests = %d, want 1", got)
	}
	if len(retried.Upload.Actions) != 0 {
		t.Fatalf("retry actions = %#v, want no recovery action after successful copy", retried.Upload.Actions)
	}
}

func TestFileManagerUploadRejectsConcurrentOperation(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		close(started)
		<-release
		fmt.Fprint(w, "https://files.example.test/result")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")

	runner := NewFileManagerUploadService(Service{
		HomeDir:   func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(context.Context, string) error { return nil }),
	})
	firstResult := make(chan FileManagerUploadResult, 1)
	go func() {
		firstResult <- runner.Upload(t.Context(), []string{filePath}, nil)
	}()
	<-started

	second := runner.Upload(t.Context(), []string{filePath}, nil)
	if second.Status != FileManagerUploadRejected || second.Failure == nil || !strings.Contains(second.Failure.Message, "already active") {
		t.Fatalf("second = %#v, want active rejection", second)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("endpoint requests while first active = %d, want 1", got)
	}
	close(release)
	if first := <-firstResult; first.Status != FileManagerUploadSucceeded {
		t.Fatalf("first = %#v, want success", first)
	}
}

func TestFileManagerUploadActionExpires(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "https://files.example.test/result")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")

	runner := NewFileManagerUploadService(Service{
		HomeDir:   func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(context.Context, string) error { return errors.New("clipboard unavailable") }),
	})
	runner.actionLifetime = time.Millisecond
	result := runner.Upload(t.Context(), []string{filePath}, nil)
	if len(result.Actions) != 1 {
		t.Fatalf("actions = %#v, want one action", result.Actions)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := runner.Dispatch(t.Context(), result.Actions[0].Token, nil); err == nil {
		t.Fatal("expired action dispatch succeeded, want error")
	}
}

func TestFileManagerUploadActionSurvivesNewServiceAndIsConsumedOnce(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "https://files.example.test/result")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")
	var copied atomic.Int32
	service := Service{
		HomeDir: func() (string, error) { return home, nil },
		// Automatic copying fails; recovery must survive a new service instance.
		Clipboard: manualUploadClipboard(func(_ context.Context, value string) error {
			if copied.Load() == 0 {
				copied.Add(1)
				return errors.New("clipboard unavailable")
			}
			if value != "https://files.example.test/result" {
				t.Errorf("copied value = %q, want final URL", value)
			}
			copied.Add(1)
			return nil
		}),
	}
	first := NewFileManagerUploadService(service)
	result := first.Upload(t.Context(), []string{filePath}, nil)
	if len(result.Actions) != 1 || result.Actions[0].Kind != FileManagerActionCopyFinalURL {
		t.Fatalf("actions = %#v, want one Copy Final URL action", result.Actions)
	}

	second := NewFileManagerUploadService(service)
	type dispatchResult struct {
		action FileManagerActionResult
		err    error
	}
	results := make(chan dispatchResult, 2)
	for range 2 {
		go func() {
			action, err := second.Dispatch(t.Context(), result.Actions[0].Token, nil)
			results <- dispatchResult{action: action, err: err}
		}()
	}
	var succeeded int
	var failed int
	for range 2 {
		outcome := <-results
		if outcome.err == nil && outcome.action.Kind == FileManagerActionCopyFinalURL {
			succeeded++
		} else if outcome.err != nil {
			failed++
		}
	}
	if succeeded != 1 || failed != 1 || copied.Load() != 2 {
		t.Fatalf("dispatch outcomes = succeeded %d, failed %d, copy attempts %d; want 1, 1, 2", succeeded, failed, copied.Load())
	}
}

func TestFileManagerUploadRejectsFileChangedAfterPreflight(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, "https://files.example.test/should-not-upload")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")
	var homeCalls atomic.Int32
	service := Service{
		HomeDir: func() (string, error) {
			if homeCalls.Add(1) == 1 {
				if err := os.WriteFile(filePath, []byte("changed after selection"), 0o600); err != nil {
					return "", err
				}
			}
			return home, nil
		},
	}

	result := NewFileManagerUploadService(service).Upload(t.Context(), []string{filePath}, nil)
	if result.Status != FileManagerUploadFailed || result.Failure == nil || result.Failure.Stage != "validation" {
		t.Fatalf("result = %#v, want validation failure", result)
	}
	if len(result.Actions) != 1 || result.Actions[0].Kind != FileManagerActionRetry {
		t.Fatalf("actions = %#v, want one Retry action", result.Actions)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("endpoint requests = %d, want 0", got)
	}
}
func TestFileManagerUploadConfigurationFailureOffersDesktopAction(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := NewFileManagerUploadService(Service{HomeDir: func() (string, error) { return home, nil }})
	result := runner.Upload(t.Context(), []string{filePath}, nil)
	if result.Status != FileManagerUploadFailed || result.Failure == nil || result.Failure.Stage != "config" {
		t.Fatalf("result = %#v, want configuration failure", result)
	}
	if len(result.Actions) != 1 || result.Actions[0].Kind != FileManagerActionOpenDesktop {
		t.Fatalf("actions = %#v, want one Open Upit Desktop action", result.Actions)
	}
	if _, err := runner.Dispatch(t.Context(), result.Actions[0].Token, nil); err != nil {
		t.Fatalf("dispatch Open Upit Desktop: %v", err)
	}
	if _, err := runner.Dispatch(t.Context(), result.Actions[0].Token, nil); err == nil {
		t.Fatal("consumed Open Upit Desktop action succeeded")
	}
}

func TestFileManagerUploadCancellationDoesNotOfferAutomaticRetry(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	writeFileManagerConfiguration(t, home, server.URL, "default", "")
	runner := NewFileManagerUploadService(Service{HomeDir: func() (string, error) { return home, nil }})
	ctx, cancel := context.WithCancel(t.Context())
	resultCh := make(chan FileManagerUploadResult, 1)
	go func() { resultCh <- runner.Upload(ctx, []string{filePath}, nil) }()
	<-started
	cancel()
	result := <-resultCh
	if result.Status != FileManagerUploadCanceled || result.Failure == nil || !result.Failure.Canceled {
		t.Fatalf("result = %#v, want canceled result", result)
	}
	if len(result.Actions) != 0 {
		t.Fatalf("actions = %#v, want no automatic retry after cancellation", result.Actions)
	}
}

func TestFileManagerUploadClipboardWarningIsSanitizedAndRecoverable(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, "https://files.example.test/result")
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")
	var copyAttempts int
	runner := NewFileManagerUploadService(Service{
		HomeDir: func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(_ context.Context, value string) error {
			copyAttempts++
			if copyAttempts == 1 {
				return errors.New("clipboard contains private endpoint")
			}
			if value != "https://files.example.test/result" {
				t.Fatalf("recovery copied %q, want existing Final URL", value)
			}
			return nil
		}),
	})
	result := runner.Upload(t.Context(), []string{filePath}, nil)
	if result.Status != FileManagerUploadSucceeded || len(result.Warnings) != 1 || result.Warnings[0] != "Final URL was not copied to the clipboard" {
		t.Fatalf("result = %#v, want sanitized clipboard warning", result)
	}
	if len(result.Actions) != 1 || result.Actions[0].Kind != FileManagerActionCopyFinalURL {
		t.Fatalf("actions = %#v, want Copy Final URL recovery action", result.Actions)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "private endpoint") || strings.Contains(string(serialized), server.URL) {
		t.Fatalf("serialized result leaks private warning data: %s", serialized)
	}
	for _, name := range []string{"config.json", "custom-uploader.json"} {
		if err := os.Remove(filepath.Join(home, ".config", "upit", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.Dispatch(t.Context(), result.Actions[0].Token, nil); err != nil {
		t.Fatalf("recover existing result without Configuration Set: %v", err)
	}
	if requests.Load() != 1 || copyAttempts != 2 {
		t.Fatalf("requests = %d, copy attempts = %d; want 1 and 2", requests.Load(), copyAttempts)
	}
}

func TestFileManagerUploadShortenerFallbackIsSanitized(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			fmt.Fprint(w, "https://files.example.test/original")
		case "/shorten":
			http.Error(w, "secret shortener endpoint diagnostic", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeFileManagerConfiguration(t, home, server.URL, "default", "")
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "default",
  "defaultShortener": "short",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "short": {"request": {"method": "POST", "url": %q, "data": {"url": "{input}"}}, "response": {"url": {"type": "json", "path": "$.url"}}}
  }
}`, server.URL+"/shorten")), 0o600); err != nil {
		t.Fatal(err)
	}

	var copied string
	result := NewFileManagerUploadService(Service{
		HomeDir:   func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(_ context.Context, value string) error { copied = value; return nil }),
	}).Upload(t.Context(), []string{filePath}, nil)
	if result.Status != FileManagerUploadSucceeded || len(result.Warnings) != 1 || result.Warnings[0] != "URL shortening was unavailable; the Original URL was retained" {
		t.Fatalf("result = %#v, want sanitized Shortener fallback", result)
	}
	if copied != "https://files.example.test/original" {
		t.Fatalf("copied = %q, want Original URL retained as Final URL", copied)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(serialized), "secret shortener endpoint diagnostic") || strings.Contains(string(serialized), server.URL) {
		t.Fatalf("serialized result leaks private Shortener data: %s", serialized)
	}
}

func writeFileManagerConfiguration(t *testing.T, home, endpoint, defaultUploader, retryUploader string) {
	t.Helper()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(fmt.Sprintf(`{
  "version": 2,
  "defaultUploader": %q,
  "defaultShortener": "",
  "copyToClipboard": false
}`, defaultUploader)), 0o600); err != nil {
		t.Fatal(err)
	}
	uploaders := fmt.Sprintf(`{
  "version": 2,
  "uploaders": {
    "default": {"request": {"method": "POST", "url": %q, "body": "binary"}, "response": {"url": {"type": "body"}}}
`, endpoint+"/upload")
	if defaultUploader == "first" {
		uploaders = fmt.Sprintf(`{
  "version": 2,
  "uploaders": {
    "first": {"request": {"method": "POST", "url": %q, "body": "binary"}, "response": {"url": {"type": "body"}}},
    "second": {"request": {"method": "POST", "url": %q, "body": "binary"}, "response": {"url": {"type": "body"}}}
`, endpoint+"/first", endpoint+"/second")
	}
	uploaders += "  }\n}\n"
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(uploaders), 0o600); err != nil {
		t.Fatal(err)
	}
}
