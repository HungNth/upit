package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/HungNth/upit/internal/cli"
)

func TestBinaryUploadExtractsURLFromHeader(t *testing.T) {
	const fileContents = "binary\x00payload\xff"
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %q, want PUT", r.Method)
		}
		if got := r.Header.Get("X-Request"); got != "ordinary" {
			t.Errorf("X-Request = %q, want ordinary", got)
		}
		if got := r.URL.Query().Get("token"); got != "query-value" {
			t.Errorf("token query = %q, want query-value", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/octet-stream" {
			t.Errorf("Content-Type = %q, want application/octet-stream", got)
		}
		if got := r.Header.Get("Content-Disposition"); got != "" {
			t.Errorf("Content-Disposition = %q, want empty", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read binary body: %v", err)
		}
		received <- body
		w.Header().Set("Location", "https://files.example.test/binary")
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method":  "PUT",
		"url":     server.URL,
		"headers": map[string]string{"X-Request": "ordinary"},
		"query":   map[string]string{"token": "query-value"},
		"body":    "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte(fileContents))

	var stdout, stderr bytes.Buffer
	exitCode := (cliRunner(home)).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	gotReceived := <-received
	if string(gotReceived) != fileContents {
		t.Errorf("received body = %q, want %q", gotReceived, fileContents)
	}
	if got, want := stdout.String(), "https://files.example.test/binary\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestBinaryUploadUsesConfiguredContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "image/png" {
			t.Errorf("Content-Type = %q, want image/png", got)
		}
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Location", "https://files.example.test/custom-type")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method":  "POST",
		"url":     server.URL,
		"headers": map[string]string{"Content-Type": "image/png"},
		"body":    "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "https://files.example.test/custom-type\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestUploaderRejectsCaseVariantDuplicateHeaders(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/invalid"}`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"headers": map[string]string{
			"Content-Type": "image/png",
			"content-type": "image/jpeg",
		},
		"body": "binary",
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called with duplicate header names")
	}
	if !strings.Contains(stderr.String(), "duplicate header names") {
		t.Errorf("stderr = %q, want duplicate-header diagnostic", stderr.String())
	}
}

func TestUploaderRejectsManagedTransportHeaders(t *testing.T) {
	for _, header := range []string{"Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Upgrade", "Proxy-Connection"} {
		t.Run(header, func(t *testing.T) {
			var called atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called.Store(true)
				fmt.Fprint(w, `{"url":"https://files.example.test/invalid"}`)
			}))
			defer server.Close()

			home, filePath := writeV3Fixture(t, server.URL, map[string]any{
				"method":  "POST",
				"url":     server.URL,
				"headers": map[string]string{header: "invalid"},
				"body":    "binary",
			}, map[string]any{
				"url": map[string]any{"type": "body"},
			}, []byte("payload"))

			var stdout, stderr bytes.Buffer
			exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
			if exitCode != 1 {
				t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
			}
			if called.Load() {
				t.Fatal("upload endpoint was called with a managed header")
			}
			if !strings.Contains(stderr.String(), "managed by the HTTP transport") {
				t.Errorf("stderr = %q, want managed-header diagnostic", stderr.String())
			}
		})
	}
}

func TestUploaderRejectsDELInHeaderName(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/invalid"}`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method":  "POST",
		"url":     server.URL,
		"headers": map[string]string{"Bad\x7fName": "invalid"},
		"body":    "binary",
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called with a DEL header name")
	}
	if !strings.Contains(stderr.String(), "invalid header") {
		t.Errorf("stderr = %q, want invalid-header diagnostic", stderr.String())
	}
}

func TestUploaderRejectsNullRegexGroup(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `https://files.example.test/invalid`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url": map[string]any{
			"type":    "regex",
			"pattern": `(https://files\.example\.test/\w+)`,
			"group":   nil,
		},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called with a null regex group")
	}
	if !strings.Contains(stderr.String(), "regex group must be a string") {
		t.Errorf("stderr = %q, want regex group type diagnostic", stderr.String())
	}
}

func TestFormUploadUsesUTF8InputAndBodyExtractor(t *testing.T) {
	const fileContents = "café & tea\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want form content type", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.PostForm.Get("payload"); got != fileContents {
			t.Errorf("payload = %q, want %q", got, fileContents)
		}
		if got := r.PostForm.Get("token"); got != "static" {
			t.Errorf("token = %q, want static", got)
		}
		fmt.Fprint(w, " \nhttps://files.example.test/form\t")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}", "token": "static"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte(fileContents))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "https://files.example.test/form\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestJSONUploadPreservesTemplateAndUsesRegexGroup(t *testing.T) {
	const fileContents = "quote: \"yes\"\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode JSON body: %v", err)
		}
		if got := body["payload"]; got != fileContents {
			t.Errorf("payload = %q, want %q", got, fileContents)
		}
		if got := body["{input}"]; got != "literal" {
			t.Errorf("placeholder-like key = %#v, want literal", got)
		}
		meta, ok := body["meta"].(map[string]any)
		if !ok || meta["enabled"] != true {
			t.Errorf("meta = %#v, want enabled=true", body["meta"])
		}
		fmt.Fprint(w, "created URL: https://files.example.test/json\n")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "json",
		"data": map[string]any{
			"{input}": "literal",
			"meta":    map[string]any{"enabled": true},
			"payload": "{input}",
			"tags":    []any{"one", 2, nil},
		},
	}, map[string]any{
		"url": map[string]any{
			"type":    "regex",
			"pattern": `(?P<url>https://files\.example\.test/[^\s]+)`,
			"group":   "url",
		},
	}, []byte(fileContents))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "https://files.example.test/json\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestTextUploadRejectsInvalidUTF8BeforeNetwork(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, "https://files.example.test/should-not-run")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte{0xff, 0xfe})

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called for invalid UTF-8 input")
	}
	if !strings.Contains(stderr.String(), "Stage: validation") || !strings.Contains(stderr.String(), "valid UTF-8") {
		t.Errorf("stderr = %q, want UTF-8 validation failure", stderr.String())
	}
}

