package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type manualUploadClipboard func(context.Context, string) error

func (copyFn manualUploadClipboard) Copy(ctx context.Context, value string) error {
	return copyFn(ctx, value)
}

func TestManualUploadUsesDefaultsAndExplicitOverrides(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/default":
			fmt.Fprint(w, "https://files.example.test/default")
		case "/alternate":
			fmt.Fprint(w, "https://files.example.test/alternate")
		case "/shorten":
			fmt.Fprint(w, `{"url":"https://short.example.test/default"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "default",
  "defaultShortener": "short",
  "copyToClipboard": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(fmt.Sprintf(`{
  "version": 2,
  "uploaders": {
    "default": {"request": {"method": "POST", "url": %q, "body": "binary"}, "response": {"url": {"type": "body"}}},
    "alternate": {"request": {"method": "POST", "url": %q, "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`, server.URL+"/default", server.URL+"/alternate")), 0o600); err != nil {
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
	filePath := filepath.Join(home, "payload.txt")
	if err := os.WriteFile(filePath, []byte("manual upload"), 0o600); err != nil {
		t.Fatal(err)
	}

	var copied []string
	service := Service{
		HomeDir: func() (string, error) { return home, nil },
		Clipboard: manualUploadClipboard(func(_ context.Context, value string) error {
			copied = append(copied, value)
			return nil
		}),
	}

	defaultResult := service.ManualUpload(t.Context(), ManualUploadOptions{FilePath: filePath})
	if !defaultResult.Success || defaultResult.Failure != nil {
		t.Fatalf("default result = %#v, want successful result", defaultResult)
	}
	if defaultResult.OriginalURL != "https://files.example.test/default" || defaultResult.FinalURL != "https://short.example.test/default" {
		t.Fatalf("default URLs = %q, %q", defaultResult.OriginalURL, defaultResult.FinalURL)
	}
	if len(copied) != 1 || copied[0] != defaultResult.FinalURL {
		t.Fatalf("copied = %#v, want final URL", copied)
	}

	overrideResult := service.ManualUpload(t.Context(), ManualUploadOptions{
		FilePath:          filePath,
		Uploader:          "alternate",
		DisableShortening: true,
		Clipboard:         "disabled",
	})
	if !overrideResult.Success || overrideResult.Failure != nil {
		t.Fatalf("override result = %#v, want successful result", overrideResult)
	}
	if overrideResult.OriginalURL != "https://files.example.test/alternate" || overrideResult.FinalURL != overrideResult.OriginalURL {
		t.Fatalf("override URLs = %q, %q", overrideResult.OriginalURL, overrideResult.FinalURL)
	}
	if len(copied) != 1 {
		t.Fatalf("copied = %#v, want no clipboard override copy", copied)
	}

	invalidTimeout := service.ManualUpload(t.Context(), ManualUploadOptions{FilePath: filePath, Timeout: "tomorrow"})
	if invalidTimeout.Success || invalidTimeout.Failure == nil || invalidTimeout.Failure.Stage != "validation" || !strings.Contains(invalidTimeout.Failure.Message, "timeout") {
		t.Fatalf("invalid timeout result = %#v, want validation failure", invalidTimeout)
	}
}

func TestManualUploadRedactsSelectedFilePathFromFailure(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "private-upload.txt")

	result := (Service{}).ManualUpload(t.Context(), ManualUploadOptions{FilePath: filePath})

	if result.Success || result.Failure == nil {
		t.Fatalf("result = %#v, want validation failure", result)
	}
	if strings.Contains(result.Failure.Message, filePath) {
		t.Fatalf("failure message = %q, must not expose selected file path %q", result.Failure.Message, filePath)
	}
}

func TestManualUploadReportsTransferProgressAndCancels(t *testing.T) {
	requestStarted := make(chan struct{})
	handlerRelease := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.CopyN(io.Discard, r.Body, 32<<10)
		_ = r.Body.Close()
		close(requestStarted)
		select {
		case <-r.Context().Done():
		case <-handlerRelease:
		}
	}))
	defer func() {
		close(handlerRelease)
		server.Close()
	}()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"base","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(fmt.Sprintf(`{"version":2,"uploaders":{"base":{"request":{"method":"POST","url":%q,"body":"binary"},"response":{"url":{"type":"body"}}}}}`, server.URL)), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(home, "payload.bin")
	contents := []byte(strings.Repeat("x", 1<<20))
	if err := os.WriteFile(filePath, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	progressObserved := make(chan struct{})
	var once sync.Once
	var progress []ManualUploadProgress
	result := make(chan ManualUploadResult, 1)
	go func() {
		result <- (Service{HomeDir: func() (string, error) { return home, nil }}).ManualUploadWithProgress(ctx, ManualUploadOptions{FilePath: filePath, Clipboard: "disabled"}, func(update ManualUploadProgress) {
			progress = append(progress, update)
			if update.Phase == "uploading" && update.Processed > 0 {
				once.Do(func() { close(progressObserved) })
			}
		})
	}()

	select {
	case <-progressObserved:
	case <-time.After(time.Second):
		t.Fatal("Manual Upload did not report transfer progress")
	}
	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("upload endpoint did not receive request")
	}
	cancel()
	select {
	case outcome := <-result:
		if outcome.Success || outcome.Failure == nil || !outcome.Failure.Canceled {
			t.Fatalf("canceled result = %#v, want canceled Manual Upload failure", outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("Manual Upload did not return after cancellation")
	}

	if len(progress) < 2 || progress[0].Phase != "preparing" {
		t.Fatalf("progress = %#v, want preparation followed by transfer", progress)
	}
	var previous int64
	for _, update := range progress {
		if update.Phase != "uploading" {
			continue
		}
		if update.Total != int64(len(contents)) || update.Processed < previous {
			t.Fatalf("progress update = %#v, want monotonic byte progress against %d", update, len(contents))
		}
		previous = update.Processed
	}
}
