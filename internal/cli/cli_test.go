package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HungNth/upit/internal/app"
	"github.com/HungNth/upit/internal/cli"
)

func TestUploadPrintsURLFromDefaultUploader(t *testing.T) {
	const fileContents = "upit streaming upload"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}

		file, header, err := r.FormFile("upload")
		if err != nil {
			t.Errorf("read multipart file: %v", err)
			http.Error(w, "missing upload", http.StatusBadRequest)
			return
		}
		defer file.Close()

		if header.Filename != "payload.txt" {
			t.Errorf("filename = %q, want payload.txt", header.Filename)
		}
		got, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("read uploaded file: %v", err)
		}
		if string(got) != fileContents {
			t.Errorf("uploaded contents = %q, want %q", got, fileContents)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"url":"https://files.example.test/payload.txt"}}`)
	}))
	defer server.Close()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	customUploaders := fmt.Sprintf(`{
  "version": 1,
  "uploaders": {
    "test": {
      "request": {
        "method": "POST",
        "url": %q,
        "body": "multipart",
        "fileField": "upload"
      },
      "response": {
        "url": {"type": "json", "path": "$.data.url"}
      }
    }
  }
}`, server.URL)
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(customUploaders), 0o600); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(t.TempDir(), "payload.txt")
	if err := os.WriteFile(filePath, []byte(fileContents), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "https://files.example.test/payload.txt\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestUploadUsesNamedUploaderRequestConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if got := r.URL.Query().Get("token"); got != "query-secret" {
			t.Errorf("query token = %q, want query-secret", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "header-secret" {
			t.Errorf("X-API-Key = %q, want header-secret", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart form: %v", err)
			http.Error(w, "invalid multipart", http.StatusBadRequest)
			return
		}
		if got := r.FormValue("folder"); got != "backups" {
			t.Errorf("folder = %q, want backups", got)
		}
		file, _, err := r.FormFile("payload")
		if err != nil {
			t.Errorf("read multipart file: %v", err)
			http.Error(w, "missing payload", http.StatusBadRequest)
			return
		}
		file.Close()

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"url":"https://files.example.test/named.bin"}`)
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
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	customUploaders := fmt.Sprintf(`{
  "version": 1,
  "uploaders": {
    "default": {
      "request": {
        "method": "POST",
        "url": "http://127.0.0.1:1",
        "body": "multipart",
        "fileField": "file"
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    },
    "selected": {
      "request": {
        "method": "PUT",
        "url": %q,
        "headers": {"X-API-Key": "header-secret"},
        "query": {"token": "query-secret"},
        "body": "multipart",
        "fileField": "payload",
        "fields": {"folder": "backups"}
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, server.URL)
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(customUploaders), 0o600); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(t.TempDir(), "named.bin")
	if err := os.WriteFile(filePath, []byte("named uploader"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	for name, args := range map[string][]string{
		"before file": {"upload", "--uploader", "selected", filePath},
		"after file":  {"upload", filePath, "--uploader", "selected"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runner.Run(context.Background(), args, &stdout, &stderr)

			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
			}
			if got, want := stdout.String(), "https://files.example.test/named.bin\n"; got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
		})
	}
}

func TestUploadShortensWithExplicitNamedShortener(t *testing.T) {
	const originalURL = "https://files.example.test/original.bin"
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()

	shortenerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if got := r.URL.Query().Get("workspace"); got != "docs" {
			t.Errorf("workspace query = %q, want docs", got)
		}
		if got := r.Header.Get("X-API-Key"); got != "shortener-secret" {
			t.Errorf("X-API-Key = %q, want shortener-secret", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body struct {
			Target struct {
				URL string `json:"url"`
			} `json:"target"`
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode shortener request: %v", err)
		}
		if body.Target.URL != originalURL || !body.Enabled {
			t.Errorf("shortener body = %+v, want original URL and enabled=true", body)
		}
		fmt.Fprint(w, `{"data":{"shortUrl":"https://sho.rt/abc123"}}`)
	}))
	defer shortenerServer.Close()

	home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
	shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "docs": {
      "request": {
        "method": "PATCH",
        "url": %q,
        "headers": {"X-API-Key": "shortener-secret"},
        "query": {"workspace": "docs"},
        "data": {"target": {"url": "{input}"}, "enabled": true}
      },
      "response": {
        "url": {"type": "json", "path": "$.data.shortUrl"}
      }
    }
  }
}`, shortenerServer.URL)
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, args := range map[string][]string{
		"before file": {"upload", "--shortener", "docs", filePath},
		"after file":  {"upload", filePath, "--shortener", "docs"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
				context.Background(), args, &stdout, &stderr,
			)

			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
			}
			if got, want := stdout.String(), "https://sho.rt/abc123\n"; got != want {
				t.Errorf("stdout = %q, want %q", got, want)
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestDefaultShortenerCanBeOverriddenOrDisabled(t *testing.T) {
	const originalURL = "https://files.example.test/original"
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()
	shortenerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/default":
			fmt.Fprint(w, `{"url":"https://sho.rt/default"}`)
		case "/override":
			fmt.Fprint(w, `{"url":"https://sho.rt/override"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer shortenerServer.Close()

	setup := func(t *testing.T) (home, filePath string) {
		t.Helper()
		home, filePath = writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
		global := `{
  "version": 2,
  "defaultUploader": "test",
  "defaultShortener": "default",
  "copyToClipboard": false
}`
		if err := os.WriteFile(filepath.Join(home, ".config", "upit", "config.json"), []byte(global), 0o600); err != nil {
			t.Fatal(err)
		}
		shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "default": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    },
    "override": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, shortenerServer.URL+"/default", shortenerServer.URL+"/override")
		if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
			t.Fatal(err)
		}
		return home, filePath
	}
	run := func(t *testing.T, home string, args []string, want string) {
		t.Helper()
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(context.Background(), args, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if got := strings.TrimSpace(stdout.String()); got != want {
			t.Errorf("stdout = %q, want %q", got, want)
		}
	}

	t.Run("configured default", func(t *testing.T) {
		home, filePath := setup(t)
		run(t, home, []string{"upload", filePath}, "https://sho.rt/default")
	})
	t.Run("explicit override", func(t *testing.T) {
		home, filePath := setup(t)
		run(t, home, []string{"upload", "--shortener", "override", filePath}, "https://sho.rt/override")
	})
	t.Run("disabled without loading Shortener file", func(t *testing.T) {
		home, filePath := setup(t)
		if err := os.Remove(filepath.Join(home, ".config", "upit", "custom-shortener.json")); err != nil {
			t.Fatal(err)
		}
		run(t, home, []string{"upload", filePath, "--no-shorten"}, originalURL)
	})
}

func TestShortenerSelectionFlagsConflict(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{}).Run(
		context.Background(), []string{"upload", "file.bin", "--shortener", "docs", "--no-shorten"}, &stdout, &stderr,
	)

	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if !strings.Contains(stderr.String(), "--shortener and --no-shorten cannot be used together") {
		t.Errorf("stderr = %q, want conflicting Shortener flags", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestShortenerFailureFallsBackWithoutLeakingSecrets(t *testing.T) {
	const originalURL = "https://files.example.test/original"
	var uploadCalls atomic.Int32
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploadCalls.Add(1)
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()
	var shortenerCalls atomic.Int32
	shortenerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		shortenerCalls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"header-secret query-secret body-secret endpoint-secret"}`)
	}))
	defer shortenerServer.Close()

	home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
	shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "limited": {
      "request": {
        "method": "POST",
        "url": %q,
        "headers": {"Authorization": "header-secret"},
        "query": {"token": "query-secret"},
        "data": {"target": "{input}", "note": "body-secret"}
      },
      "response": {
        "url": {"type": "json", "path": "$.url"},
        "error": {"type": "json", "path": "$.error"}
      }
    }
  }
}`, shortenerServer.URL+"/links?key=endpoint-secret")
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
		context.Background(), []string{"upload", filePath, "--shortener", "limited", "--json"}, &stdout, &stderr,
	)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	wantJSON := `{"success":true,"originalUrl":"https://files.example.test/original","finalUrl":"https://files.example.test/original"}` + "\n"
	if got := stdout.String(); got != wantJSON {
		t.Errorf("stdout = %q, want %q", got, wantJSON)
	}
	if !strings.HasPrefix(stderr.String(), "Warning: shorten URL: ") {
		t.Errorf("stderr = %q, want Shortener warning", stderr.String())
	}
	for _, secret := range []string{"header-secret", "query-secret", "body-secret", "endpoint-secret"} {
		if strings.Contains(stderr.String(), secret) {
			t.Errorf("stderr leaked %q: %q", secret, stderr.String())
		}
	}
	if got := uploadCalls.Load(); got != 1 {
		t.Errorf("upload calls = %d, want 1", got)
	}
	if got := shortenerCalls.Load(); got != 1 {
		t.Errorf("Shortener calls = %d, want 1", got)
	}
}

func TestShortenerDeadlineFallsBackToOriginalURL(t *testing.T) {
	const originalURL = "https://files.example.test/original"
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()
	canceled := make(chan struct{})
	shortenerServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		r.Body.Close()
		<-r.Context().Done()
		close(canceled)
	}))
	defer shortenerServer.Close()

	home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
	shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "slow": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, shortenerServer.URL)
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
		context.Background(), []string{"upload", filePath, "--shortener", "slow", "--timeout", "100ms"}, &stdout, &stderr,
	)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != originalURL {
		t.Errorf("stdout = %q, want Original URL", got)
	}
	if !strings.HasPrefix(stderr.String(), "Warning: shorten URL: ") {
		t.Errorf("stderr = %q, want Shortener deadline warning", stderr.String())
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("Shortener endpoint did not observe deadline cancellation")
	}
}

func TestShortenerResponseFailuresFallBack(t *testing.T) {
	const originalURL = "https://files.example.test/original"
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()

	tests := map[string]struct {
		responsePath string
		handler      http.HandlerFunc
		wantWarning  string
	}{
		"oversized response": {
			responsePath: "$.url",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, strings.Repeat("x", (1<<20)+1))
			},
			wantWarning: "exceeds 1 MiB",
		},
		"invalid JSON": {
			responsePath: "$.url",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{`)
			},
			wantWarning: "decode shortener response",
		},
		"multiple extracted URLs": {
			responsePath: "$.urls[*]",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{"urls":["https://sho.rt/a","https://sho.rt/b"]}`)
			},
			wantWarning: "selected 2 values",
		},
		"invalid Final URL": {
			responsePath: "$.url",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, `{"url":"ftp://sho.rt/a"}`)
			},
			wantWarning: "absolute HTTP(S) URL",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			shortenerServer := httptest.NewServer(test.handler)
			defer shortenerServer.Close()
			home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
			shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "broken": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": %q}}
    }
  }
}`, shortenerServer.URL, test.responsePath)
			if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
				t.Fatal(err)
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
				context.Background(), []string{"upload", filePath, "--shortener", "broken"}, &stdout, &stderr,
			)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
			}
			if got := strings.TrimSpace(stdout.String()); got != originalURL {
				t.Errorf("stdout = %q, want Original URL", got)
			}
			if !strings.Contains(stderr.String(), test.wantWarning) {
				t.Errorf("stderr = %q, want %q", stderr.String(), test.wantWarning)
			}
		})
	}
}

