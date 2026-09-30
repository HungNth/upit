package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestUploadRejectsNestedDuplicateKeyBeforeNetwork(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		fmt.Fprint(w, `{"url":"https://files.example.test/original"}`)
	}))
	defer server.Close()

	home, filePath := writeBasicUploadFixture(t, server.URL, "$.url", "")
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	duplicate := fmt.Sprintf(`{
  "version": 2,
  "uploaders": {
    "test": {
      "request": {
        "method": "POST",
        "url": %q,
        "headers": {
          "Authorization": "first",
          "Authorization": "second"
        },
        "body": "multipart",
        "fileField": "file"
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`, server.URL)
	if err := os.WriteFile(uploaderPath, []byte(duplicate), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if called.Load() {
		t.Fatal("upload endpoint was called for duplicate configuration")
	}
	message := stderr.String()
	if !strings.Contains(message, filepath.Clean(uploaderPath)) || !strings.Contains(message, `$.uploaders["test"].request.headers`) {
		t.Errorf("stderr = %q, want absolute file and nested configuration path", message)
	}
	if !strings.Contains(message, `duplicate key "Authorization"`) {
		t.Errorf("stderr = %q, want duplicate-key cause", message)
	}
	if strings.Contains(message, "first") || strings.Contains(message, "second") {
		t.Errorf("stderr = %q, leaked configured header value", message)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestUploadReportsMalformedJSONLocation(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	malformed := "{\n  \"version\": 2,\n  \"defaultUploader\": \"test\",\n  \"copyToClipboard\": false,\n}\n"
	if err := os.WriteFile(configPath, []byte(malformed), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, filepath.Clean(configPath)) || !strings.Contains(message, "$") || !strings.Contains(message, "line") || !strings.Contains(message, "column") {
		t.Errorf("stderr = %q, want absolute root JSON location", message)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
}

func TestUploadRejectsInvalidDefinitionNameWithPath(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	invalid := fmt.Sprintf(`{
  "version": 2,
  "uploaders": {
    %q: {
      "request": {"method": "POST", "url": "https://upload.example.test", "body": "binary"},
      "response": {"url": {"type": "body"}}
    }
  }
}`, " leading-space")
	if err := os.WriteFile(uploaderPath, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, `$.uploaders[" leading-space"]`) || !strings.Contains(message, "printable Unicode") {
		t.Errorf("stderr = %q, want invalid-name path and correction", message)
	}
}

func TestUploadRejectsEmptyUploaderDocument(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	if err := os.WriteFile(uploaderPath, []byte(`{"version":2,"uploaders":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	if message := stderr.String(); !strings.Contains(message, `$.uploaders`) || !strings.Contains(message, "at least one Uploader") {
		t.Errorf("stderr = %q, want empty-document diagnostic", message)
	}
}

func TestUploadRejectsCaseVariantDuplicateShortenerHeaders(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "defaultShortener": "selected",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	shortenerPath := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	if err := os.WriteFile(shortenerPath, []byte(`{
  "version": 1,
  "shorteners": {
    "selected": {
      "request": {
        "method": "POST",
        "url": "https://short.example.test/links",
        "headers": {"X-Key": "one", "x-key": "two"},
        "data": {"target": "{input}"}
      },
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, `$.shorteners["selected"].request.headers`) || !strings.Contains(message, "duplicate header names") {
		t.Errorf("stderr = %q, want Shortener duplicate-header diagnostic", message)
	}
}

func TestUploadValidatesDefinitionNamesInLexicographicOrder(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	if err := os.WriteFile(uploaderPath, []byte(`{
  "version": 2,
  "uploaders": {
    "zulu": {"request": {}, "response": {"url": {"type": "body"}}},
    "alpha": {"request": {}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, `$.uploaders["alpha"].request.method`) || strings.Contains(message, `$.uploaders["zulu"]`) {
		t.Errorf("stderr = %q, want first lexicographic definition error only", message)
	}
}

func TestConfigPathPrintsFixedAbsoluteDirectoryWithoutCreatingIt(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "path"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if got, want := stdout.String(), configDir+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatalf("config directory stat error = %v, want directory to remain absent", err)
	}
}

func TestConfigShowLoadsOnlyGlobalConfiguration(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "missing-uploader",
  "copyToClipboard": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "show"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	want := "Default Uploader: \"missing-uploader\"\nDefault Shortener: none\nCopy to Clipboard: enabled\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestConfigListUploadersSortsAndQuotesNames(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "zulu": {"request": {"method": "POST", "url": "https://upload.example.test/z", "body": "binary"}, "response": {"url": {"type": "body"}}},
    "alpha beta": {"request": {"method": "POST", "url": "https://upload.example.test/a", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "list-uploaders"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	want := "Uploaders:\n- \"alpha beta\"\n- \"zulu\"\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

func TestConfigListShortenersReportsAbsentOptionalDocument(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "list-shorteners"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if stdout.String() != "No Shorteners configured.\n" || stderr.Len() != 0 {
		t.Errorf("stdout = %q; stderr = %q, want absent-document success", stdout.String(), stderr.String())
	}
}

func TestConfigValidateChecksGlobalBeforeUploaderAndShortener(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{"version":2,"uploaders":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, "defaultUploader is required") || strings.Contains(message, "at least one Uploader") || stdout.Len() != 0 {
		t.Errorf("stdout = %q; stderr = %q, want first Global error only", stdout.String(), message)
	}
}

func TestConfigValidatePrintsSuccessForCompleteSet(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	if err := os.Remove(filepath.Join(home, ".config", "upit", "config.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".config", "upit", "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Configuration is valid.\n" || stderr.Len() != 0 {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want validation success", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigValidateRejectsPresentInvalidOptionalShortener(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	shortenerPath := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	if err := os.WriteFile(shortenerPath, []byte(`{"version":1,"shorteners":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "at least one Shortener") {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want present Shortener failure", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigUsageAndHelpAreCommandSpecific(t *testing.T) {
	home := t.TempDir()
	var helpOut, helpErr bytes.Buffer
	if exitCode := cliRunner(home).Run(context.Background(), []string{"config", "--help"}, &helpOut, &helpErr); exitCode != 0 {
		t.Fatalf("config help exit code = %d; stdout = %q; stderr = %q", exitCode, helpOut.String(), helpErr.String())
	}
	if !strings.Contains(helpOut.String(), "upit config") || !strings.Contains(helpOut.String(), "list-uploaders") || helpErr.Len() != 0 {
		t.Errorf("config help stdout = %q; stderr = %q", helpOut.String(), helpErr.String())
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "show", "--json"}, &stdout, &stderr)
	if exitCode != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage: upit config") {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want config usage failure", exitCode, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exitCode = cliRunner(home).Run(context.Background(), []string{"config", "path", "--help"}, &stdout, &stderr)
	if exitCode != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage: upit config") {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want subcommand usage failure", exitCode, stdout.String(), stderr.String())
	}
}

func TestUploadNormalizesNestedMalformedJSONToRootPath(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	uploaderPath := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	malformed := `{
  "version": 2,
  "uploaders": {
    "test": {
      "request": {
        "method": "POST",
        "url": "https://upload.example.test"
`
	if err := os.WriteFile(uploaderPath, []byte(malformed), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, filepath.Clean(uploaderPath)+" $") || strings.Contains(message, "$.uploaders") || !strings.Contains(message, "line") || !strings.Contains(message, "column") {
		t.Errorf("stderr = %q, want root-path malformed JSON diagnostic", message)
	}
}

func TestUploadUnsupportedVersionReportsMigrationRemediation(t *testing.T) {
	home, filePath := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 1,
  "defaultUploader": "test",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"upload", filePath}, &stdout, &stderr)
	if exitCode != 1 {
		t.Fatalf("exit code = %d; stderr = %q", exitCode, stderr.String())
	}
	message := stderr.String()
	if !strings.Contains(message, filepath.Clean(configPath)+" $.version") || !strings.Contains(message, "config version = 1, want 2") || !strings.Contains(message, "automatic migration is unavailable") || !strings.Contains(message, "received version 1") || !strings.Contains(message, "expected version 2") || !strings.Contains(message, "schemas/config.schema.json") {
		t.Errorf("stderr = %q, want exact unsupported-version remediation", message)
	}
}

func TestConfigSetDefaultUploaderPublishesCanonicalGlobalConfiguration(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "defaultShortener": "short",
  "copyToClipboard": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "test": {"request": {"method": "POST", "url": "https://upload.example.test/test", "body": "binary"}, "response": {"url": {"type": "body"}}},
    "next": {"request": {"method": "POST", "url": "https://upload.example.test/next", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-uploader", "next"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	if stdout.String() != "Default Uploader set to \"next\".\n" || stderr.Len() != 0 {
		t.Errorf("stdout = %q; stderr = %q, want mutation confirmation", stdout.String(), stderr.String())
	}
	want := "{\n    \"version\": 2,\n    \"defaultUploader\": \"next\",\n    \"defaultShortener\": \"short\",\n    \"copyToClipboard\": true\n}\n"
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Errorf("config bytes = %q, want %q", string(data), want)
	}
}

func TestConfigValidateRequiresSelectedShortenerDocumentAndName(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "defaultShortener": "selected",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "requires custom-shortener.json") {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q, want missing selected Shortener document", exitCode, stdout.String(), stderr.String())
	}

	shortenerPath := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	if err := os.WriteFile(shortenerPath, []byte(`{
  "version": 1,
  "shorteners": {
    "other": {
      "request": {"method": "POST", "url": "https://short.example.test", "data": {"target": "{input}"}},
      "response": {"url": {"type": "json", "path": "$.url"}}
    }
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	exitCode = cliRunner(home).Run(context.Background(), []string{"config", "validate"}, &stdout, &stderr)
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), `default Shortener "selected" does not exist`) {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want missing selected Shortener name", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigSetDefaultUploaderNoOpPreservesBytesAndModificationTime(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"version":2,"defaultUploader":"test","copyToClipboard":false}`)
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "test": {"request": {"method": "POST", "url": "https://upload.example.test", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-uploader", "test"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Default Uploader is already \"test\".\n" || stderr.Len() != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	afterInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) || !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Errorf("no-op changed bytes or modification time: bytes=%q, before=%v, after=%v", string(data), beforeInfo.ModTime(), afterInfo.ModTime())
	}
}

func TestConfigSetDefaultUploaderRefusesSymlinkedGlobalConfiguration(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	realPath := filepath.Join(home, "real-config.json")
	if err := os.WriteFile(realPath, []byte(`{"version":2,"defaultUploader":"test","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.Symlink(realPath, configPath); err != nil {
		t.Skipf("creating a symlink is unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-uploader.json"), []byte(`{
  "version": 2,
  "uploaders": {
    "next": {"request": {"method": "POST", "url": "https://upload.example.test", "body": "binary"}, "response": {"url": {"type": "body"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-uploader", "next"}, &stdout, &stderr)
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "symlink") {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want symlink refusal", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigSetDefaultUploaderRepairsMissingCurrentReference(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{
  "version": 2,
  "defaultUploader": "missing",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-uploader", "test"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Default Uploader set to \"test\".\n" || stderr.Len() != 0 {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want repair success", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigSetDefaultUploaderRejectsUnknownTargetWithoutWriting(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configPath := filepath.Join(home, ".config", "upit", "config.json")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-uploader", "missing"}, &stdout, &stderr)
	message := stderr.String()
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(message, `uploader "missing" does not exist`) || !strings.Contains(message, filepath.Clean(configPath)+" $.defaultUploader") {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q, want path-aware unknown-target failure", exitCode, stdout.String(), message)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("config changed after rejected target: before=%q after=%q", string(before), string(after))
	}
}

func TestConfigSetDefaultShortenerRejectsUnknownTargetWithoutWriting(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"version":2,"defaultUploader":"missing-uploader","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(`{"version":1,"shorteners":{"other":{"request":{"method":"POST","url":"https://short.example.test","data":{"target":"{input}"}},"response":{"url":{"type":"json","path":"$.url"}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-shortener", "missing"}, &stdout, &stderr)
	message := stderr.String()
	if exitCode != 1 || stdout.Len() != 0 || !strings.Contains(message, `shortener "missing" does not exist`) || !strings.Contains(message, filepath.Clean(configPath)+" $.defaultShortener") {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q, want path-aware unknown-target failure", exitCode, stdout.String(), message)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("config changed after rejected target: before=%q after=%q", string(before), string(after))
	}
}

func TestConfigSetDefaultShortenerLoadsOnlyShortenerDocument(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "missing-uploader",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(`{
  "version": 1,
  "shorteners": {
    "selected": {"request": {"method": "POST", "url": "https://short.example.test", "data": {"target": "{input}"}}, "response": {"url": {"type": "json", "path": "$.url"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "set-default-shortener", "selected"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Default Shortener set to \"selected\".\n" || stderr.Len() != 0 {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q, want Shortener mutation success", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigClearDefaultShortenerRepairsMalformedOptionalDocument(t *testing.T) {
	home, _ := writeBasicUploadFixture(t, "https://upload.example.test", "$.url", "")
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "test",
  "defaultShortener": "selected",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	shortenerPath := filepath.Join(configDir, "custom-shortener.json")
	if err := os.WriteFile(shortenerPath, []byte(`{"version":1,"shorteners":`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "clear-default-shortener"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Default Shortener cleared.\n" || stderr.Len() != 0 {
		t.Fatalf("exit code = %d; stdout = %q; stderr = %q, want clear success", exitCode, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"defaultShortener": ""`) {
		t.Errorf("config bytes = %q, want cleared default Shortener", string(data))
	}
}

func TestConfigClipboardMutationsDoNotNeedOtherDocuments(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "missing-uploader",
  "copyToClipboard": false
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "enable-clipboard"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Clipboard copying enabled.\n" || stderr.Len() != 0 {
		t.Fatalf("enable exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
	stdout.Reset()
	exitCode = cliRunner(home).Run(context.Background(), []string{"config", "disable-clipboard"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Clipboard copying disabled.\n" {
		t.Errorf("disable exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
}

func TestConfigMutationNoOpOutputsAreExplicit(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{
  "version": 2,
  "defaultUploader": "missing-uploader",
  "defaultShortener": "selected",
  "copyToClipboard": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "custom-shortener.json"), []byte(`{
  "version": 1,
  "shorteners": {
    "selected": {"request": {"method": "POST", "url": "https://short.example.test", "data": {"target": "{input}"}}, "response": {"url": {"type": "json", "path": "$.url"}}}
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"config", "set-default-shortener", "selected"}, want: "Default Shortener is already \"selected\".\n"},
		{args: []string{"config", "enable-clipboard"}, want: "Clipboard copying is already enabled.\n"},
	} {
		var stdout, stderr bytes.Buffer
		if exitCode := cliRunner(home).Run(context.Background(), test.args, &stdout, &stderr); exitCode != 0 || stdout.String() != test.want || stderr.Len() != 0 {
			t.Errorf("args=%v: exit code=%d; stdout=%q; stderr=%q, want no-op output %q", test.args, exitCode, stdout.String(), stderr.String(), test.want)
		}
	}
}

func TestConfigClearAndDisableNoOpOutputs(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"missing-uploader","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"config", "clear-default-shortener"}, want: "Default Shortener is already clear.\n"},
		{args: []string{"config", "disable-clipboard"}, want: "Clipboard copying is already disabled.\n"},
	} {
		var stdout, stderr bytes.Buffer
		if exitCode := cliRunner(home).Run(context.Background(), test.args, &stdout, &stderr); exitCode != 0 || stdout.String() != test.want || stderr.Len() != 0 {
			t.Errorf("args=%v: exit code=%d; stdout=%q; stderr=%q, want %q", test.args, exitCode, stdout.String(), stderr.String(), test.want)
		}
	}
}

func TestConfigShowQuotesShortenerNamedNone(t *testing.T) {
	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "upit")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"version":2,"defaultUploader":"missing-uploader","defaultShortener":"none","copyToClipboard":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	exitCode := cliRunner(home).Run(context.Background(), []string{"config", "show"}, &stdout, &stderr)
	if exitCode != 0 || stdout.String() != "Default Uploader: \"missing-uploader\"\nDefault Shortener: \"none\"\nCopy to Clipboard: disabled\n" || stderr.Len() != 0 {
		t.Errorf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout.String(), stderr.String())
	}
}
