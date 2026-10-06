package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBinaryValidationRejectsInvalidContentType(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/invalid"}`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method":  "POST",
		"url":     server.URL,
		"headers": map[string]string{"Content-Type": "not a media type"},
		"body":    "binary",
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 || called.Load() {
		t.Fatalf("exit code = %d, called = %t, stdout = %q, stderr = %q", exitCode, called.Load(), stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid binary Content-Type") {
		t.Errorf("stderr = %q, want invalid media type diagnostic", stderr.String())
	}
}

func TestBinaryUploadCancellationExits130(t *testing.T) {
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

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "PUT",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte("payload"))
	if err := os.Truncate(filePath, 16<<20); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- cliRunner(home).Run(ctx, []string{"upload", filePath}, &stdout, &stderr)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("binary endpoint never received request")
	}
	cancel()
	select {
	case exitCode := <-exitCodes:
		if exitCode != 130 {
			t.Fatalf("exit code = %d, want 130; stderr = %q", exitCode, stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("binary upload did not exit after cancellation")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("binary endpoint did not observe cancellation")
	}
}

func TestBinaryUploadHandlesLargeInput(t *testing.T) {
	const size = 4 << 20
	var received atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			t.Errorf("read large binary body: %v", err)
		}
		received.Store(count)
		w.Header().Set("Location", "https://files.example.test/large")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "PUT",
		"url":    server.URL,
		"body":   "binary",
	}, map[string]any{
		"url": map[string]any{"type": "header", "header": "Location"},
	}, []byte("payload"))
	if err := os.Truncate(filePath, size); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got := received.Load(); got != size {
		t.Errorf("received bytes = %d, want %d", got, size)
	}
}

func TestFormUploadAcceptsEmptyInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.PostForm.Get("payload"); got != "" {
			t.Errorf("payload = %q, want empty", got)
		}
		fmt.Fprint(w, "https://files.example.test/empty")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, nil)

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
	if got, want := stdout.String(), "https://files.example.test/empty\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestJSONUploadRejectsInvalidUTF8BeforeNetwork(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `https://files.example.test/invalid`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "json",
		"data":   map[string]any{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte{0xff, 0xfe})

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 || called.Load() {
		t.Fatalf("exit code = %d, called = %t, stderr = %q", exitCode, called.Load(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Stage: validation") || !strings.Contains(stderr.String(), "valid UTF-8") {
		t.Errorf("stderr = %q, want JSON UTF-8 validation failure", stderr.String())
	}
}

func TestFormPreflightCancellationExits130(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `https://files.example.test/invalid`)
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))
	if err := os.Truncate(filePath, 64<<20); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- cliRunner(home).Run(ctx, []string{"upload", filePath}, &stdout, &stderr)
	}()
	select {
	case exitCode := <-exitCodes:
		if exitCode != 130 {
			t.Fatalf("exit code = %d, want 130; stderr = %q", exitCode, stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("form preflight did not stop after cancellation")
	}
	if called.Load() || stdout.Len() != 0 {
		t.Fatalf("endpoint called = %t; stdout = %q", called.Load(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "Stage: validation") {
		t.Errorf("stderr = %q, want validation stage", stderr.String())
	}
}

func TestFormUploadEncodesUTF8AtBufferBoundary(t *testing.T) {
	fileContents := strings.Repeat("a", 32<<10-1) + "é +&\x00"
	expected := "payload=" + strings.Repeat("a", 32<<10-1) + "%C3%A9+%2B%26%00&static=value%2B%7E"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read form body: %v", err)
		}
		if string(body) != expected {
			t.Errorf("form body has unexpected encoding: suffix=%q", string(body[len(body)-40:]))
		}
		fmt.Fprint(w, "https://files.example.test/form-encoding")
	}))
	defer server.Close()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}", "static": "value+~"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte(fileContents))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %q", exitCode, stderr.String())
	}
}

