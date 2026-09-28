package app

import (
	"fmt"

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
	Version         int    `json:"version"`
	DefaultUploader string `json:"defaultUploader"`
	CopyToClipboard bool   `json:"copyToClipboard"`
}

type uploaderDocument struct {
	Version   int                 `json:"version"`
	Uploaders map[string]uploader `json:"uploaders"`
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
}

type responseConfig struct {
	URL   extractorConfig  `json:"url"`
	Error *extractorConfig `json:"error"`
}

type extractorConfig struct {
	Type     string `json:"type"`
	Path     string `json:"path"`
	compiled *jsonpath.Path
}
