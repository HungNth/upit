package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

type jsonField struct {
	key        string
	path       string
	objectPath string
}

type jsonDecodeError struct {
	path    string
	line    int
	column  int
	message string
	cause   error
	file    string
}

func (e *jsonDecodeError) Error() string {
	location := e.path
	if e.line > 0 {
		location = fmt.Sprintf("%s (line %d, column %d)", location, e.line, e.column)
	}
	if e.file != "" {
		return fmt.Sprintf("%s %s: %s", e.file, location, e.message)
	}
	return fmt.Sprintf("%s: %s", location, e.message)
}

func (e *jsonDecodeError) Unwrap() error {
	return e.cause
}

func decodeJSON(data []byte, target any) ([]jsonField, error) {
	fields, err := scanJSON(data)
	if err != nil {
		return nil, err
	}
	if err := validateKnownConfigurationFields(fields, target); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		message := err.Error()
		path := "$"
		var nested *jsonDecodeError
		if errors.As(err, &nested) {
			message = nested.message
			path = nested.path
		}
		if field := unknownFieldName(message); field != "" {
			message = fmt.Sprintf("unknown field %q", field)
			for _, candidate := range fields {
				if candidate.key == field {
					path = candidate.path
					break
				}
			}
		}
		return fields, &jsonDecodeError{path: path, message: message, cause: err}
	}
	return fields, nil
}

func scanJSON(data []byte) ([]jsonField, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	fields := make([]jsonField, 0)
	var walk func(path string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return jsonTokenError(data, path, decoder.InputOffset(), err)
		}
		delim, isDelimiter := token.(json.Delim)
		if !isDelimiter {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return jsonTokenError(data, path, decoder.InputOffset(), err)
				}
				key, ok := keyToken.(string)
				if !ok {
					return jsonTokenError(data, path, decoder.InputOffset(), fmt.Errorf("object member name is not a string"))
				}
				keyPath := jsonPathKey(path, key)
				fields = append(fields, jsonField{key: key, path: keyPath, objectPath: path})
				if _, exists := seen[key]; exists {
					return jsonTokenError(data, path, decoder.InputOffset(), fmt.Errorf("duplicate key %q", key))
				}
				seen[key] = struct{}{}
				if err := walk(keyPath); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil {
				return jsonTokenError(data, path, decoder.InputOffset(), err)
			}
			if closing != json.Delim('}') {
				return jsonTokenError(data, path, decoder.InputOffset(), fmt.Errorf("object is not closed"))
			}
		case '[':
			for index := 0; decoder.More(); index++ {
				if err := walk(fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
			}
			closing, err := decoder.Token()
			if err != nil {
				return jsonTokenError(data, path, decoder.InputOffset(), err)
			}
			if closing != json.Delim(']') {
				return jsonTokenError(data, path, decoder.InputOffset(), fmt.Errorf("array is not closed"))
			}
		case '}', ']':
			return jsonTokenError(data, path, decoder.InputOffset(), fmt.Errorf("unexpected JSON delimiter %q", delim))
		}
		return nil
	}

	if err := walk("$"); err != nil {
		return nil, err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return nil, jsonTokenError(data, "$", decoder.InputOffset(), fmt.Errorf("multiple JSON values"))
		}
		return nil, jsonTokenError(data, "$", decoder.InputOffset(), err)
	} else if token != nil {
		return nil, jsonTokenError(data, "$", decoder.InputOffset(), fmt.Errorf("multiple JSON values"))
	}
	return fields, nil
}

func jsonTokenError(data []byte, path string, offset int64, err error) *jsonDecodeError {
	message := err.Error()
	if err == io.EOF {
		message = "unexpected end of JSON input"
	}
	if syntax, ok := err.(*json.SyntaxError); ok {
		offset = syntax.Offset
	}
	if !strings.HasPrefix(message, "duplicate key ") {
		path = "$"
	}
	line, column := jsonPosition(data, offset)
	return &jsonDecodeError{path: path, line: line, column: column, message: message, cause: err}
}

