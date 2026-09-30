package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/theory/jsonpath"
)

type Result struct {
	OriginalURL string
	FinalURL    string
	StatusCode  int
}

type Failure struct {
	Stage      string
	StatusCode int
	Message    string
	Cause      error
}

func (e *Failure) Error() string {
	return e.Message
}

func (e *Failure) Unwrap() error {
	return e.Cause
}

func failure(stage, message string, cause error) *Failure {
	return &Failure{Stage: stage, Message: message, Cause: cause}
}

func failuref(stage string, cause error, format string, args ...any) *Failure {
	return failure(stage, fmt.Sprintf(format, args...), cause)
}

type settings struct {
	Version             int    `json:"version"`
	DefaultUploader     string `json:"defaultUploader"`
	DefaultShortener    string `json:"defaultShortener"`
	CopyToClipboard     bool   `json:"copyToClipboard"`
	hasVersion          bool
	hasDefaultUploader  bool
	hasDefaultShortener bool
	hasCopyToClipboard  bool
}

func (s *settings) UnmarshalJSON(data []byte) error {
	type plain settings
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	if err := rejectNullFields(fields, "version", "defaultUploader", "defaultShortener", "copyToClipboard"); err != nil {
		return err
	}
	*s = settings(decoded)
	_, s.hasVersion = fields["version"]
	_, s.hasDefaultUploader = fields["defaultUploader"]
	_, s.hasDefaultShortener = fields["defaultShortener"]
	_, s.hasCopyToClipboard = fields["copyToClipboard"]
	return nil
}

type uploaderDocument struct {
	Version   int                 `json:"version"`
	Uploaders map[string]uploader `json:"uploaders"`
}

type shortenerDocument struct {
	Version    int                  `json:"version"`
	Shorteners map[string]shortener `json:"shorteners"`
}

type shortener struct {
	Request         shortenerRequestConfig `json:"request"`
	Response        responseConfig         `json:"response"`
	hasRequest      bool
	hasResponse     bool
	sensitiveValues []string
}

func (s *shortener) UnmarshalJSON(data []byte) error {
	type plain shortener
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	*s = shortener(decoded)
	_, s.hasRequest = fields["request"]
	_, s.hasResponse = fields["response"]
	return nil
}

type shortenerRequestConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Data    map[string]any    `json:"data"`
}

func (s *shortenerRequestConfig) UnmarshalJSON(data []byte) error {
	type plain shortenerRequestConfig
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	if err := rejectNullFields(fields, "method", "url", "headers", "query", "data"); err != nil {
		return err
	}
	*s = shortenerRequestConfig(decoded)
	return nil
}

type uploader struct {
	Request         requestConfig  `json:"request"`
	Response        responseConfig `json:"response"`
	hasRequest      bool
	hasResponse     bool
	sensitiveValues []string
}

func (u *uploader) UnmarshalJSON(data []byte) error {
	type plain uploader
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	*u = uploader(decoded)
	_, u.hasRequest = fields["request"]
	_, u.hasResponse = fields["response"]
	return nil
}

type requestConfig struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Query     map[string]string `json:"query"`
	Body      string            `json:"body"`
	FileField string            `json:"fileField"`
	Fields    map[string]string `json:"fields"`
	Data      any               `json:"data"`

	hasFileField bool
	hasFields    bool
	hasData      bool
}

func (r *requestConfig) UnmarshalJSON(data []byte) error {
	type plain requestConfig
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	if err := rejectNullFields(fields, "method", "url", "headers", "query", "body", "fileField", "fields", "data"); err != nil {
		return err
	}
	*r = requestConfig(decoded)
	_, r.hasFileField = fields["fileField"]
	_, r.hasFields = fields["fields"]
	_, r.hasData = fields["data"]
	return nil
}

type responseConfig struct {
	URL    extractorConfig  `json:"url"`
	Error  *extractorConfig `json:"error"`
	hasURL bool
}

func (r *responseConfig) UnmarshalJSON(data []byte) error {
	type plain responseConfig
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	if err := rejectNullFields(fields, "url", "error"); err != nil {
		return err
	}
	*r = responseConfig(decoded)
	_, r.hasURL = fields["url"]
	return nil
}

type extractorConfig struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Header  string `json:"header"`
	Pattern string `json:"pattern"`
	Group   string `json:"group"`

	hasPath    bool
	hasHeader  bool
	hasPattern bool
	hasGroup   bool
	compiled   *jsonpath.Path
	regex      *regexp.Regexp
	groupIndex int
}

func (e *extractorConfig) UnmarshalJSON(data []byte) error {
	type plain extractorConfig
	var decoded plain
	fields, err := decodeStrictObject(data, &decoded)
	if err != nil {
		return err
	}
	if err := rejectNullFields(fields, "type", "path", "header", "pattern"); err != nil {
		return err
	}
	*e = extractorConfig(decoded)
	_, e.hasPath = fields["path"]
	_, e.hasHeader = fields["header"]
	_, e.hasPattern = fields["pattern"]
	_, e.hasGroup = fields["group"]
	if raw, ok := fields["group"]; ok {
		var group any
		if err := json.Unmarshal(raw, &group); err != nil {
			return err
		}
		if _, ok := group.(string); !ok {
			return fmt.Errorf("regex group must be a string")
		}
	}
	return nil
}

func rejectNullFields(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		raw, ok := fields[name]
		if ok && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return fmt.Errorf("%s must not be null", name)
		}
	}
	return nil
}

func decodeStrictObject(data []byte, target any) (map[string]json.RawMessage, error) {
	_, err := decodeJSON(data, target)
	if err != nil {
		return nil, err
	}
	var rawFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawFields); err != nil {
		return nil, err
	}
	if rawFields == nil {
		return nil, fmt.Errorf("must be a JSON object")
	}
	return rawFields, nil
}
