package app

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
)

type ShortenerEditorDraft struct {
	Revision     string                `json:"revision"`
	OriginalName string                `json:"originalName"`
	Name         string                `json:"name"`
	Request      ShortenerRequestDraft `json:"request"`
	Response     UploaderResponseDraft `json:"response"`
}

type ShortenerRequestDraft struct {
	Method   string             `json:"method"`
	URL      string             `json:"url"`
	Headers  []UploaderMapEntry `json:"headers"`
	Query    []UploaderMapEntry `json:"query"`
	DataJSON string             `json:"dataJSON"`
}

type ShortenerEditorState struct {
	ConfigurationPath string               `json:"configurationPath"`
	Present           bool                 `json:"present"`
	Revision          string               `json:"revision"`
	DefaultShortener  string               `json:"defaultShortener"`
	Shorteners        []string             `json:"shorteners"`
	Draft             ShortenerEditorDraft `json:"draft"`
}

type ShortenerRenameDraft struct {
	Revision     string `json:"revision"`
	OriginalName string `json:"originalName"`
	NewName      string `json:"newName"`
}

type ShortenerDeleteDraft struct {
	Revision string `json:"revision"`
	Name     string `json:"name"`
}

func (s Service) LoadShortenerEditor(name string) (ShortenerEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return ShortenerEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	path := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	data, err := os.ReadFile(path)
	present := true
	if os.IsNotExist(err) {
		present = false
		data = nil
	} else if err != nil {
		return ShortenerEditorState{}, failuref("config", err, "read Shortener document %s: %v", absolutePath(path), err)
	}
	var document shortenerDocument
	if present {
		document, err = loadShortenerConfiguration(home)
		if err != nil {
			return ShortenerEditorState{}, err
		}
	}
	candidate := shortener{}
	if name != "" {
		var ok bool
		candidate, ok = document.Shorteners[name]
		if !ok {
			return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "Shortener %q does not exist; choose one listed by config list-shorteners", name)
		}
	} else {
		candidate = shortener{hasRequest: true, hasResponse: true, Request: shortenerRequestConfig{Method: "POST"}, Response: responseConfig{hasURL: true, URL: extractorConfig{Type: "json"}}}
	}
	revision := configurationRevision(data)
	draft := shortenerEditorDraftFrom(name, candidate)
	draft.Revision = revision
	return ShortenerEditorState{
		ConfigurationPath: absolutePath(path),
		Present:           present,
		Revision:          revision,
		DefaultShortener:  global.DefaultShortener,
		Shorteners:        slices.Sorted(maps.Keys(document.Shorteners)),
		Draft:             draft,
	}, nil
}

func (s Service) SaveShortenerEditor(draft ShortenerEditorDraft) (ShortenerEditorState, error) {
	home, err := s.homeDirectory()
	if err != nil {
		return ShortenerEditorState{}, failuref("config", err, "resolve user home directory: %v", err)
	}
	path := filepath.Join(home, ".config", "upit", "custom-shortener.json")
	data, err := os.ReadFile(path)
	present := true
	if os.IsNotExist(err) {
		present = false
		data = nil
	} else if err != nil {
		return ShortenerEditorState{}, failuref("config", err, "read Shortener document %s: %v", absolutePath(path), err)
	}
	if configurationRevision(data) != draft.Revision {
		return ShortenerEditorState{}, failure("config", "Shortener document changed on disk; refresh before saving", nil)
	}
	var info os.FileInfo
	if present {
		info, err = refuseUnsafeDocumentTarget(path, "Shortener document")
		if err != nil {
			return ShortenerEditorState{}, err
		}
	}
	document := shortenerDocument{Version: 1, Shorteners: map[string]shortener{}}
	if present {
		document, err = loadShortenerConfiguration(home)
		if err != nil {
			return ShortenerEditorState{}, err
		}
	}
	if draft.Name == "" || !validDefinitionName(draft.Name) {
		return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "Shortener name must be non-empty printable Unicode without leading or trailing Unicode whitespace")
	}
	if draft.OriginalName != "" && draft.OriginalName != draft.Name {
		return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "renaming Shorteners is not part of this editor operation")
	}
	if draft.OriginalName == "" {
		if _, exists := document.Shorteners[draft.Name]; exists {
			return ShortenerEditorState{}, configLocation{file: absolutePath(path), path: "$.shorteners"}.fail(nil, "Shortener %q already exists", draft.Name)
		}
	}
	var existing *shortener
	if draft.OriginalName != "" {
		candidate, ok := document.Shorteners[draft.OriginalName]
		if !ok {
			return ShortenerEditorState{}, failure("config", "selected Shortener no longer exists; refresh before saving", nil)
		}
		existing = &candidate
	}
	candidate, err := shortenerFromEditorDraft(draft, existing)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	document.Shorteners[draft.Name] = candidate
	if err := validateShortenerDocument(&document, path); err != nil {
		return ShortenerEditorState{}, err
	}
	global, err := loadGlobalConfiguration(home)
	if err != nil {
		return ShortenerEditorState{}, err
	}
	if err := validateShortenerSetReferences(home, global, document); err != nil {
		return ShortenerEditorState{}, err
	}
	canonical, err := json.MarshalIndent(document, "", "    ")
	if err != nil {
		return ShortenerEditorState{}, failuref("config", err, "encode Shortener document: %v", err)
	}
	canonical = append(canonical, '\n')
	if err := publishDocument(path, canonical, info, "Shortener document"); err != nil {
		return ShortenerEditorState{}, err
	}
	return s.LoadShortenerEditor(draft.Name)
}