func TestFormUploadTimeoutCancelsTransformedRequest(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(2 * time.Second)
	}))
	defer func() {
		server.CloseClientConnections()
		server.Close()
	}()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))
	if err := os.Truncate(filePath, 1<<20); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- cliRunner(home).Run(context.Background(), []string{"upload", filePath, "--timeout", "1s"}, &stdout, &stderr)
	}()
	var exitCode int
	select {
	case exitCode = <-exitCodes:
	case <-time.After(5 * time.Second):
		t.Fatal("form timeout runner did not exit")
	}
	if exitCode != 1 {
		t.Fatalf("exit code = %d, want 1; stderr = %q", exitCode, stderr.String())
	}
	if !strings.Contains(stderr.String(), "deadline exceeded") {
		t.Errorf("stderr = %q, want deadline diagnostic", stderr.String())
	}
	select {
	case <-started:
	default:
		t.Fatal("form endpoint never received request")
	}
}

func TestFormUploadParentCancellationExits130(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(time.Second)
	}))
	defer func() {
		server.CloseClientConnections()
		server.Close()
	}()

	home, filePath := writeV3Fixture(t, server.URL, map[string]any{
		"method": "POST",
		"url":    server.URL,
		"body":   "form",
		"fields": map[string]string{"payload": "{input}"},
	}, map[string]any{
		"url": map[string]any{"type": "body"},
	}, []byte("payload"))
	if err := os.Truncate(filePath, 16<<20); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	exitCodes := make(chan int, 1)
	go func() {
		exitCodes <- cliRunner(home).Run(ctx, []string{"upload", filePath}, &stdout, &stderr)
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("form endpoint never received request")
	}
	cancel()
	select {
	case exitCode := <-exitCodes:
		if exitCode != 130 {
			t.Fatalf("exit code = %d, want 130; stderr = %q", exitCode, stderr.String())
		}
	case <-time.After(time.Second):
		t.Fatal("form upload did not exit after cancellation")
	}
}

func TestRegexExtractorBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		response  []byte
		extractor map[string]any
		wantURL   string
		wantError string
	}{
		{
			name:      "default whole match",
			response:  []byte("prefix https://files.example.test/first suffix"),
			extractor: map[string]any{"type": "regex", "pattern": `https://files\.example\.test/\w+`},
			wantURL:   "https://files.example.test/first",
		},
		{
			name:      "explicit whole match",
			response:  []byte("prefix https://files.example.test/explicit suffix"),
			extractor: map[string]any{"type": "regex", "pattern": `https://files\.example\.test/\w+`, "group": "0"},
			wantURL:   "https://files.example.test/explicit",
		},
		{
			name:      "numeric group",
			response:  []byte("URL=https://files.example.test/numeric"),
			extractor: map[string]any{"type": "regex", "pattern": `URL=(https://files\.example\.test/\w+)`, "group": "1"},
			wantURL:   "https://files.example.test/numeric",
		},
		{
			name:      "named group",
			response:  []byte("URL=https://files.example.test/named"),
			extractor: map[string]any{"type": "regex", "pattern": `URL=(?P<url>https://files\.example\.test/\w+)`, "group": "url"},
			wantURL:   "https://files.example.test/named",
		},
		{
			name:      "first match",
			response:  []byte("https://files.example.test/one https://files.example.test/two"),
			extractor: map[string]any{"type": "regex", "pattern": `https://files\.example\.test/\w+`},
			wantURL:   "https://files.example.test/one",
		},
		{
			name:      "no match",
			response:  []byte("no URL"),
			extractor: map[string]any{"type": "regex", "pattern": `https://files\.example\.test/\w+`},
			wantError: "Stage: parse",
		},
		{
			name:      "empty capture",
			response:  []byte("URL="),
			extractor: map[string]any{"type": "regex", "pattern": `URL=(https://files\.example\.test/\w+)?`, "group": "1"},
			wantError: "empty or unmatched",
		},
		{
			name:      "invalid UTF-8",
			response:  []byte{0xff},
			extractor: map[string]any{"type": "regex", "pattern": `.`},
			wantError: "not valid UTF-8",
		},
		{
			name:      "invalid URL",
			response:  []byte("relative/path"),
			extractor: map[string]any{"type": "regex", "pattern": `relative/[^\s]+`},
			wantError: "absolute HTTP(S) URL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(test.response)
			}))
			defer server.Close()

			home, filePath := writeV3Fixture(t, server.URL, map[string]any{
				"method": "POST",
				"url":    server.URL,
				"body":   "binary",
			}, map[string]any{
				"url": test.extractor,
			}, []byte("payload"))

			var stdout, stderr bytes.Buffer
			exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
			if test.wantURL != "" {
				if exitCode != 0 || stdout.String() != test.wantURL+"\n" {
					t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
				}
				return
			}
			if exitCode != 1 || !strings.Contains(stderr.String(), test.wantError) {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q; want %q", exitCode, stdout.String(), stderr.String(), test.wantError)
			}
		})
	}
}

func TestHeaderExtractorBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		configured  string
		setHeader   bool
		headerValue string
		wantURL     string
		wantError   string
	}{
		{name: "case insensitive", configured: "location", setHeader: true, headerValue: "https://files.example.test/case", wantURL: "https://files.example.test/case"},
		{name: "missing", configured: "Location", wantError: "has no values"},
		{name: "whitespace only", configured: "Location", setHeader: true, headerValue: "   ", wantError: "is empty"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.setHeader {
					w.Header().Set("Location", test.headerValue)
				}
			}))
			defer server.Close()

			home, filePath := writeV3Fixture(t, server.URL, map[string]any{
				"method": "POST",
				"url":    server.URL,
				"body":   "binary",
			}, map[string]any{
				"url": map[string]any{"type": "header", "header": test.configured},
			}, []byte("payload"))

			var stdout, stderr bytes.Buffer
			exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
			if test.wantURL != "" {
				if exitCode != 0 || stdout.String() != test.wantURL+"\n" {
					t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
				}
				return
			}
			if exitCode != 1 || !strings.Contains(stderr.String(), test.wantError) {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q; want %q", exitCode, stdout.String(), stderr.String(), test.wantError)
			}
		})
	}
}

func TestHeaderExtractorHonorsResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://files.example.test/oversized")
		_, _ = w.Write(make([]byte, (1<<20)+1))
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
	if exitCode != 1 || stdout.Len() != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "Stage: response") || !strings.Contains(stderr.String(), "exceeds 1 MiB") {
		t.Errorf("stderr = %q, want bounded header response failure", stderr.String())
	}
}

func TestBodyExtractorBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		response  []byte
		wantError string
	}{
		{name: "empty", response: nil, wantError: "body is empty"},
		{name: "whitespace only", response: []byte(" \n\t"), wantError: "body is empty"},
		{name: "invalid UTF-8", response: []byte{0xff}, wantError: "not valid UTF-8"},
		{name: "invalid URL", response: []byte("relative/path"), wantError: "absolute HTTP(S) URL"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(test.response)
			}))
			defer server.Close()

			home, filePath := writeV3Fixture(t, server.URL, map[string]any{
				"method": "POST",
				"url":    server.URL,
				"body":   "binary",
			}, map[string]any{
				"url": map[string]any{"type": "body"},
			}, []byte("payload"))

			var stdout, stderr bytes.Buffer
			exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
			if exitCode != 1 || !strings.Contains(stderr.String(), test.wantError) {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q; want %q", exitCode, stdout.String(), stderr.String(), test.wantError)
			}
		})
	}
}

func TestUploaderRejectsEmptyRegexGroup(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, "https://files.example.test/invalid")
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
			"group":   "",
		},
	}, []byte("payload"))

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 || called.Load() {
		t.Fatalf("exit code = %d, called = %t, stderr = %q", exitCode, called.Load(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "regex group must not be empty") {
		t.Errorf("stderr = %q, want empty-group diagnostic", stderr.String())
	}
}
