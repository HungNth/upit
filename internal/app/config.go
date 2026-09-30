package app

import (
	"cmp"
	"errors"
	"fmt"
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
	global, err := loadGlobalConfiguration(homeDir)
	if err != nil {
		return settings{}, uploaderDocument{}, err
	}
	globalPath := filepath.Join(homeDir, ".config", "upit", "config.json")
	if err := validateGlobalStructure(global, configLocation{file: absolutePath(globalPath), path: "$"}); err != nil {
		return settings{}, uploaderDocument{}, err
	}
	uploaders, err := loadUploaderConfiguration(homeDir)
	if err != nil {
		return settings{}, uploaderDocument{}, err
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
		return shortenerDocument{}, configurationReadFailure(path, err, "copy examples/custom-shortener.example.json to this location")
	}
	shortenerLocation := configLocation{file: absolutePath(path), path: "$.shorteners"}
	if shorteners.Version != 1 {
		return shortenerDocument{}, configLocation{file: absolutePath(path), path: "$.version"}.fail(nil, "custom shortener version = %d, want 1; received version %d, expected version 1; automatic migration is unavailable; update version to 1 manually; see schemas/custom-shortener.schema.json", shorteners.Version, shorteners.Version)
	}
	if len(shorteners.Shorteners) == 0 {
		return shortenerDocument{}, shortenerLocation.fail(nil, "at least one Shortener is required; add a named Shortener")
	}
	for _, name := range slices.Sorted(maps.Keys(shorteners.Shorteners)) {
		candidate := shorteners.Shorteners[name]
		if err := validateShortener(shortenerLocation, name, &candidate); err != nil {
			return shortenerDocument{}, err
		}
		shorteners.Shorteners[name] = candidate
	}
	return shorteners, nil
}

func absolutePath(path string) string {
	if absolute, err := filepath.Abs(path); err == nil {
		return absolute
	}
	return filepath.Clean(path)
}

type configLocation struct {
	file string
	path string
}

func (l configLocation) child(key string) configLocation {
	return configLocation{file: l.file, path: jsonPathKey(l.path, key)}
}

func (l configLocation) fail(cause error, format string, args ...any) *Failure {
	return failuref("validation", cause, "%s %s: %s", l.file, l.path, fmt.Sprintf(format, args...))
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if _, err := decodeJSON(data, target); err != nil {
		var decodeErr *jsonDecodeError
		if errors.As(err, &decodeErr) {
			if absolute, absErr := filepath.Abs(path); absErr == nil {
				decodeErr.file = absolute
			}
		}
		return err
	}
	return nil
}