func TestShortenerConfigurationIsValidatedBeforeUpload(t *testing.T) {
	var uploadCalled atomic.Bool
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploadCalled.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/original"}`)
	}))
	defer uploadServer.Close()

	home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
	shorteners := `{
  "version": 1,
  "shorteners": {
    "selected": {
      "request": {
        "method": "POST",
        "url": "https://short.example.test/links",
        "data": {"target": "{input}"}
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    },
    "broken": {
      "request": {
        "method": "POST",
        "url": "https://short.example.test/links",
        "data": {"target": "literal"}
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
		context.Background(), []string{"upload", filePath, "--shortener", "selected"}, &stdout, &stderr,
	)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if uploadCalled.Load() {
		t.Fatal("upload endpoint was called before all Shorteners were valid")
	}
	if !strings.Contains(stderr.String(), `shortener "broken": data must contain exactly one`) {
		t.Errorf("stderr = %q, want invalid unused Shortener error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestUploadRejectsUnsafeShortenerFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}

	var uploadCalled atomic.Bool
	uploadServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		uploadCalled.Store(true)
	}))
	defer uploadServer.Close()

	home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
	path := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	if err := os.WriteFile(path, []byte(`{
  "version": 1,
  "shorteners": {
    "selected": {
      "request": {
        "method": "POST",
        "url": "https://short.example.test/links",
        "data": {"target": "{input}"}
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
		context.Background(), []string{"upload", filePath, "--shortener", "selected"}, &stdout, &stderr,
	)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if uploadCalled.Load() {
		t.Fatal("upload endpoint was called with unsafe Shortener permissions")
	}
	if !strings.Contains(stderr.String(), "chmod 600") {
		t.Errorf("stderr = %q, want chmod guidance", stderr.String())
	}
}

func TestExampleConfigurationPassesStrictLoading(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for source, destination := range map[string]string{
		filepath.Join("..", "..", "examples", "config.example.json"):          filepath.Join(configDir, "config.json"),
		filepath.Join("..", "..", "examples", "custom-uploader.example.json"): filepath.Join(configDir, "custom-uploader.json"),
	} {
		data, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read example %s: %v", source, err)
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filepath.Join(home, "missing.bin")}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Stage: validation") {
		t.Errorf("stderr = %q, want validation-stage file error", stderr.String())
	}
}

func TestUploadRejectsGlobalConfigV1BeforeNetwork(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/legacy"}`)
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "config.json"), []byte(`{
  "version": 1,
  "defaultUploader": "test",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := (cli.Runner{HomeDir: func() (string, error) { return home, nil }}).Run(
		context.Background(), []string{"upload", filePath}, &stdout, &stderr,
	)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called for global config version 1")
	}
	if !strings.Contains(stderr.String(), "config version = 1, want 2") {
		t.Errorf("stderr = %q, want version-2 guidance", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestUploadRejectsUnknownConfigurationFields(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "personal",
  "copyToClipboard": false,
  "urlShortener": {"enabled": true}
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", "missing.bin"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown field "urlShortener"`) {
		t.Errorf("stderr = %q, want unknown-field error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestUploadRejectsUnsafeUploaderFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "personal",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	uploaderPath := filepath.Join(configDir, "custom-uploader.json")
	if err := os.WriteFile(uploaderPath, []byte(`{
  "version": 1,
  "uploaders": {}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(uploaderPath, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", "missing.bin"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "chmod 600") {
		t.Errorf("stderr = %q, want chmod guidance", stderr.String())
	}
}

func TestUploadDoesNotFollowRedirects(t *testing.T) {
	var targetCalled atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalled.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/redirected"}`)
	}))
	defer target.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()

	home, filePath := writeBasicUploadFixture(t, redirect.URL, "$.url", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if targetCalled.Load() {
		t.Error("redirect target was called")
	}
	if !strings.Contains(stderr.String(), "HTTP: 302") {
		t.Errorf("stderr = %q, want HTTP 302", stderr.String())
	}
}

func TestUploadSanitizesExtractedEndpointError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"token Bearer secret-token and query-secret rejected"}`)
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "$.error")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if strings.Contains(stderr.String(), "secret-token") || strings.Contains(stderr.String(), "query-secret") {
		t.Errorf("stderr leaked configured values: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "token [REDACTED] and [REDACTED] rejected") {
		t.Errorf("stderr = %q, want sanitized endpoint error", stderr.String())
	}
	if !strings.Contains(stderr.String(), "HTTP: 401") {
		t.Errorf("stderr = %q, want HTTP 401", stderr.String())
	}
	if calls.Load() != 1 {
		t.Errorf("endpoint calls = %d, want 1", calls.Load())
	}
}

func TestNetworkErrorsDoNotExposeRequestQuery(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "http://127.0.0.1:1/upload?existing=url-secret", "$.url", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	for _, leaked := range []string{"url-secret", "query-secret", "?existing="} {
		if strings.Contains(stderr.String(), leaked) {
			t.Errorf("stderr leaked %q: %q", leaked, stderr.String())
		}
	}
	if !strings.Contains(stderr.String(), "Stage: network") {
		t.Errorf("stderr = %q, want network stage", stderr.String())
	}
}

func TestUploadRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(make([]byte, (1<<20)+1))
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Stage: response") || !strings.Contains(stderr.String(), "exceeds 1 MiB") {
		t.Errorf("stderr = %q, want bounded response error", stderr.String())
	}
}

func TestJSONPathExtractionRequiresExactlyOneString(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		path       string
		exitCode   int
		wantOutput string
		wantError  string
	}{
		{
			name:       "filter selects one array element",
			response:   `{"files":[{"primary":false,"url":"https://files.example.test/old"},{"primary":true,"url":"https://files.example.test/current"}]}`,
			path:       `$.files[?@.primary == true].url`,
			exitCode:   0,
			wantOutput: "https://files.example.test/current\n",
		},
		{
			name:      "wildcard selects multiple values",
			response:  `{"files":[{"url":"https://files.example.test/one"},{"url":"https://files.example.test/two"}]}`,
			path:      `$.files[*].url`,
			exitCode:  1,
			wantError: "selected 2 values, want 1",
		},
		{
			name:      "null is not coerced",
			response:  `{"url":null}`,
			path:      `$.url`,
			exitCode:  1,
			wantError: "must select a JSON string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, test.response)
			}))
			defer server.Close()

			home, filePath := writeBasicUploadFixture(t, server.URL, test.path, "")
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
			exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)

			if exitCode != test.exitCode {
				t.Fatalf("exit code = %d, want %d; stderr = %q", exitCode, test.exitCode, stderr.String())
			}
			if stdout.String() != test.wantOutput {
				t.Errorf("stdout = %q, want %q", stdout.String(), test.wantOutput)
			}
			if test.wantError != "" && !strings.Contains(stderr.String(), test.wantError) {
				t.Errorf("stderr = %q, want %q", stderr.String(), test.wantError)
			}
		})
	}
}

func TestJSONOutputContract(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, `{"url":"https://files.example.test/json-success"}`)
		}))
		defer server.Close()

		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--json"}, &stdout, &stderr)

		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		var result struct {
			Success     bool   `json:"success"`
			OriginalURL string `json:"originalUrl"`
			FinalURL    string `json:"finalUrl"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("decode stdout JSON: %v; stdout = %q", err, stdout.String())
		}
		if !result.Success || result.OriginalURL != "https://files.example.test/json-success" || result.FinalURL != result.OriginalURL {
			t.Errorf("result = %+v", result)
		}
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q, want empty", stderr.String())
		}
	})

	t.Run("failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid token"}`)
		}))
		defer server.Close()

		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "$.error")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
		exitCode := runner.Run(context.Background(), []string{"upload", "--json", filePath}, &stdout, &stderr)

		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("stdout = %q, want empty", stdout.String())
		}
		var failure struct {
			Success    bool   `json:"success"`
			Stage      string `json:"stage"`
			Message    string `json:"message"`
			StatusCode int    `json:"statusCode"`
		}
		if err := json.Unmarshal(stderr.Bytes(), &failure); err != nil {
			t.Fatalf("decode stderr JSON: %v; stderr = %q", err, stderr.String())
		}
		if failure.Success || failure.Stage != "response" || failure.Message != "invalid token" || failure.StatusCode != http.StatusUnauthorized {
			t.Errorf("failure = %+v", failure)
		}
	})
}

func TestUsageErrorsExitTwo(t *testing.T) {
	runner := cli.Runner{}
	for name, args := range map[string][]string{
		"unknown command":    {"unknown"},
		"missing file":       {"upload", "--json"},
		"extra file":         {"upload", "one", "two"},
		"unknown flag":       {"upload", "--unknown", "file"},
		"negative timeout":   {"upload", "--timeout", "-1s", "file"},
		"clipboard conflict": {"upload", "--clipboard", "--no-clipboard", "file"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runner.Run(context.Background(), args, &stdout, &stderr)
			if exitCode != 2 {
				t.Errorf("exit code = %d, want 2", exitCode)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage: upit upload") {
				t.Errorf("stderr = %q, want usage", stderr.String())
			}
		})
	}
}

func TestHelpDescribesUploadFlags(t *testing.T) {
	runner := cli.Runner{}
	for name, args := range map[string][]string{
		"root":   {"--help"},
		"upload": {"upload", "--help"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exitCode := runner.Run(context.Background(), args, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), "upit upload") || !strings.Contains(stdout.String(), "--timeout") || !strings.Contains(stdout.String(), "--no-clipboard") {
				t.Errorf("stdout = %q, want upload help", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestUploadTimeoutCancelsRequest(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		buffer := make([]byte, 64<<10)
		for {
			if _, err := r.Body.Read(buffer); err != nil {
				close(canceled)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	if err := os.Truncate(filePath, 16<<20); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--timeout", "25ms"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "deadline exceeded") {
		t.Errorf("stderr = %q, want deadline error", stderr.String())
	}
	select {
	case <-started:
	default:
		t.Fatal("server never received request")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("server did not observe cancellation")
	}
}

func TestParentCancellationExits130(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		buffer := make([]byte, 64<<10)
		for {
			if _, err := r.Body.Read(buffer); err != nil {
				close(canceled)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	if err := os.Truncate(filePath, 16<<20); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- runner.Run(ctx, []string{"upload", filePath}, &stdout, &stderr)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("server never received request")
	}
	cancel()
	select {
	case exitCode := <-exitCodes:
		if exitCode != 130 {
			t.Fatalf("exit code = %d, want 130; stderr = %q", exitCode, stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("command did not exit after cancellation")
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("server did not observe canceled upload stream")
	}
}

func TestShortenerCancellationExits130WithStageShortener(t *testing.T) {
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"url":"https://files.example.test/original"}`)
	}))
	defer uploadServer.Close()

	for _, asJSON := range []bool{false, true} {
		name := "plain"
		if asJSON {
			name = "json"
		}
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			canceled := make(chan struct{})
			shortenerServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				r.Body.Close()
				close(started)
				<-r.Context().Done()
				close(canceled)
			}))
			defer shortenerServer.Close()

			home, filePath := writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
			shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "slow": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, shortenerServer.URL)
			if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
			args := []string{"upload", filePath, "--shortener", "slow"}
			if asJSON {
				args = append(args, "--json")
			}
			exitCodes := make(chan int, 1)
			go func() {
				exitCodes <- runner.Run(ctx, args, &stdout, &stderr)
			}()

			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("Shortener endpoint never received request")
			}
			cancel()
			select {
			case exitCode := <-exitCodes:
				if exitCode != 130 {
					t.Fatalf("exit code = %d, want 130; stderr = %q", exitCode, stderr.String())
				}
			case <-time.After(time.Second):
				t.Fatal("command did not exit after cancellation")
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if asJSON {
				if !strings.Contains(stderr.String(), `"stage":"shortener"`) {
					t.Errorf("stderr = %q, want stage shortener", stderr.String())
				}
			} else {
				if !strings.Contains(stderr.String(), "Stage: shortener") {
					t.Errorf("stderr = %q, want Stage: shortener", stderr.String())
				}
			}
			select {
			case <-canceled:
			case <-time.After(time.Second):
				t.Fatal("Shortener endpoint did not observe cancellation")
			}
		})
	}
}

