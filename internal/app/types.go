package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Version          int    `json:"version"`
	DefaultUploader  string `json:"defaultUploader"`
	DefaultShortener string `json:"defaultShortener"`
	CopyToClipboard  bool   `json:"copyToClipboard"`
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
	sensitiveValues []string
}

type shortenerRequestConfig struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Data    map[string]any    `json:"data"`
}

type uploader struct {
	Request         requestConfig  `json:"request"`
	Response        responseConfig `json:"response"`
	sensitiveValues []string
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
	*r = requestConfig(decoded)
	_, r.hasFileField = fields["fileField"]
	_, r.hasFields = fields["fields"]
	_, r.hasData = fields["data"]
	return nil
}

type responseConfig struct {
	URL   extractorConfig  `json:"url"`
	Error *extractorConfig `json:"error"`
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

func decodeStrictObject(data []byte, target any) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("must be a JSON object")
	}
	return fields, nil
}
