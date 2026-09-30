package app

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

const redactedConfigurationValue = "••••••"

type UploaderMapEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Sensitive bool   `json:"sensitive"`
}

type UploaderExtractorDraft struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Header  string `json:"header"`
	Pattern string `json:"pattern"`
	Group   string `json:"group"`
}

type UploaderRequestDraft struct {
	Method    string             `json:"method"`
	URL       string             `json:"url"`
	Headers   []UploaderMapEntry `json:"headers"`
	Query     []UploaderMapEntry `json:"query"`
	Body      string             `json:"body"`
	FileField string             `json:"fileField"`
	Fields    []UploaderMapEntry `json:"fields"`
	DataJSON  string             `json:"dataJSON"`
}

type UploaderResponseDraft struct {
	URL   UploaderExtractorDraft  `json:"url"`
	Error *UploaderExtractorDraft `json:"error"`
}

type UploaderEditorDraft struct {
	Revision     string                `json:"revision"`
	OriginalName string                `json:"originalName"`
	Name         string                `json:"name"`
	Request      UploaderRequestDraft  `json:"request"`
	Response     UploaderResponseDraft `json:"response"`
}

type UploaderEditorState struct {
	ConfigurationPath string              `json:"configurationPath"`
	Revision          string              `json:"revision"`
	DefaultUploader   string              `json:"defaultUploader"`
	Uploaders         []string            `json:"uploaders"`
	Draft             UploaderEditorDraft `json:"draft"`
}

func (s Service) LoadUploaderEditor(name string) (UploaderEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return UploaderEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	document, err := loadUploaderConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	path := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return UploaderEditorState{}, failuref("config", err, "read Uploader document %s: %v", absolutePath(path), err)
	}
	candidate := uploader{}
	if name != "" {
		var ok bool
		candidate, ok = document.Uploaders[name]
		if !ok {
			return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "Uploader %q does not exist; choose one listed by config list-uploaders", name)
		}
	} else {
		candidate = newUploaderDraftValue()
	}
	revision := configurationRevision(data)
	draft := uploaderEditorDraftFrom(name, candidate)
	draft.Revision = revision
	return UploaderEditorState{
		ConfigurationPath: absolutePath(path),
		Revision:          revision,
		DefaultUploader:   global.DefaultUploader,
		Uploaders:         slices.Sorted(maps.Keys(document.Uploaders)),
		Draft:             draft,
	}, nil
}

func (s Service) SaveUploaderEditor(draft UploaderEditorDraft) (UploaderEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return UploaderEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-uploader.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return UploaderEditorState{}, failuref("config", err, "read Uploader document %s: %v", absolutePath(path), err)
	}
	if draft.Revision == "" || configurationRevision(data) != draft.Revision {
		return UploaderEditorState{}, failure("config", "Uploader document changed on disk; refresh before saving", nil)
	}
	info, err := refuseUnsafeDocumentTarget(path, "Uploader document")
	if err != nil {
		return UploaderEditorState{}, err
	}
	document, err := loadUploaderConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if draft.Name == "" {
		return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "Uploader name is required")
	}
	if draft.OriginalName != "" && draft.OriginalName != draft.Name {
		return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "renaming Uploaders is not part of this editor operation")
	}
	if draft.OriginalName == "" {
		if _, exists := document.Uploaders[draft.Name]; exists {
			return UploaderEditorState{}, configLocation{file: absolutePath(path), path: "$.uploaders"}.fail(nil, "Uploader %q already exists", draft.Name)
		}
	}
	var existing *uploader
	if draft.OriginalName != "" {
		candidate, ok := document.Uploaders[draft.OriginalName]
		if !ok {
			return UploaderEditorState{}, failure("config", "selected Uploader no longer exists; refresh before saving", nil)
		}
		existing = &candidate
	}
	candidate, err := uploaderFromEditorDraft(draft, existing)
	if err != nil {
		return UploaderEditorState{}, err
	}
	document.Uploaders[draft.Name] = candidate
	if err := validateUploaderDocument(&document, path); err != nil {
		return UploaderEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if _, _, err := validateConfigurationSetWithUploaderDocument(home, global, document); err != nil {
		return UploaderEditorState{}, err
	}
	canonical, err := marshalUploaderDocument(document)
	if err != nil {
		return UploaderEditorState{}, err
	}
	if err := publishDocument(path, canonical, info, "Uploader document"); err != nil {
		return UploaderEditorState{}, err
	}
	return s.LoadUploaderEditor(draft.Name)
}