func validateShortenerDocument(document *shortenerDocument, path string) error {
	location := configLocation{file: absolutePath(path), path: "$.shorteners"}
	if len(document.Shorteners) == 0 {
		return location.fail(nil, "at least one Shortener is required; add a named Shortener")
	}
	for _, name := range slices.Sorted(maps.Keys(document.Shorteners)) {
		candidate := document.Shorteners[name]
		if err := validateShortener(location, name, &candidate); err != nil {
			return err
		}
		document.Shorteners[name] = candidate
	}
	return nil
}

func shortenerEditorDraftFrom(name string, candidate shortener) ShortenerEditorDraft {
	return ShortenerEditorDraft{
		OriginalName: name,
		Name:         name,
		Request: ShortenerRequestDraft{
			Method:   candidate.Request.Method,
			URL:      candidate.Request.URL,
			Headers:  mapEntries(candidate.Request.Headers, candidate.sensitiveValues),
			Query:    mapEntries(candidate.Request.Query, candidate.sensitiveValues),
			DataJSON: redactDataJSON(candidate.Request.Data, candidate.sensitiveValues),
		},
		Response: UploaderResponseDraft{URL: extractorEditorDraftFrom(candidate.Response.URL), Error: extractorPointerEditorDraftFrom(candidate.Response.Error)},
	}
}

func shortenerFromEditorDraft(draft ShortenerEditorDraft, existing *shortener) (shortener, error) {
	var existingRequest shortenerRequestConfig
	var sensitive []string
	if existing != nil {
		existingRequest = existing.Request
		sensitive = existing.sensitiveValues
	}
	request := shortenerRequestConfig{Method: draft.Request.Method, URL: draft.Request.URL}
	var err error
	request.Headers, err = restoreMapEntries(draft.Request.Headers, existingRequest.Headers, sensitive)
	if err != nil {
		return shortener{}, err
	}
	request.Query, err = restoreMapEntries(draft.Request.Query, existingRequest.Query, sensitive)
	if err != nil {
		return shortener{}, err
	}
	if draft.Request.DataJSON != "" {
		var decoded any
		if _, err := decodeJSON([]byte(draft.Request.DataJSON), &decoded); err != nil {
			return shortener{}, failuref("validation", err, "decode Shortener request data: %v", err)
		}
		decoded = restoreRedactedJSON(decoded, existingRequest.Data, sensitive)
		object, ok := decoded.(map[string]any)
		if !ok {
			return shortener{}, failure("validation", "Shortener request data must be an object", nil)
		}
		request.Data = object
	}
	candidate := shortener{Request: request, Response: responseConfig{URL: extractorConfigFromDraft(draft.Response.URL), hasURL: true}, hasRequest: true, hasResponse: true}
	if draft.Response.Error != nil {
		errorExtractor := extractorConfigFromDraft(*draft.Response.Error)
		candidate.Response.Error = &errorExtractor
	}
	return candidate, nil
}