func TestClipboardReceivesResolvedFinalURLAndChainsWarnings(t *testing.T) {
	const originalURL = "https://files.example.test/original"
	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"url":%q}`, originalURL)
	}))
	defer uploadServer.Close()
	shortenerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, `{"url":"https://sho.rt/success"}`)
	}))
	defer shortenerServer.Close()

	setup := func(t *testing.T) (home, filePath string) {
		t.Helper()
		home, filePath = writeBasicUploadFixture(t, uploadServer.URL, "$.url", "")
		shorteners := fmt.Sprintf(`{
  "version": 1,
  "shorteners": {
    "ok": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    },
    "bad": {
      "request": {"method": "POST", "url": %q, "data": {"url": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, shortenerServer.URL+"/ok", shortenerServer.URL+"/fail")
		if err := os.WriteFile(filepath.Join(home, ".config", "upit", "custom-shortener.json"), []byte(shorteners), 0o600); err != nil {
			t.Fatal(err)
		}
		return home, filePath
	}

	t.Run("success copies shortened URL", func(t *testing.T) {
		home, filePath := setup(t)
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--shortener", "ok", "--clipboard"}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if len(clipboard.values) != 1 || clipboard.values[0] != "https://sho.rt/success" {
			t.Errorf("clipboard values = %v, want [https://sho.rt/success]", clipboard.values)
		}
	})

	t.Run("fallback copies Original URL", func(t *testing.T) {
		home, filePath := setup(t)
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--shortener", "bad", "--clipboard"}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if len(clipboard.values) != 1 || clipboard.values[0] != originalURL {
			t.Errorf("clipboard values = %v, want [%s]", clipboard.values, originalURL)
		}
		if !strings.Contains(stderr.String(), "Warning: shorten URL: ") {
			t.Errorf("stderr = %q, want Shortener warning", stderr.String())
		}
	})

	t.Run("both warnings emitted when both fail", func(t *testing.T) {
		home, filePath := setup(t)
		clipboard := &recordingClipboard{err: errors.New("clipboard broke")}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--shortener", "bad", "--clipboard"}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if got, want := strings.TrimSpace(stdout.String()), originalURL; got != want {
			t.Errorf("stdout = %q, want %q", got, want)
		}
		if !strings.Contains(stderr.String(), "Warning: shorten URL: ") || !strings.Contains(stderr.String(), "Warning: copy to clipboard: ") {
			t.Errorf("stderr = %q, want both shorten and clipboard warnings", stderr.String())
		}
		shortenIdx := strings.Index(stderr.String(), "Warning: shorten URL: ")
		clipboardIdx := strings.Index(stderr.String(), "Warning: copy to clipboard: ")
		if shortenIdx > clipboardIdx {
			t.Errorf("shortener warning should precede clipboard warning, got: %q", stderr.String())
		}
	})
}