func validateUploaderDocument(document *uploaderDocument, path string) error {
	location := configLocation{file: absolutePath(path), path: "$.uploaders"}
	if len(document.Uploaders) == 0 {
		return location.fail(nil, "at least one Uploader is required; add a named Uploader")
	}
	for _, name := range slices.Sorted(maps.Keys(document.Uploaders)) {
		candidate := document.Uploaders[name]
		if err := validateUploader(location, name, &candidate); err != nil {
			return err
		}
		document.Uploaders[name] = candidate
	}
	return nil
}

func validateConfigurationSetWithUploaderDocument(homeDir string, global settings, uploaders uploaderDocument) (shortenerDocument, bool, error) {
	globalPath := filepath.Join(homeDir, ".config", "upit", "config.json")
	globalLocation := configLocation{file: absolutePath(globalPath), path: "$"}
	if err := validateGlobalStructure(global, globalLocation); err != nil {
		return shortenerDocument{}, false, err
	}
	if _, ok := uploaders.Uploaders[global.DefaultUploader]; !ok {
		return shortenerDocument{}, false, globalLocation.child("defaultUploader").fail(nil, "default Uploader %q does not exist; choose one listed by config list-uploaders", global.DefaultUploader)
	}
	shorteners, present, err := loadOptionalShortenerConfiguration(homeDir)
	if err != nil {
		return shortenerDocument{}, false, err
	}
	if global.DefaultShortener != "" {
		if !present {
			return shortenerDocument{}, false, globalLocation.child("defaultShortener").fail(nil, "selected default Shortener %q requires custom-shortener.json", global.DefaultShortener)
		}
		if _, ok := shorteners.Shorteners[global.DefaultShortener]; !ok {
			return shortenerDocument{}, false, globalLocation.child("defaultShortener").fail(nil, "default Shortener %q does not exist; choose one listed by config list-shorteners", global.DefaultShortener)
		}
	}
	return shorteners, present, nil
}

func marshalUploaderDocument(document uploaderDocument) ([]byte, error) {
	data, err := json.MarshalIndent(document, "", "    ")
	if err != nil {
		return nil, failuref("config", err, "encode Uploader document: %v", err)
	}
	return append(data, '\n'), nil
}

func uploaderEditorDraftFrom(name string, candidate uploader) UploaderEditorDraft {
	sensitive := candidate.sensitiveValues
	return UploaderEditorDraft{
		OriginalName: name,
		Name:         name,
		Request: UploaderRequestDraft{
			Method:    candidate.Request.Method,
			URL:       candidate.Request.URL,
			Headers:   mapEntries(candidate.Request.Headers, sensitive),
			Query:     mapEntries(candidate.Request.Query, sensitive),
			Body:      candidate.Request.Body,
			FileField: candidate.Request.FileField,
			Fields:    mapEntries(candidate.Request.Fields, sensitive),
			DataJSON:  redactDataJSON(candidate.Request.Data, sensitive),
		},
		Response: UploaderResponseDraft{
			URL:   extractorEditorDraftFrom(candidate.Response.URL),
			Error: extractorPointerEditorDraftFrom(candidate.Response.Error),
		},
	}
}

func newUploaderDraftValue() uploader {
	return uploader{
		hasRequest:  true,
		hasResponse: true,
		Request:     requestConfig{Method: "POST", Body: "multipart", FileField: "file"},
		Response:    responseConfig{hasURL: true, URL: extractorConfig{Type: "body"}},
	}
}

func mapEntries(values map[string]string, sensitive []string) []UploaderMapEntry {
	entries := make([]UploaderMapEntry, 0, len(values))
	for _, key := range slices.Sorted(maps.Keys(values)) {
		value := values[key]
		masked := slices.Contains(sensitive, value)
		if masked {
			value = redactedConfigurationValue
		}
		entries = append(entries, UploaderMapEntry{Key: key, Value: value, Sensitive: masked})
	}
	return entries
}

func extractorEditorDraftFrom(value extractorConfig) UploaderExtractorDraft {
	return UploaderExtractorDraft{Type: value.Type, Path: value.Path, Header: value.Header, Pattern: value.Pattern, Group: value.Group}
}

func extractorPointerEditorDraftFrom(value *extractorConfig) *UploaderExtractorDraft {
	if value == nil {
		return nil
	}
	draft := extractorEditorDraftFrom(*value)
	return &draft
}

