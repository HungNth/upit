package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const inputPlaceholder = "{input}"

func shortenURL(ctx context.Context, client *http.Client, originalURL string, selected shortener) (string, error) {
	replaceInputPlaceholder(selected.Request.Data, originalURL)
	body, err := json.Marshal(selected.Request.Data)
	if err != nil {
		return "", failuref("request", err, "encode shortener request: %v", err)
	}

	requestURL, err := url.Parse(selected.Request.URL)
	if err != nil {
		return "", failuref("request", err, "parse shortener request URL: %v", err)
	}
	query := requestURL.Query()
	for key, value := range selected.Request.Query {
		query.Set(key, value)
	}
	requestURL.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, selected.Request.Method, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return "", failuref("request", err, "build shortener request: %v", err)
	}
	for key, value := range selected.Request.Headers {
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return "", &Failure{Stage: "network", Message: selected.sanitize("send shortener request: " + networkErrorMessage(err)), Cause: err}
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return "", failuref("response", err, "read shortener response: %v", err)
	}
	if len(responseBody) > maxResponseSize {
		return "", &Failure{Stage: "response", StatusCode: response.StatusCode, Message: "shortener response exceeds 1 MiB"}
	}

	var document any
	decodeErr := json.Unmarshal(responseBody, &document)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("shortener endpoint returned HTTP %d", response.StatusCode)
		if decodeErr == nil && selected.Response.Error != nil {
			if extracted, extractErr := extractString(document, *selected.Response.Error, "response error"); extractErr == nil {
				message = selected.sanitize(extracted)
			}
		}
		return "", &Failure{Stage: "response", StatusCode: response.StatusCode, Message: message}
	}
	if decodeErr != nil {
		return "", failuref("parse", decodeErr, "decode shortener response: %v", decodeErr)
	}

	extracted, err := extractString(document, selected.Response.URL, "response URL")
	if err != nil {
		message := err.Error()
		if selected.Response.Error != nil {
			if endpointMessage, extractErr := extractString(document, *selected.Response.Error, "response error"); extractErr == nil {
				message = selected.sanitize(endpointMessage)
			}
		}
		return "", failure("parse", message, err)
	}
	parsed, err := url.Parse(extracted)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", failure("parse", "response URL must be an absolute HTTP(S) URL", err)
	}
	return extracted, nil
}

func countInputPlaceholders(value any) int {
	switch value := value.(type) {
	case map[string]any:
		count := 0
		for child := range value {
			count += countInputPlaceholders(value[child])
		}
		return count
	case []any:
		count := 0
		for _, child := range value {
			count += countInputPlaceholders(child)
		}
		return count
	case string:
		if value == inputPlaceholder {
			return 1
		}
	}
	return 0
}

func replaceInputPlaceholder(value any, input string) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if text, ok := child.(string); ok && text == inputPlaceholder {
				value[key] = input
				continue
			}
			replaceInputPlaceholder(child, input)
		}
	case []any:
		for index, child := range value {
			if text, ok := child.(string); ok && text == inputPlaceholder {
				value[index] = input
				continue
			}
			replaceInputPlaceholder(child, input)
		}
	}
}

func collectJSONStringValues(value any, appendValue func(string)) {
	switch value := value.(type) {
	case map[string]any:
		for child := range value {
			collectJSONStringValues(value[child], appendValue)
		}
	case []any:
		for _, child := range value {
			collectJSONStringValues(child, appendValue)
		}
	case string:
		appendValue(value)
	}
}

func (s shortener) sanitize(message string) string {
	message = normalizeProviderMessage(message)
	for _, sensitive := range s.sensitiveValues {
		message = strings.ReplaceAll(message, sensitive, "[REDACTED]")
	}
	return message
}

func networkErrorMessage(err error) string {
	message := err.Error()
	var urlError *url.Error
	if errors.As(err, &urlError) {
		message = urlError.Err.Error()
	}
	return normalizeProviderMessage(message)
}
