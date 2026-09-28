package app

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode"

	"github.com/theory/jsonpath"
)

func loadConfiguration(homeDir string) (settings, uploaderDocument, error) {
	configDir := filepath.Join(homeDir, ".config", "upit")

	configPath := filepath.Join(configDir, "config.json")
	var global settings
	if err := readJSON(configPath, &global); err != nil {
		return settings{}, uploaderDocument{}, failuref("config", err, "load %s: %v; copy examples/config.example.json to this location", configPath, err)
	}
	if global.Version != 1 {
		return settings{}, uploaderDocument{}, failuref("validation", nil, "config version = %d, want 1", global.Version)
	}
	if global.DefaultUploader == "" {
		return settings{}, uploaderDocument{}, failure("validation", "defaultUploader is required", nil)
	}

	uploaderPath := filepath.Join(configDir, "custom-uploader.json")
	if err := validateUploaderFilePermissions(uploaderPath); err != nil {
		return settings{}, uploaderDocument{}, err
	}
	var uploaders uploaderDocument
	if err := readJSON(uploaderPath, &uploaders); err != nil {
		return settings{}, uploaderDocument{}, failuref("config", err, "load %s: %v; copy examples/custom-uploader.example.json to this location", uploaderPath, err)
	}
	if uploaders.Version != 1 {
		return settings{}, uploaderDocument{}, failuref("validation", nil, "custom uploader version = %d, want 1", uploaders.Version)
	}

	for name, candidate := range uploaders.Uploaders {
		if err := validateUploader(name, &candidate); err != nil {
			return settings{}, uploaderDocument{}, err
		}
		uploaders.Uploaders[name] = candidate
	}

	return global, uploaders, nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("decode %s: multiple JSON values", path)
	}
	return nil
}

func validateUploaderFilePermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.Mode().Perm()&0o077 != 0 {
		return failuref("validation", nil, "%s contains credentials and must have mode 0600; run: chmod 600 %s", path, path)
	}
	return nil
}

func validateUploader(name string, candidate *uploader) error {
	request, err := http.NewRequest(candidate.Request.Method, candidate.Request.URL, nil)
	if err != nil {
		return failuref("validation", err, "uploader %q: invalid request method or URL", name)
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" || request.URL.Host == "" {
		return failuref("validation", nil, "uploader %q: request URL must be absolute HTTP(S)", name)
	}
	if candidate.Request.Body != "multipart" {
		return failuref("validation", nil, "uploader %q: body must be multipart", name)
	}
	if candidate.Request.FileField == "" {
		return failuref("validation", nil, "uploader %q: fileField is required", name)
	}
	for key, value := range candidate.Request.Headers {
		if !validHeaderName(key) || strings.ContainsAny(value, "\r\n") {
			return failuref("validation", nil, "uploader %q: invalid header %q", name, key)
		}
		if strings.EqualFold(key, "Content-Type") {
			return failuref("validation", nil, "uploader %q: Content-Type is owned by Upit for multipart boundaries", name)
		}
	}
	for key := range candidate.Request.Query {
		if key == "" {
			return failuref("validation", nil, "uploader %q: query keys must not be empty", name)
		}
	}
	for key := range candidate.Request.Fields {
		if key == "" {
			return failuref("validation", nil, "uploader %q: multipart field names must not be empty", name)
		}
	}
	if err := compileExtractor(name, "response URL", &candidate.Response.URL); err != nil {
		return err
	}
	if candidate.Response.Error != nil {
		if err := compileExtractor(name, "response error", candidate.Response.Error); err != nil {
			return err
		}
	}
	candidate.sensitiveValues = collectSensitiveValues(candidate.Request)
	return nil
}

func compileExtractor(uploaderName, label string, extractor *extractorConfig) error {
	if extractor.Type != "json" {
		return failuref("validation", nil, "uploader %q: %s type must be json", uploaderName, label)
	}
	path, err := jsonpath.Parse(extractor.Path)
	if err != nil {
		return failuref("validation", err, "uploader %q: invalid %s path", uploaderName, label)
	}
	extractor.compiled = path
	return nil
}

func collectSensitiveValues(request requestConfig) []string {
	values := make([]string, 0, len(request.Headers)+len(request.Query)+len(request.Fields))
	appendValue := func(value string) {
		if value != "" {
			values = append(values, value)
		}
	}
	for value := range maps.Values(request.Headers) {
		appendValue(value)
	}
	for value := range maps.Values(request.Query) {
		appendValue(value)
	}
	for value := range maps.Values(request.Fields) {
		appendValue(value)
	}
	parsed, err := url.Parse(request.URL)
	if err == nil {
		for queryValues := range maps.Values(parsed.Query()) {
			for _, value := range queryValues {
				appendValue(value)
			}
		}
	}
	slices.SortFunc(values, func(a, b string) int {
		if order := cmp.Compare(len(b), len(a)); order != 0 {
			return order
		}
		return cmp.Compare(a, b)
	})
	return slices.Compact(values)
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r > unicode.MaxASCII || r <= ' ' || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return false
		}
	}
	return true
}