func jsonPosition(data []byte, offset int64) (int, int) {
	if offset < 1 {
		offset = 1
	}
	if offset > int64(len(data)+1) {
		offset = int64(len(data) + 1)
	}
	line, column := 1, 1
	for index := int64(0); index < offset-1; index++ {
		if data[index] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

func unknownFieldName(message string) string {
	const prefix = `json: unknown field "`
	if !strings.HasPrefix(message, prefix) || !strings.HasSuffix(message, `"`) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(message, prefix), `"`)
}

func jsonPathKey(parent, key string) string {
	if jsonPathUsesDynamicKeys(parent) || strings.Contains(parent, ".data") {
		return parent + "[" + strconv.Quote(key) + "]"
	}
	if key != "" {
		for index, r := range key {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9' || index == 0) && r != '_' {
				return parent + "[" + strconv.Quote(key) + "]"
			}
		}
		return parent + "." + key
	}
	return parent + "[\"\"]"
}

func jsonPathUsesDynamicKeys(path string) bool {
	for _, suffix := range []string{".uploaders", ".shorteners", ".headers", ".query", ".fields"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func validateKnownConfigurationFields(fields []jsonField, target any) error {
	shape := configurationJSONShapeFor(target)
	if shape == configurationJSONUnknown {
		return nil
	}
	for _, field := range fields {
		if !allowedConfigurationField(shape, field.objectPath, field.key) {
			return &jsonDecodeError{path: field.path, message: fmt.Sprintf("unknown field %q", field.key)}
		}
	}
	return nil
}

type configurationJSONShape uint8

const (
	configurationJSONUnknown configurationJSONShape = iota
	configurationJSONSettings
	configurationJSONUploaders
	configurationJSONShorteners
)

func configurationJSONShapeFor(target any) configurationJSONShape {
	switch target.(type) {
	case *settings:
		return configurationJSONSettings
	case *uploaderDocument:
		return configurationJSONUploaders
	case *shortenerDocument:
		return configurationJSONShorteners
	default:
		return configurationJSONUnknown
	}
}

func allowedConfigurationField(shape configurationJSONShape, objectPath, key string) bool {
	switch shape {
	case configurationJSONSettings:
		return objectPath == "$" && oneOf(key, "version", "defaultUploader", "defaultShortener", "copyToClipboard")
	case configurationJSONUploaders:
		return allowedDefinitionField(objectPath, key, "uploaders", true)
	case configurationJSONShorteners:
		return allowedDefinitionField(objectPath, key, "shorteners", false)
	default:
		return true
	}
}

func allowedDefinitionField(objectPath, key, collection string, uploader bool) bool {
	if objectPath == "$" {
		return oneOf(key, "version", collection)
	}
	if objectPath == "$."+collection {
		return true
	}
	if strings.Contains(objectPath, ".data") {
		return true
	}
	prefix := "$." + collection + "["
	if !strings.HasPrefix(objectPath, prefix) {
		return false
	}
	tailIndex := strings.Index(objectPath[len(prefix):], "\"]")
	if tailIndex < 0 {
		return false
	}
	tailIndex += len(prefix)
	tail := objectPath[tailIndex+2:]
	if tail == "" {
		return oneOf(key, "request", "response")
	}
	if strings.HasSuffix(tail, ".headers") || strings.HasSuffix(tail, ".query") || strings.HasSuffix(tail, ".fields") {
		return true
	}
	if strings.HasSuffix(tail, ".request") {
		if uploader {
			return oneOf(key, "method", "url", "headers", "query", "body", "fileField", "fields", "data")
		}
		return oneOf(key, "method", "url", "headers", "query", "data")
	}
	if strings.HasSuffix(tail, ".response") {
		return oneOf(key, "url", "error")
	}
	if strings.HasSuffix(tail, ".response.url") || strings.HasSuffix(tail, ".response.error") {
		return oneOf(key, "type", "path", "header", "pattern", "group")
	}
	return false
}

func oneOf(value string, options ...string) bool {
	return slices.Contains(options, value)
}
