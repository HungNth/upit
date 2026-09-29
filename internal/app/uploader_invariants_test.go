package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestValidateUTF8FileAcrossBufferBoundary(t *testing.T) {
	tests := map[string]struct {
		contents []byte
		wantErr  bool
	}{
		"valid rune crosses buffer boundary": {
			contents: append([]byte(strings.Repeat("a", 32<<10-1)), []byte("é")...),
		},
		"invalid sequence": {
			contents: append([]byte(strings.Repeat("a", 32<<10-1)), 0xc3, 0x28),
			wantErr:  true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := t.TempDir() + "/input"
			if err := os.WriteFile(path, test.contents, 0o600); err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			err = validateUTF8File(context.Background(), file)
			file.Close()
			if (err != nil) != test.wantErr {
				t.Fatalf("validateUTF8File() error = %v, want error = %t", err, test.wantErr)
			}
		})
	}
}

func TestFormEncodingInvariant(t *testing.T) {
	path := t.TempDir() + "/input"
	if err := os.WriteFile(path, []byte("a b+~é\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var output bytes.Buffer
	if err := writeFormEncodedFile(&output, file); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "a+b%2B%7E%C3%A9%00"; got != want {
		t.Errorf("encoded file = %q, want %q", got, want)
	}
}

func TestJSONTemplateEscapesInputAndPreservesPlaceholderLikeKeys(t *testing.T) {
	path := t.TempDir() + "/input"
	const contents = "quote: \"yes\"\nline\t"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var output bytes.Buffer
	data := map[string]any{"{input}": "literal", "payload": inputPlaceholder}
	if err := writeJSONBody(&output, file, data); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("decode generated JSON: %v; body = %q", err, output.String())
	}
	if decoded["{input}"] != "literal" || decoded["payload"] != contents {
		t.Errorf("decoded JSON = %#v, want placeholder-like key and escaped input", decoded)
	}
}

func TestRegexGroupSelectionInvariant(t *testing.T) {
	extractor := extractorConfig{
		Type:       "regex",
		Group:      "1",
		regex:      regexp.MustCompile(`URL=(https://files\.example\.test/\w+)`),
		groupIndex: 1,
	}
	got, err := extractResponseValue(&http.Response{Header: make(http.Header)}, []byte("URL=https://files.example.test/item"), extractor, "response URL")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://files.example.test/item" {
		t.Errorf("selected regex group = %q, want full URL", got)
	}
}
