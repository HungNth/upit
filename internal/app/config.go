package app

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
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
	if global.Version != 2 {
		return settings{}, uploaderDocument{}, failuref("validation", nil, "config version = %d, want 2", global.Version)
	}
	if global.DefaultUploader == "" {
		return settings{}, uploaderDocument{}, failure("validation", "defaultUploader is required", nil)
	}

	uploaderPath := filepath.Join(configDir, "custom-uploader.json")
	if err := validateCredentialFilePermissions(uploaderPath); err != nil {
		return settings{}, uploaderDocument{}, err
	}
	var uploaders uploaderDocument
	if err := readJSON(uploaderPath, &uploaders); err != nil {
		return settings{}, uploaderDocument{}, failuref("config", err, "load %s: %v; copy examples/custom-uploader.example.json to this location", uploaderPath, err)
	}
	if uploaders.Version != 2 {
		return settings{}, uploaderDocument{}, failuref("validation", nil, "custom uploader version = %d, want 2", uploaders.Version)
	}

	for name, candidate := range uploaders.Uploaders {
		if err := validateUploader(name, &candidate); err != nil {
			return settings{}, uploaderDocument{}, err
		}
		uploaders.Uploaders[name] = candidate
	}

	return global, uploaders, nil
}

func loadShortenerConfiguration(homeDir string) (shortenerDocument, error) {
	path := filepath.Join(homeDir, ".config", "upit", "custom-shortener.json")
	if err := validateCredentialFilePermissions(path); err != nil {
		return shortenerDocument{}, err
	}
	var shorteners shortenerDocument
	if err := readJSON(path, &shorteners); err != nil {
		return shortenerDocument{}, failuref("config", err, "load %s: %v; copy examples/custom-shortener.example.json to this location", path, err)
	}
	if shorteners.Version != 1 {
		return shortenerDocument{}, failuref("validation", nil, "custom shortener version = %d, want 1", shorteners.Version)
	}
	for name, candidate := range shorteners.Shorteners {
		if err := validateShortener(name, &candidate); err != nil {
			return shortenerDocument{}, err
		}
		shorteners.Shorteners[name] = candidate
	}
	return shorteners, nil
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
		if err == nil {
			return fmt.Errorf("decode %s: multiple JSON values", path)
		}
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func validateCredentialFilePermissions(path string) error {
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
	if candidate.Request.Method == "" {
		return failuref("validation", nil, "uploader %q: request method is required", name)
	}
	request, err := http.NewRequest(candidate.Request.Method, candidate.Request.URL, nil)
	if err != nil {
		return failuref("validation", err, "uploader %q: invalid request method or URL", name)
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" || request.URL.Host == "" {
		return failuref("validation", nil, "uploader %q: request URL must be absolute HTTP(S)", name)
	}
	if err := validateUploaderBody(name, candidate.Request); err != nil {
		return err
	}
	seenHeaders := make(map[string]string, len(candidate.Request.Headers))
	for key, value := range candidate.Request.Headers {
		canonicalName := strings.ToLower(key)
		if previous, ok := seenHeaders[canonicalName]; ok {
			return failuref("validation", nil, "uploader %q: duplicate header names %q and %q", name, previous, key)
		}
		seenHeaders[canonicalName] = key
		if !validHeaderName(key) || strings.ContainsAny(value, "\r\n") {
			return failuref("validation", nil, "uploader %q: invalid header %q", name, key)
		}
		if isManagedRequestHeader(key) {
			return failuref("validation", nil, "uploader %q: header %q is managed by the HTTP transport", name, key)
		}
		if strings.EqualFold(key, "Content-Type") {
			if candidate.Request.Body != "binary" {
				return failuref("validation", nil, "uploader %q: Content-Type is owned by Upit for %s requests", name, candidate.Request.Body)
			}
			if strings.TrimSpace(value) == "" {
				return failuref("validation", nil, "uploader %q: binary Content-Type must not be empty", name)
			}
			if _, _, err := mime.ParseMediaType(value); err != nil {
				return failuref("validation", err, "uploader %q: invalid binary Content-Type", name)
			}
		}
	}
	for key := range candidate.Request.Query {
		if key == "" {
			return failuref("validation", nil, "uploader %q: query keys must not be empty", name)
		}
	}
	for key := range candidate.Request.Fields {
		if key == "" {
			return failuref("validation", nil, "uploader %q: field names must not be empty", name)
		}
	}
	if err := validateUploaderExtractor(name, "response URL", &candidate.Response.URL); err != nil {
		return err
	}
	if candidate.Response.Error != nil {
		if err := validateUploaderExtractor(name, "response error", candidate.Response.Error); err != nil {
			return err
		}
	}
	candidate.sensitiveValues = collectSensitiveValues(candidate.Request)
	return nil
}

func validateUploaderBody(name string, request requestConfig) error {
	switch request.Body {
	case "multipart":
		if request.FileField == "" {
			return failuref("validation", nil, "uploader %q: fileField is required for multipart body", name)
		}
		if request.hasData {
			return failuref("validation", nil, "uploader %q: data is not valid for multipart body", name)
		}
	case "binary":
		if request.hasFileField || request.hasFields || request.hasData {
			return failuref("validation", nil, "uploader %q: binary body does not accept fileField, fields, or data", name)
		}
	case "form":
		if request.hasFileField || request.hasData {
			return failuref("validation", nil, "uploader %q: form body accepts fields only", name)
		}
		if !request.hasFields {
			return failuref("validation", nil, "uploader %q: fields are required for form body", name)
		}
		if countStringPlaceholders(request.Fields) != 1 {
			return failuref("validation", nil, "uploader %q: form fields must contain exactly one string value equal to {input}", name)
		}
	case "json":
		if request.hasFileField || request.hasFields {
			return failuref("validation", nil, "uploader %q: JSON body accepts data only", name)
		}
		if !request.hasData {
			return failuref("validation", nil, "uploader %q: data is required for JSON body", name)
		}
		data, ok := request.Data.(map[string]any)
		if !ok {
			return failuref("validation", nil, "uploader %q: JSON data must be an object", name)
		}
		if countInputPlaceholders(data) != 1 {
			return failuref("validation", nil, "uploader %q: JSON data must contain exactly one string value equal to {input}", name)
		}
	default:
		return failuref("validation", nil, "uploader %q: body must be one of multipart, binary, form, or json", name)
	}
	return nil
}

func validateShortener(name string, candidate *shortener) error {
	if candidate.Request.Method == "" {
		return failuref("validation", nil, "shortener %q: request method is required", name)
	}
	request, err := http.NewRequest(candidate.Request.Method, candidate.Request.URL, nil)
	if err != nil {
		return failuref("validation", err, "shortener %q: invalid request method or URL", name)
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" || request.URL.Host == "" {
		return failuref("validation", nil, "shortener %q: request URL must be absolute HTTP(S)", name)
	}
	for key, value := range candidate.Request.Headers {
		if !validHeaderName(key) || strings.ContainsAny(value, "\r\n") {
			return failuref("validation", nil, "shortener %q: invalid header %q", name, key)
		}
		if strings.EqualFold(key, "Content-Type") {
			return failuref("validation", nil, "shortener %q: Content-Type is owned by Upit for JSON requests", name)
		}
	}
	for key := range candidate.Request.Query {
		if key == "" {
			return failuref("validation", nil, "shortener %q: query keys must not be empty", name)
		}
	}
	if placeholders := countInputPlaceholders(candidate.Request.Data); placeholders != 1 {
		return failuref("validation", nil, "shortener %q: data must contain exactly one string value equal to {input}; found %d", name, placeholders)
	}
	if err := compileExtractor("shortener", name, "response URL", &candidate.Response.URL); err != nil {
		return err
	}
	if candidate.Response.Error != nil {
		if err := compileExtractor("shortener", name, "response error", candidate.Response.Error); err != nil {
			return err
		}
	}
	candidate.sensitiveValues = collectShortenerSensitiveValues(candidate.Request)
	return nil
}

func validateUploaderExtractor(name, label string, extractor *extractorConfig) error {
	switch extractor.Type {
	case "json":
		if !extractor.hasPath || extractor.Path == "" {
			return failuref("validation", nil, "uploader %q: %s path is required", name, label)
		}
		if extractor.hasHeader || extractor.hasPattern || extractor.hasGroup {
			return failuref("validation", nil, "uploader %q: %s has fields incompatible with json extraction", name, label)
		}
		path, err := jsonpath.Parse(extractor.Path)
		if err != nil {
			return failuref("validation", err, "uploader %q: invalid %s path", name, label)
		}
		extractor.compiled = path
	case "header":
		if !extractor.hasHeader || !validHeaderName(extractor.Header) {
			return failuref("validation", nil, "uploader %q: %s header name is invalid", name, label)
		}
		if extractor.hasPath || extractor.hasPattern || extractor.hasGroup {
			return failuref("validation", nil, "uploader %q: %s has fields incompatible with header extraction", name, label)
		}
	case "regex":
		if !extractor.hasPattern || extractor.Pattern == "" {
			return failuref("validation", nil, "uploader %q: %s regex pattern is required", name, label)
		}
		if extractor.hasPath || extractor.hasHeader {
			return failuref("validation", nil, "uploader %q: %s has fields incompatible with regex extraction", name, label)
		}
		compiled, err := regexp.Compile(extractor.Pattern)
		if err != nil {
			return failuref("validation", err, "uploader %q: invalid %s regex pattern", name, label)
		}
		group := extractor.Group
		if !extractor.hasGroup {
			extractor.groupIndex = 0
		} else if group == "" {
			return failuref("validation", nil, "uploader %q: %s regex group must not be empty", name, label)
		} else if index, err := strconv.Atoi(group); err == nil {
			if index < 0 || index > compiled.NumSubexp() {
				return failuref("validation", nil, "uploader %q: %s regex group %q does not exist", name, label, group)
			}
			extractor.groupIndex = index
		} else {
			index := compiled.SubexpIndex(group)
			if index < 0 {
				return failuref("validation", nil, "uploader %q: %s regex group %q does not exist", name, label, group)
			}
			extractor.groupIndex = index
		}
		extractor.regex = compiled
	case "body":
		if extractor.hasPath || extractor.hasHeader || extractor.hasPattern || extractor.hasGroup {
			return failuref("validation", nil, "uploader %q: %s body extractor does not accept extra fields", name, label)
		}
	default:
		return failuref("validation", nil, "uploader %q: %s type must be json, header, regex, or body", name, label)
	}
	return nil
}

func compileExtractor(kind, name, label string, extractor *extractorConfig) error {
	if extractor.Type != "json" {
		return failuref("validation", nil, "%s %q: %s type must be json", kind, name, label)
	}
	if !extractor.hasPath || extractor.Path == "" || extractor.hasHeader || extractor.hasPattern || extractor.hasGroup {
		return failuref("validation", nil, "%s %q: invalid %s JSON extractor fields", kind, name, label)
	}
	path, err := jsonpath.Parse(extractor.Path)
	if err != nil {
		return failuref("validation", err, "%s %q: invalid %s path", kind, name, label)
	}
	extractor.compiled = path
	return nil
}

func countStringPlaceholders(values map[string]string) int {
	count := 0
	for _, value := range values {
		if value == inputPlaceholder {
			count++
		}
	}
	return count
}

func collectSensitiveValues(request requestConfig) []string {
	values := make([]string, 0, len(request.Headers)+len(request.Query)+len(request.Fields))
	appendValue := func(value string) {
		value = normalizeProviderMessage(value)
		if value != "" && value != inputPlaceholder {
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
	collectJSONStringValues(request.Data, appendValue)
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

func collectShortenerSensitiveValues(request shortenerRequestConfig) []string {
	values := make([]string, 0, len(request.Headers)+len(request.Query))
	appendValue := func(value string) {
		value = normalizeProviderMessage(value)
		if value != "" && value != inputPlaceholder {
			values = append(values, value)
		}
	}
	for value := range maps.Values(request.Headers) {
		appendValue(value)
	}
	for value := range maps.Values(request.Query) {
		appendValue(value)
	}
	collectJSONStringValues(request.Data, appendValue)
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
		if r > unicode.MaxASCII || r <= ' ' || r == unicode.MaxASCII || strings.ContainsRune("()<>@,;:\\\"/[]?={}", r) {
			return false
		}
	}
	return true
}

func isManagedRequestHeader(name string) bool {
	switch {
	case strings.EqualFold(name, "Host"):
		return true
	case strings.EqualFold(name, "Content-Length"):
		return true
	case strings.EqualFold(name, "Transfer-Encoding"):
		return true
	case strings.EqualFold(name, "Connection"):
		return true
	case strings.EqualFold(name, "Trailer"):
		return true
	case strings.EqualFold(name, "Upgrade"):
		return true
	case strings.EqualFold(name, "Proxy-Connection"):
		return true
	default:
		return false
	}
}