func uploaderFromEditorDraft(draft UploaderEditorDraft, existing *uploader) (uploader, error) {
	request := requestConfig{Method: draft.Request.Method, URL: draft.Request.URL, Body: draft.Request.Body, FileField: draft.Request.FileField}
	var existingRequest requestConfig
	var sensitive []string
	if existing != nil {
		existingRequest = existing.Request
		sensitive = existing.sensitiveValues
	}
	var err error
	request.Headers, err = restoreMapEntries(draft.Request.Headers, existingRequest.Headers, sensitive)
	if err != nil {
		return uploader{}, err
	}
	request.Query, err = restoreMapEntries(draft.Request.Query, existingRequest.Query, sensitive)
	if err != nil {
		return uploader{}, err
	}
	request.Fields, err = restoreMapEntries(draft.Request.Fields, existingRequest.Fields, sensitive)
	if err != nil {
		return uploader{}, err
	}
	request.hasFileField = request.FileField != ""
	request.hasFields = len(draft.Request.Fields) > 0
	request.hasData = request.Body == "json" && draft.Request.DataJSON != ""
	if request.hasData {
		var decoded any
		if _, err := decodeJSON([]byte(draft.Request.DataJSON), &decoded); err != nil {
			return uploader{}, failuref("validation", err, "decode Uploader request data: %v", err)
		}
		request.Data = restoreRedactedJSON(decoded, existingRequest.Data, sensitive)
	}
	switch request.Body {
	case "multipart":
		request.Data = nil
		request.hasData = false
	case "binary":
		request.FileField = ""
		request.Fields = nil
		request.Data = nil
		request.hasFileField = false
		request.hasFields = false
		request.hasData = false
	case "form":
		request.FileField = ""
		request.Data = nil
		request.hasFileField = false
		request.hasData = false
	case "json":
		request.FileField = ""
		request.Fields = nil
		request.hasFileField = false
		request.hasFields = false
	}
	candidate := uploader{
		Request:     request,
		Response:    responseConfig{URL: extractorConfigFromDraft(draft.Response.URL), hasURL: true},
		hasRequest:  true,
		hasResponse: true,
	}
	if draft.Response.Error != nil {
		errorExtractor := extractorConfigFromDraft(*draft.Response.Error)
		candidate.Response.Error = &errorExtractor
	}
	return candidate, nil
}

func extractorConfigFromDraft(draft UploaderExtractorDraft) extractorConfig {
	return extractorConfig{Type: draft.Type, Path: draft.Path, Header: draft.Header, Pattern: draft.Pattern, Group: draft.Group, hasPath: draft.Path != "", hasHeader: draft.Header != "", hasPattern: draft.Pattern != "", hasGroup: draft.Group != ""}
}

func restoreMapEntries(entries []UploaderMapEntry, existing map[string]string, sensitive []string) (map[string]string, error) {
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		if _, exists := values[entry.Key]; exists {
			return nil, failuref("validation", nil, "duplicate key %q in Uploader editor", entry.Key)
		}
		value := entry.Value
		if entry.Sensitive && value == redactedConfigurationValue {
			value = existing[entry.Key]
		}
		values[entry.Key] = value
	}
	return values, nil
}

func redactDataJSON(data any, sensitive []string) string {
	if data == nil {
		return ""
	}
	redacted := redactJSONValue(data, sensitive)
	encoded, err := json.MarshalIndent(redacted, "", "    ")
	if err != nil {
		return ""
	}
	return string(encoded)
}

func redactJSONValue(value any, sensitive []string) any {
	switch typed := value.(type) {
	case string:
		if slices.Contains(sensitive, typed) {
			return redactedConfigurationValue
		}
		return typed
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = redactJSONValue(item, sensitive)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = redactJSONValue(item, sensitive)
		}
		return result
	default:
		return value
	}
}

func restoreRedactedJSON(value, existing any, sensitive []string) any {
	switch typed := value.(type) {
	case string:
		if typed == redactedConfigurationValue {
			if old, ok := existing.(string); ok && slices.Contains(sensitive, old) {
				return old
			}
		}
		return typed
	case []any:
		old, _ := existing.([]any)
		result := make([]any, len(typed))
		for index, item := range typed {
			var previous any
			if index < len(old) {
				previous = old[index]
			}
			result[index] = restoreRedactedJSON(item, previous, sensitive)
		}
		return result
	case map[string]any:
		old, _ := existing.(map[string]any)
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = restoreRedactedJSON(item, old[key], sensitive)
		}
		return result
	default:
		return value
	}
}
