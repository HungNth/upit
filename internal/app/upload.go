package app

import (
	"bufio"
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
	"unicode"
	"unicode/utf8"
)

const maxResponseSize = 1 << 20

var errResponseBeforeRequestComplete = errors.New("upload endpoint responded before request body completed")

type uploadProgressFunc func(processed, total int64)

const uploadProgressStride = 256 << 10

type uploadProgressReader struct {
	ctx          context.Context
	reader       io.Reader
	total        int64
	processed    int64
	lastReported int64
	progress     uploadProgressFunc
}

func (r *uploadProgressReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(data)
	if n > 0 {
		r.processed += int64(n)
		if r.progress != nil && (r.processed == r.total || r.processed-r.lastReported >= uploadProgressStride) {
			r.lastReported = r.processed
			r.progress(r.processed, r.total)
		}
	}
	return n, err
}

func uploadFile(ctx context.Context, client *http.Client, filePath string, selected uploader, progress uploadProgressFunc) (Result, error) {
	return uploadFileWithSnapshot(ctx, client, filePath, selected, nil, progress)
}

type uploadFileSnapshot struct {
	info os.FileInfo
}

func uploadFileWithSnapshot(ctx context.Context, client *http.Client, filePath string, selected uploader, expected *uploadFileSnapshot, progress uploadProgressFunc) (Result, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Result{}, failuref("validation", err, "open upload file: %v", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return Result{}, failuref("validation", err, "inspect upload file: %v", err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return Result{}, failure("validation", "upload path must be a regular file", nil)
	}
	if expected != nil && (!os.SameFile(expected.info, info) || expected.info.Size() != info.Size() || !expected.info.ModTime().Equal(info.ModTime())) {
		_ = file.Close()
		return Result{}, failure("validation", "selected file changed before upload", nil)
	}
	if progress != nil {
		progress(0, info.Size())
	}
	if selected.Request.Body == "form" || selected.Request.Body == "json" {
		if err := validateUTF8File(ctx, file); err != nil {
			_ = file.Close()
			return Result{}, failuref("validation", err, "validate UTF-8 upload file: %v", err)
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			_ = file.Close()
			return Result{}, failuref("validation", err, "rewind upload file: %v", err)
		}
	}

	requestURL, err := url.Parse(selected.Request.URL)
	if err != nil {
		_ = file.Close()
		return Result{}, failuref("request", err, "parse upload request URL: %v", err)
	}
	query := requestURL.Query()
	for key, value := range selected.Request.Query {
		query.Set(key, value)
	}
	requestURL.RawQuery = query.Encode()

	pipeReader, pipeWriter := io.Pipe()
	var multipartWriter *multipart.Writer
	if selected.Request.Body == "multipart" {
		multipartWriter = multipart.NewWriter(pipeWriter)
	}
	request, err := http.NewRequestWithContext(ctx, selected.Request.Method, requestURL.String(), pipeReader)
	if err != nil {
		_ = file.Close()
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		return Result{}, failuref("request", err, "build upload request: %v", err)
	}
	for key, value := range selected.Request.Headers {
		request.Header.Set(key, value)
	}
	switch selected.Request.Body {
	case "multipart":
		request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	case "form":
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	case "json":
		request.Header.Set("Content-Type", "application/json")
	case "binary":
		if _, ok := headerValue(request.Header, "Content-Type"); !ok {
			request.Header.Set("Content-Type", "application/octet-stream")
		}
	}

	writeDone := make(chan error, 1)
	go func() {
		writeErr := writeRequestBody(ctx, pipeWriter, multipartWriter, file, filePath, info.Size(), selected.Request, progress)
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
	if writeErr := joinRequestBody(pipeReader, writeDone); writeErr != nil {
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
	return finishUploadResponse(response, body, selected)
}

func writeRequestBody(ctx context.Context, w io.Writer, multipartWriter *multipart.Writer, file *os.File, filePath string, fileSize int64, request requestConfig, progress uploadProgressFunc) error {
	defer file.Close()
	reader := &uploadProgressReader{ctx: ctx, reader: file, total: fileSize, progress: progress}
	switch request.Body {
	case "multipart":
		for _, name := range slices.Sorted(maps.Keys(request.Fields)) {
			if err := multipartWriter.WriteField(name, request.Fields[name]); err != nil {
				return err
			}
		}
		part, err := multipartWriter.CreateFormFile(request.FileField, filepath.Base(filePath))
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, reader); err != nil {
			return err
		}
		return multipartWriter.Close()
	case "binary":
		_, err := io.Copy(w, reader)
		return err
	case "form":
		return writeFormBody(w, reader, request.Fields)
	case "json":
		return writeJSONBody(w, reader, request.Data)
	default:
		return fmt.Errorf("unsupported upload body %q", request.Body)
	}
}

func writeFormBody(w io.Writer, file io.Reader, fields map[string]string) error {
	buffered := bufio.NewWriterSize(w, 32<<10)
	for index, name := range slices.Sorted(maps.Keys(fields)) {
		if index > 0 {
			if err := writeString(buffered, "&"); err != nil {
				return err
			}
		}
		if err := writeFormEncodedString(buffered, name); err != nil {
			return err
		}
		if err := writeString(buffered, "="); err != nil {
			return err
		}
		if fields[name] == inputPlaceholder {
			if err := writeFormEncodedFile(buffered, file); err != nil {
				return err
			}
			continue
		}
		if err := writeFormEncodedString(buffered, fields[name]); err != nil {
			return err
		}
	}
	return buffered.Flush()
}

func writeFormEncodedString(w io.Writer, value string) error {
	for index := 0; index < len(value); index++ {
		if err := writeFormEncodedByte(w, value[index]); err != nil {
			return err
		}
	}
	return nil
}

func writeFormEncodedFile(w io.Writer, file io.Reader) error {
	reader := bufio.NewReaderSize(file, 32<<10)
	for {
		runeValue, size, err := reader.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read UTF-8 upload file: %w", err)
		}
		if runeValue == utf8.RuneError && size == 1 {
			return errors.New("upload file is not valid UTF-8")
		}
		if err := writeFormEncodedRune(w, runeValue); err != nil {
			return err
		}
	}
}

func writeFormEncodedByte(w io.Writer, value byte) error {
	if isFormUnescaped(value) {
		var encoded [1]byte
		encoded[0] = value
		return writeBytes(w, encoded[:])
	}
	if value == ' ' {
		return writeString(w, "+")
	}
	return writePercentEncodedByte(w, value)
}

func writeFormEncodedRune(w io.Writer, value rune) error {
	if value == ' ' {
		return writeString(w, "+")
	}
	if value >= 0 && value < utf8.RuneSelf {
		return writeFormEncodedByte(w, byte(value))
	}
	var encoded [utf8.UTFMax]byte
	size := utf8.EncodeRune(encoded[:], value)
	for _, value := range encoded[:size] {
		if err := writePercentEncodedByte(w, value); err != nil {
			return err
		}
	}
	return nil
}

func isFormUnescaped(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '*' || value == '-' || value == '.' || value == '_'
}

func writePercentEncodedByte(w io.Writer, value byte) error {
	const hex = "0123456789ABCDEF"
	encoded := [3]byte{'%', hex[value>>4], hex[value&0x0f]}
	return writeBytes(w, encoded[:])
}

func writeJSONBody(w io.Writer, file io.Reader, data any) error {
	buffered := bufio.NewWriterSize(w, 32<<10)
	if err := writeJSONValue(buffered, file, data); err != nil {
		return err
	}
	return buffered.Flush()
}

func writeJSONValue(w io.Writer, file io.Reader, value any) error {
	switch value := value.(type) {
	case map[string]any:
		if value == nil {
			return writeString(w, "null")
		}
		if err := writeString(w, "{"); err != nil {
			return err
		}
		for index, key := range slices.Sorted(maps.Keys(value)) {
			if index > 0 {
				if err := writeString(w, ","); err != nil {
					return err
				}
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			if err := writeBytes(w, encodedKey); err != nil {
				return err
			}
			if err := writeString(w, ":"); err != nil {
				return err
			}
			if err := writeJSONValue(w, file, value[key]); err != nil {
				return err
			}
		}
		return writeString(w, "}")
	case []any:
		if value == nil {
			return writeString(w, "null")
		}
		if err := writeString(w, "["); err != nil {
			return err
		}
		for index, child := range value {
			if index > 0 {
				if err := writeString(w, ","); err != nil {
					return err
				}
			}
			if err := writeJSONValue(w, file, child); err != nil {
				return err
			}
		}
		return writeString(w, "]")
	case string:
		if value == inputPlaceholder {
			if err := writeString(w, `"`); err != nil {
				return err
			}
			if err := writeJSONEscapedFile(w, file); err != nil {
				return err
			}
			return writeString(w, `"`)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return writeBytes(w, encoded)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return writeBytes(w, encoded)
	}
}

func writeJSONEscapedFile(w io.Writer, file io.Reader) error {
	reader := bufio.NewReaderSize(file, 32<<10)
	for {
		runeValue, size, err := reader.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read UTF-8 upload file: %w", err)
		}
		if runeValue == utf8.RuneError && size == 1 {
			return errors.New("upload file is not valid UTF-8")
		}
		var escaped [6]byte
		var encoded []byte
		switch runeValue {
		case '"':
			escaped[0], escaped[1] = '\\', '"'
			encoded = escaped[:2]
		case '\\':
			escaped[0], escaped[1] = '\\', '\\'
			encoded = escaped[:2]
		case '\b':
			escaped[0], escaped[1] = '\\', 'b'
			encoded = escaped[:2]
		case '\f':
			escaped[0], escaped[1] = '\\', 'f'
			encoded = escaped[:2]
		case '\n':
			escaped[0], escaped[1] = '\\', 'n'
			encoded = escaped[:2]
		case '\r':
			escaped[0], escaped[1] = '\\', 'r'
			encoded = escaped[:2]
		case '\t':
			escaped[0], escaped[1] = '\\', 't'
			encoded = escaped[:2]
		default:
			if runeValue < 0x20 {
				const hex = "0123456789abcdef"
				escaped[0], escaped[1], escaped[2], escaped[3] = '\\', 'u', '0', '0'
				escaped[4], escaped[5] = hex[byte(runeValue)>>4], hex[byte(runeValue)&0x0f]
				encoded = escaped[:]
			} else {
				var buffer [utf8.UTFMax]byte
				count := utf8.EncodeRune(buffer[:], runeValue)
				encoded = buffer[:count]
			}
		}
		if err := writeBytes(w, encoded); err != nil {
			return err
		}
	}
}

func writeBytes(w io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := w.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func writeString(w io.Writer, value string) error {
	if stringWriter, ok := w.(io.StringWriter); ok {
		written, err := stringWriter.WriteString(value)
		if err == nil && written != len(value) {
			return io.ErrShortWrite
		}
		return err
	}
	return writeBytes(w, []byte(value))
}

func validateUTF8File(ctx context.Context, file *os.File) error {
	reader := bufio.NewReaderSize(file, 32<<10)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		runeValue, size, err := reader.ReadRune()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if runeValue == utf8.RuneError && size == 1 {
			return errors.New("upload file is not valid UTF-8")
		}
	}
}

func joinRequestBody(pipeReader *io.PipeReader, writeDone <-chan error) error {
	select {
	case err := <-writeDone:
		return err
	default:
		_ = pipeReader.CloseWithError(errResponseBeforeRequestComplete)
		return <-writeDone
	}
}

func finishUploadResponse(response *http.Response, body []byte, selected uploader) (Result, error) {
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		message := fmt.Sprintf("upload endpoint returned HTTP %d", response.StatusCode)
		if selected.Response.Error != nil {
			if extracted, ok := extractProviderMessage(response, body, *selected.Response.Error, "response error"); ok {
				message = selected.sanitize(extracted)
			}
		}
		return Result{}, &Failure{
			Stage:      "response",
			StatusCode: response.StatusCode,
			Message:    message,
		}
	}

	extracted, err := extractResponseValue(response, body, selected.Response.URL, "response URL")
	if err != nil {
		message := err.Error()
		if selected.Response.Error != nil {
			if endpointMessage, ok := extractProviderMessage(response, body, *selected.Response.Error, "response error"); ok {
				message = selected.sanitize(endpointMessage)
			}
		}
		return Result{}, failure("parse", message, err)
	}
	if !isHTTPURL(extracted) {
		urlError := errors.New("response URL must be an absolute HTTP(S) URL")
		message := urlError.Error()
		if selected.Response.Error != nil {
			if endpointMessage, ok := extractProviderMessage(response, body, *selected.Response.Error, "response error"); ok {
				message = selected.sanitize(endpointMessage)
			}
		}
		return Result{}, failure("parse", message, urlError)
	}

	return Result{OriginalURL: extracted, FinalURL: extracted, StatusCode: response.StatusCode}, nil
}

func extractProviderMessage(response *http.Response, body []byte, extractor extractorConfig, label string) (string, bool) {
	value, err := extractResponseValue(response, body, extractor, label)
	if err != nil {
		return "", false
	}
	return value, normalizeProviderMessage(value) != ""
}

func extractResponseValue(response *http.Response, body []byte, extractor extractorConfig, label string) (string, error) {
	switch extractor.Type {
	case "json":
		var document any
		if err := json.Unmarshal(body, &document); err != nil {
			return "", fmt.Errorf("decode %s JSON response: %w", label, err)
		}
		return extractString(document, extractor, label)
	case "header":
		values := response.Header.Values(extractor.Header)
		if len(values) == 0 {
			return "", fmt.Errorf("%s header %q has no values", label, extractor.Header)
		}
		if len(values) > 1 {
			return "", fmt.Errorf("%s header %q has %d values, want 1", label, extractor.Header, len(values))
		}
		if strings.TrimSpace(values[0]) == "" {
			return "", fmt.Errorf("%s header %q is empty", label, extractor.Header)
		}
		return values[0], nil
	case "body":
		if !utf8.Valid(body) {
			return "", fmt.Errorf("%s body is not valid UTF-8", label)
		}
		value := strings.TrimSpace(string(body))
		if value == "" {
			return "", fmt.Errorf("%s body is empty", label)
		}
		return value, nil
	case "regex":
		if !utf8.Valid(body) {
			return "", fmt.Errorf("%s body is not valid UTF-8", label)
		}
		text := string(body)
		matches := extractor.regex.FindStringSubmatchIndex(text)
		if matches == nil {
			return "", fmt.Errorf("%s regex matched no values", label)
		}
		start := matches[extractor.groupIndex*2]
		end := matches[extractor.groupIndex*2+1]
		if start < 0 || end < 0 || start == end {
			return "", fmt.Errorf("%s regex group %q is empty or unmatched", label, extractor.Group)
		}
		return text[start:end], nil
	default:
		return "", fmt.Errorf("%s extractor type %q is unsupported", label, extractor.Type)
	}
}

func extractString(document any, extractor extractorConfig, label string) (string, error) {
	if extractor.compiled == nil {
		return "", fmt.Errorf("%s JSON path is not compiled", label)
	}
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

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.IsAbs() && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func headerValue(headers http.Header, name string) (string, bool) {
	values := headers.Values(name)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func normalizeProviderMessage(message string) string {
	message = strings.Map(func(value rune) rune {
		if unicode.IsControl(value) {
			return ' '
		}
		return value
	}, message)
	return strings.TrimSpace(message)
}

func (u uploader) sanitize(message string) string {
	message = normalizeProviderMessage(message)
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