func TestEarlyResponseStopsBlockedMultipartProducer(t *testing.T) {
	release := make(chan struct{})
	response := `{"url":"https://files.example.test/too-early"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprint(len(response)))
		fmt.Fprint(w, response)
		w.(http.Flusher).Flush()
		<-release
	}))
	defer server.Close()
	defer close(release)

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	if err := os.Truncate(filePath, 16<<20); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }}
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	}()

	select {
	case exitCode := <-exitCodes:
		if exitCode != 1 {
			t.Fatalf("exit code = %d, want 1; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
		}
		if !strings.Contains(stderr.String(), "responded before request body completed") {
			t.Errorf("stderr = %q, want early-response upload error", stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("command blocked waiting for multipart producer")
	}
}

func TestClipboardIsOptionalAndNonFatal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"url":"https://files.example.test/clipboard"}`)
	}))
	defer server.Close()

	t.Run("disabled by default", func(t *testing.T) {
		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
		if exitCode != 0 || len(clipboard.values) != 0 {
			t.Errorf("exit = %d, copied = %v, stderr = %q", exitCode, clipboard.values, stderr.String())
		}
	})

	t.Run("enabled by flag", func(t *testing.T) {
		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--clipboard"}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		if len(clipboard.values) != 1 || clipboard.values[0] != "https://files.example.test/clipboard" {
			t.Errorf("copied values = %v", clipboard.values)
		}
	})

	t.Run("enabled by config", func(t *testing.T) {
		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		setClipboardDefault(t, home, true)
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
		if exitCode != 0 || len(clipboard.values) != 1 {
			t.Errorf("exit = %d, copied = %v, stderr = %q", exitCode, clipboard.values, stderr.String())
		}
	})

	t.Run("disabled flag overrides config", func(t *testing.T) {
		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		setClipboardDefault(t, home, true)
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", "--no-clipboard", filePath}, &stdout, &stderr)
		if exitCode != 0 || len(clipboard.values) != 0 {
			t.Errorf("exit = %d, copied = %v, stderr = %q", exitCode, clipboard.values, stderr.String())
		}
	})

	t.Run("failure preserves JSON success", func(t *testing.T) {
		home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
		clipboard := &recordingClipboard{err: errors.New("clipboard unavailable")}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--clipboard", "--json"}, &stdout, &stderr)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
		}
		var result struct {
			Success bool `json:"success"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.Success {
			t.Errorf("stdout = %q, want JSON success", stdout.String())
		}
		if !strings.Contains(stderr.String(), "clipboard unavailable") {
			t.Errorf("stderr = %q, want clipboard warning", stderr.String())
		}
	})

	t.Run("upload failure is not copied", func(t *testing.T) {
		failureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"error":"failed"}`)
		}))
		defer failureServer.Close()
		home, filePath := writeBasicUploadFixture(t, failureServer.URL, "$.url", "$.error")
		clipboard := &recordingClipboard{}
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		runner := cli.Runner{HomeDir: func() (string, error) { return home, nil }, Clipboard: clipboard}
		exitCode := runner.Run(context.Background(), []string{"upload", filePath, "--clipboard"}, &stdout, &stderr)
		if exitCode != 1 || len(clipboard.values) != 0 {
			t.Errorf("exit = %d, copied = %v, stderr = %q", exitCode, clipboard.values, stderr.String())
		}
	})
}

