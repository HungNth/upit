package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const maxResponseSize = 1 << 20

var errResponseBeforeRequestComplete = errors.New("upload endpoint responded before request body completed")

func uploadFile(ctx context.Context, client *http.Client, filePath string, selected uploader) (Result, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Result{}, failuref("validation", err, "open upload file: %v", err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return Result{}, failuref("validation", err, "inspect upload file: %v", err)
	}
	if !info.Mode().IsRegular() {
		file.Close()
		return Result{}, failure("validation", "upload path must be a regular file", nil)
	}

	requestURL, err := url.Parse(selected.Request.URL)
	if err != nil {
		file.Close()
		return Result{}, failuref("request", err, "parse upload request URL: %v", err)
	}
	query := requestURL.Query()
	for key, value := range selected.Request.Query {
		query.Set(key, value)
	}
	requestURL.RawQuery = query.Encode()

	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	request, err := http.NewRequestWithContext(ctx, selected.Request.Method, requestURL.String(), pipeReader)
	if err != nil {
		file.Close()
		pipeReader.Close()
		pipeWriter.Close()
		return Result{}, failuref("request", err, "build upload request: %v", err)
	}
	for key, value := range selected.Request.Headers {
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	writeDone := make(chan error, 1)
	go func() {
		defer file.Close()

		var writeErr error
		for _, name := range slices.Sorted(maps.Keys(selected.Request.Fields)) {
			writeErr = multipartWriter.WriteField(name, selected.Request.Fields[name])
			if writeErr != nil {
				break
			}
		}
		var part io.Writer
		if writeErr == nil {
			part, writeErr = multipartWriter.CreateFormFile(selected.Request.FileField, filepath.Base(filePath))
		}
		if writeErr == nil {
			_, writeErr = io.Copy(part, file)
		}
		if writeErr == nil {
			writeErr = multipartWriter.Close()
		}
		if writeErr != nil {
			_ = pipeWriter.CloseWithError(writeErr)
		} else {
			_ = pipeWriter.Close()
		}
		writeDone <- writeErr
	}()

	response, err := client.Do(request)
	if err != nil {
		_ = pipeReader.CloseWithError(err)
		writeErr := <-writeDone
		if writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
			return Result{}, failuref("request", writeErr, "stream upload file: %v", writeErr)
		}
		return Result{}, &Failure{Stage: "network", Message: selected.sanitize(networkFailureMessage(err)), Cause: err}
	}
	defer response.Body.Close()
	defer pipeReader.Close()

	body, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if writeErr := joinMultipartWriter(pipeReader, writeDone); writeErr != nil {
		if errors.Is(writeErr, io.ErrClosedPipe) || errors.Is(writeErr, errResponseBeforeRequestComplete) {
			return Result{}, failure("request", errResponseBeforeRequestComplete.Error(), writeErr)
		}
		return Result{}, failuref("request", writeErr, "stream upload file: %v", writeErr)
	}
	if readErr != nil {
		return Result{}, failuref("response", readErr, "read upload response: %v", readErr)
	}
	if len(body) > maxResponseSize {
		return Result{}, &Failure{
			Stage:      "response",
			StatusCode: response.StatusCode,
			Message:    "upload response exceeds 1 MiB",
		}
	}
	var document any
	decodeErr := json.Unmarshal(body, &document)
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("upload endpoint returned HTTP %d", response.StatusCode)
		if decodeErr == nil && selected.Response.Error != nil {
			if extracted, extractErr := extractString(document, *selected.Response.Error, "response error"); extractErr == nil {
				message = selected.sanitize(extracted)
			}
		}
		return Result{}, &Failure{
			Stage:      "response",
			StatusCode: response.StatusCode,
			Message:    message,
		}
	}

	if decodeErr != nil {
		return Result{}, failuref("parse", decodeErr, "decode upload response: %v", decodeErr)
	}
	extracted, err := extractString(document, selected.Response.URL, "response URL")
	if err != nil {
		message := err.Error()
		if selected.Response.Error != nil {
			if endpointMessage, extractErr := extractString(document, *selected.Response.Error, "response error"); extractErr == nil {
				message = selected.sanitize(endpointMessage)
			}
		}
		return Result{}, failure("parse", message, err)
	}
	parsed, err := url.Parse(extracted)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return Result{}, failure("parse", "response URL must be an absolute HTTP(S) URL", err)
	}

	return Result{OriginalURL: extracted, FinalURL: extracted, StatusCode: response.StatusCode}, nil
}

func joinMultipartWriter(pipeReader *io.PipeReader, writeDone <-chan error) error {
	select {
	case err := <-writeDone:
		return err
	default:
		_ = pipeReader.CloseWithError(errResponseBeforeRequestComplete)
		return <-writeDone
	}
}

func extractString(document any, extractor extractorConfig, label string) (string, error) {
	nodes := extractor.compiled.Select(document)
	if len(nodes) == 0 {
		return "", fmt.Errorf("%s path selected no values", label)
	}
	if len(nodes) > 1 {
		return "", fmt.Errorf("%s path selected %d values, want 1", label, len(nodes))
	}
	value, ok := nodes[0].(string)
	if !ok {
		return "", fmt.Errorf("%s must select a JSON string", label)
	}
	return value, nil
}

func (u uploader) sanitize(message string) string {
	for _, sensitive := range u.sensitiveValues {
		message = strings.ReplaceAll(message, sensitive, "[REDACTED]")
	}
	return message
}

func networkFailureMessage(err error) string {
	message := err.Error()
	var urlError *url.Error
	if errors.As(err, &urlError) {
		message = urlError.Err.Error()
	}
	return "send upload request: " + message
}