func configurationReadFailure(path string, err error, hint string) *Failure {
	var decodeErr *jsonDecodeError
	if errors.As(err, &decodeErr) {
		return failure("config", decodeErr.Error(), err)
	}
	return failuref("config", err, "load %s: %v; %s", path, err, hint)
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

func validateUploader(documentLocation configLocation, name string, candidate *uploader) error {
	definitionLocation := documentLocation.child(name)
	if !validDefinitionName(name) {
		return definitionFailure(definitionLocation, "uploader", name, nil, "name must be non-empty printable Unicode without leading or trailing Unicode whitespace")
	}
	if !candidate.hasRequest {
		return definitionFailure(definitionLocation.child("request"), "uploader", name, nil, "request is required")
	}
	if !candidate.hasResponse {
		return definitionFailure(definitionLocation.child("response"), "uploader", name, nil, "response is required")
	}
	if !candidate.Response.hasURL {
		return definitionFailure(definitionLocation.child("response").child("url"), "uploader", name, nil, "response URL extractor is required")
	}

	requestLocation := definitionLocation.child("request")
	if candidate.Request.Method == "" {
		return definitionFailure(requestLocation.child("method"), "uploader", name, nil, "request method is required")
	}
	request, err := http.NewRequest(candidate.Request.Method, candidate.Request.URL, nil)
	if err != nil {
		return definitionFailure(requestLocation, "uploader", name, err, "request method or URL is invalid")
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" || request.URL.Host == "" {
		return definitionFailure(requestLocation.child("url"), "uploader", name, nil, "request URL must be absolute HTTP(S)")
	}
	if err := validateUploaderBody(requestLocation, name, candidate.Request); err != nil {
		return err
	}
	seenHeaders := make(map[string]string, len(candidate.Request.Headers))
	for _, key := range slices.Sorted(maps.Keys(candidate.Request.Headers)) {
		value := candidate.Request.Headers[key]
		canonicalName := strings.ToLower(key)
		if previous, ok := seenHeaders[canonicalName]; ok {
			return definitionFailure(requestLocation.child("headers"), "uploader", name, nil, "duplicate header names %q and %q", previous, key)
		}
		seenHeaders[canonicalName] = key
		if !validHeaderName(key) || strings.ContainsAny(value, "\r\n") {
			return definitionFailure(requestLocation.child("headers"), "uploader", name, nil, "invalid header %q", key)
		}
		if isManagedRequestHeader(key) {
			return definitionFailure(requestLocation.child("headers"), "uploader", name, nil, "header %q is managed by the HTTP transport", key)
		}
		if strings.EqualFold(key, "Content-Type") {
			if candidate.Request.Body != "binary" {
				return definitionFailure(requestLocation.child("headers"), "uploader", name, nil, "Content-Type is owned by Upit for %s requests", candidate.Request.Body)
			}
			if strings.TrimSpace(value) == "" {
				return definitionFailure(requestLocation.child("headers"), "uploader", name, nil, "binary Content-Type must not be empty")
			}
			if _, _, err := mime.ParseMediaType(value); err != nil {
				return definitionFailure(requestLocation.child("headers"), "uploader", name, err, "invalid binary Content-Type")
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(candidate.Request.Query)) {
		if key == "" {
			return definitionFailure(requestLocation.child("query"), "uploader", name, nil, "query keys must not be empty")
		}
	}
	for _, key := range slices.Sorted(maps.Keys(candidate.Request.Fields)) {
		if key == "" {
			return definitionFailure(requestLocation.child("fields"), "uploader", name, nil, "field names must not be empty")
		}
	}
	responseLocation := definitionLocation.child("response")
	if err := validateExtractor(responseLocation.child("url"), "uploader", name, "response URL", &candidate.Response.URL); err != nil {
		return err
	}
	if candidate.Response.Error != nil {
		if err := validateExtractor(responseLocation.child("error"), "uploader", name, "response error", candidate.Response.Error); err != nil {
			return err
		}
	}
	candidate.sensitiveValues = collectSensitiveValues(candidate.Request)
	return nil
}

func validateUploaderBody(location configLocation, name string, request requestConfig) error {
	switch request.Body {
	case "multipart":
		if request.FileField == "" {
			return definitionFailure(location.child("fileField"), "uploader", name, nil, "fileField is required for multipart body")
		}
		if request.hasData {
			return definitionFailure(location.child("data"), "uploader", name, nil, "data is not valid for multipart body")
		}
	case "binary":
		if request.hasFileField || request.hasFields || request.hasData {
			return definitionFailure(location, "uploader", name, nil, "binary body does not accept fileField, fields, or data")
		}
	case "form":
		if request.hasFileField || request.hasData {
			return definitionFailure(location, "uploader", name, nil, "form body accepts fields only")
		}
		if !request.hasFields {
			return definitionFailure(location.child("fields"), "uploader", name, nil, "fields are required for form body")
		}
		if countStringPlaceholders(request.Fields) != 1 {
			return definitionFailure(location.child("fields"), "uploader", name, nil, "form fields must contain exactly one string value equal to {input}")
		}
	case "json":
		if request.hasFileField || request.hasFields {
			return definitionFailure(location, "uploader", name, nil, "JSON body accepts data only")
		}
		if !request.hasData {
			return definitionFailure(location.child("data"), "uploader", name, nil, "data is required for JSON body")
		}
		data, ok := request.Data.(map[string]any)
		if !ok {
			return definitionFailure(location.child("data"), "uploader", name, nil, "JSON data must be an object")
		}
		if countInputPlaceholders(data) != 1 {
			return definitionFailure(location.child("data"), "uploader", name, nil, "JSON data must contain exactly one string value equal to {input}")
		}
	default:
		return definitionFailure(location.child("body"), "uploader", name, nil, "body must be one of multipart, binary, form, or json")
	}
	return nil
}

func validateShortener(documentLocation configLocation, name string, candidate *shortener) error {
	definitionLocation := documentLocation.child(name)
	if !validDefinitionName(name) {
		return definitionFailure(definitionLocation, "shortener", name, nil, "name must be non-empty printable Unicode without leading or trailing Unicode whitespace")
	}
	if !candidate.hasRequest {
		return definitionFailure(definitionLocation.child("request"), "shortener", name, nil, "request is required")
	}
	if !candidate.hasResponse {
		return definitionFailure(definitionLocation.child("response"), "shortener", name, nil, "response is required")
	}
	if !candidate.Response.hasURL {
		return definitionFailure(definitionLocation.child("response").child("url"), "shortener", name, nil, "response URL extractor is required")
	}

	requestLocation := definitionLocation.child("request")
	if candidate.Request.Method == "" {
		return definitionFailure(requestLocation.child("method"), "shortener", name, nil, "request method is required")
	}
	request, err := http.NewRequest(candidate.Request.Method, candidate.Request.URL, nil)
	if err != nil {
		return definitionFailure(requestLocation, "shortener", name, err, "request method or URL is invalid")
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" || request.URL.Host == "" {
		return definitionFailure(requestLocation.child("url"), "shortener", name, nil, "request URL must be absolute HTTP(S)")
	}
	seenHeaders := make(map[string]string, len(candidate.Request.Headers))
	for _, key := range slices.Sorted(maps.Keys(candidate.Request.Headers)) {
		value := candidate.Request.Headers[key]
		canonicalName := strings.ToLower(key)
		if previous, ok := seenHeaders[canonicalName]; ok {
			return definitionFailure(requestLocation.child("headers"), "shortener", name, nil, "duplicate header names %q and %q", previous, key)
		}
		seenHeaders[canonicalName] = key
		if !validHeaderName(key) || strings.ContainsAny(value, "\r\n") {
			return definitionFailure(requestLocation.child("headers"), "shortener", name, nil, "invalid header %q", key)
		}
		if strings.EqualFold(key, "Content-Type") {
			return definitionFailure(requestLocation.child("headers"), "shortener", name, nil, "Content-Type is owned by Upit for JSON requests")
		}
	}
	for _, key := range slices.Sorted(maps.Keys(candidate.Request.Query)) {
		if key == "" {
			return definitionFailure(requestLocation.child("query"), "shortener", name, nil, "query keys must not be empty")
		}
	}
	if placeholders := countInputPlaceholders(candidate.Request.Data); placeholders != 1 {
		return definitionFailure(requestLocation.child("data"), "shortener", name, nil, "data must contain exactly one string value equal to {input}; found %d", placeholders)
	}
	responseLocation := definitionLocation.child("response")
	if err := validateShortenerExtractor(responseLocation.child("url"), name, "response URL", &candidate.Response.URL); err != nil {
		return err
	}
	if candidate.Response.Error != nil {
		if err := validateShortenerExtractor(responseLocation.child("error"), name, "response error", candidate.Response.Error); err != nil {
			return err
		}
	}
	candidate.sensitiveValues = collectShortenerSensitiveValues(candidate.Request)
	return nil
}

func validateShortenerExtractor(location configLocation, name, label string, extractor *extractorConfig) error {
	if extractor.Type != "json" {
		return definitionFailure(location, "shortener", name, nil, "%s type must be json", label)
	}
	return validateExtractor(location, "shortener", name, label, extractor)
}

func validateExtractor(location configLocation, kind, name, label string, extractor *extractorConfig) error {
	switch extractor.Type {
	case "json":
		if !extractor.hasPath || extractor.Path == "" {
			return definitionFailure(location, kind, name, nil, "%s path is required", label)
		}
		if extractor.hasHeader || extractor.hasPattern || extractor.hasGroup {
			return definitionFailure(location, kind, name, nil, "%s has fields incompatible with json extraction", label)
		}
		path, err := jsonpath.Parse(extractor.Path)
		if err != nil {
			return definitionFailure(location, kind, name, err, "invalid %s path", label)
		}
		extractor.compiled = path
	case "header":
		if !extractor.hasHeader || !validHeaderName(extractor.Header) {
			return definitionFailure(location, kind, name, nil, "%s header name is invalid", label)
		}
		if extractor.hasPath || extractor.hasPattern || extractor.hasGroup {
			return definitionFailure(location, kind, name, nil, "%s has fields incompatible with header extraction", label)
		}
	case "regex":
		if !extractor.hasPattern || extractor.Pattern == "" {
			return definitionFailure(location, kind, name, nil, "%s regex pattern is required", label)
		}
		if extractor.hasPath || extractor.hasHeader {
			return definitionFailure(location, kind, name, nil, "%s has fields incompatible with regex extraction", label)
		}
		compiled, err := regexp.Compile(extractor.Pattern)
		if err != nil {
			return definitionFailure(location, kind, name, err, "invalid %s regex pattern", label)
		}
		group := extractor.Group
		if !extractor.hasGroup {
			extractor.groupIndex = 0
		} else if group == "" {
			return definitionFailure(location, kind, name, nil, "%s regex group must not be empty", label)
		} else if index, err := strconv.Atoi(group); err == nil {
			if index < 0 || index > compiled.NumSubexp() {
				return definitionFailure(location, kind, name, nil, "%s regex group %q does not exist", label, group)
			}
			extractor.groupIndex = index
		} else {
			index := compiled.SubexpIndex(group)
			if index < 0 {
				return definitionFailure(location, kind, name, nil, "%s regex group %q does not exist", label, group)
			}
			extractor.groupIndex = index
		}
		extractor.regex = compiled
	case "body":
		if extractor.hasPath || extractor.hasHeader || extractor.hasPattern || extractor.hasGroup {
			return definitionFailure(location, kind, name, nil, "%s body extractor does not accept extra fields", label)
		}
	default:
		return definitionFailure(location, kind, name, nil, "%s type must be json, header, regex, or body", label)
	}
	return nil
}

func definitionFailure(location configLocation, kind, name string, cause error, format string, args ...any) error {
	values := make([]any, 0, len(args)+2)
	values = append(values, kind, name)
	values = append(values, args...)
	return location.fail(cause, "%s %q: "+format, values...)
}

func validDefinitionName(name string) bool {
	if name == "" {
		return false
	}
	runes := []rune(name)
	for index, r := range runes {
		if !unicode.IsPrint(r) || ((index == 0 || index == len(runes)-1) && unicode.IsSpace(r)) {
			return false
		}
	}
	return true
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