type recordingClipboard struct {
	values []string
	err    error
}

func (c *recordingClipboard) Copy(_ context.Context, value string) error {
	c.values = append(c.values, value)
	return c.err
}

func setClipboardDefault(t *testing.T, home string, enabled bool) {
	t.Helper()
	config := fmt.Sprintf(`{
  "version": 2,
  "defaultUploader": "test",
  "copyToClipboard": %t
}`, enabled)
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "config.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
}

var _ app.Clipboard = (*recordingClipboard)(nil)

func writeBasicUploadFixture(t *testing.T, endpoint, urlPath, errorPath string) (home, filePath string) {
	t.Helper()
	home = t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	errorExtractor := ""
	if errorPath != "" {
		errorExtractor = fmt.Sprintf(`,
        "error": {"type": "json", "path": %q}`, errorPath)
	}
	uploaders := fmt.Sprintf(`{
  "version": 1,
  "uploaders": {
    "test": {
      "request": {
        "method": "POST",
        "url": %q,
        "headers": {"Authorization": "Bearer secret-token"},
        "query": {"token": "query-secret"},
        "body": "multipart",
        "fileField": "file"
      },
      "response": {
        "url": {"type": "json", "path": %q}%s
      }
    }
  }
}`, endpoint, urlPath, errorExtractor)
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(uploaders), 0o600); err != nil {
		t.Fatal(err)
	}
	filePath = filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(filePath, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	return home, filePath
}
