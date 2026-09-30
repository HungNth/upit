package app

import "encoding/json"

func (r requestConfig) MarshalJSON() ([]byte, error) {
	value := struct {
		Method    string            `json:"method"`
		URL       string            `json:"url"`
		Headers   map[string]string `json:"headers,omitempty"`
		Query     map[string]string `json:"query,omitempty"`
		Body      string            `json:"body"`
		FileField string            `json:"fileField,omitempty"`
		Fields    map[string]string `json:"fields,omitempty"`
		Data      any               `json:"data,omitempty"`
	}{
		Method:  r.Method,
		URL:     r.URL,
		Headers: r.Headers,
		Query:   r.Query,
		Body:    r.Body,
	}
	if r.hasFileField {
		value.FileField = r.FileField
	}
	if r.hasFields {
		value.Fields = r.Fields
	}
	if r.hasData {
		value.Data = r.Data
	}
	return json.Marshal(value)
}

func (r responseConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		URL   extractorConfig  `json:"url"`
		Error *extractorConfig `json:"error,omitempty"`
	}{URL: r.URL, Error: r.Error})
}

func (e extractorConfig) MarshalJSON() ([]byte, error) {
	value := map[string]string{"type": e.Type}
	if e.hasPath {
		value["path"] = e.Path
	}
	if e.hasHeader {
		value["header"] = e.Header
	}
	if e.hasPattern {
		value["pattern"] = e.Pattern
	}
	if e.hasGroup {
		value["group"] = e.Group
	}
	return json.Marshal(value)
}