func TestUploaderVersion1IsRejectedBeforeNetwork(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/legacy"}`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte("payload"))
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	data, err := os.ReadFile(uploaderPath)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"version": 2`), []byte(`"version": 1`), 1)
	if err := os.WriteFile(uploaderPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called for version-1 Uploader")
	}
	if !strings.Contains(stderr.String(), "custom uploader version = 1, want 2") {
		t.Errorf("stderr = %q, want version cutover diagnostic", stderr.String())
	}
}

func TestUploaderRejectsUnknownNestedRequestField(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/invalid"}`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method":  "POST",
		"url":     server.URL,
		"body":    "binary",
		"unknown": true,
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called for an unknown request field")
	}
	if !strings.Contains(stderr.String(), `unknown field "unknown"`) {
		t.Errorf("stderr = %q, want strict nested-field diagnostic", stderr.String())
	}
}

func TestHeaderExtractorRejectsDuplicateValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Location", "https://files.example.test/one")
		w.Header().Add("Location", "https://files.example.test/two")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Stage: parse") || !strings.Contains(stderr.String(), "want 1") {
		t.Errorf("stderr = %q, want duplicate-header parse failure", stderr.String())
	}
}

func TestProviderErrorIsNormalizedAndRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "token=secret\nvalue\nsecond\tline")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"query":  map[string]string{"token": "secret\tvalue"},
		"body":   "binary",
	}, map[string]any{
		"url":   map[string]any{"type": "body"},
		"error": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if strings.Contains(stderr.String(), "secret\n") || strings.Contains(stderr.String(), "second\tline") {
		t.Errorf("stderr contains unnormalized provider text: %q", stderr.String())
	}
	if strings.Contains(stderr.String(), "secret value") {
		t.Errorf("stderr leaked normalized secret: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "token=[REDACTED] second line") {
		t.Errorf("stderr = %q, want normalized redacted message", stderr.String())
	}
}

func TestHeaderAndRegexErrorExtractors(t *testing.T) {
	t.Run("header error on non-2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Provider-Error", "header failure")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "ignored body")
		}))
		defer server.Close()

		home, filePath := writeV3Fixture(t, server.URL, map[string]any{
			"method": "POST",
			"url":    server.URL,
			"body":   "binary",
		}, map[string]any{
			"url":   map[string]any{"type": "body"},
			"error": map[string]any{"type": "header", "header": "X-Provider-Error"},
		}, []byte("payload"))

		var stdout, stderr bytes.Buffer
		exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
		if exitCode != 1 || !strings.Contains(stderr.String(), "header failure") || !strings.Contains(stderr.String(), "Stage: response") {
			t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
		}
	})

	t.Run("regex error on URL parse failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "provider=regex-failure")
		}))
		defer server.Close()

		home, filePath := writeV3Fixture(t, server.URL, map[string]any{
			"method": "POST",
			"url":    server.URL,
			"body":   "binary",
		}, map[string]any{
			"url":   map[string]any{"type": "header", "header": "Location"},
			"error": map[string]any{"type": "regex", "pattern": `provider=(regex-[a-z]+)`, "group": "1"},
		}, []byte("payload"))

		var stdout, stderr bytes.Buffer
		exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
		if exitCode != 1 || !strings.Contains(stderr.String(), "regex-failure") || !strings.Contains(stderr.String(), "Stage: parse") {
			t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
		}
	})
}

func TestInvalidURLUsesProviderErrorExtractor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "relative/path")
		fmt.Fprint(w, "provider\nmessage")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url":   map[string]any{"type": "header", "header": "Location"},
		"error": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "provider message") {
		t.Errorf("stderr = %q, want provider error precedence", stderr.String())
	}
	if strings.Contains(stderr.String(), "relative/path") {
		t.Errorf("stderr = %q, want invalid URL hidden by provider diagnostic", stderr.String())
	}
}

func cliRunner(home string) cli.Runner {
	return cli.Runner{HomeDir: func() (string, error) { return home, nil }}
}

func writeV3Fixture(t *testing.T, endpoint string, request, response map[string]any, fileContents []byte) (home, filePath string) {
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
	document := map[string]any{
		"version": 2,
		"uploaders": map[string]any{
			"test": map[string]any{
				"request":  request,
				"response": response,
			},
		},
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	filePath = filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(filePath, fileContents, 0o600); err != nil {
		t.Fatal(err)
	}
	return home, filePath
}
