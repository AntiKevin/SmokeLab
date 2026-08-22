package logs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// ErrEmptyLine reports a line that has no JSON value.
	ErrEmptyLine = errors.New("NDJSON line is empty")
	// ErrInvalidUTF8 reports bytes that do not encode UTF-8 text.
	ErrInvalidUTF8 = errors.New("NDJSON line is not valid UTF-8")
	// ErrJSONObjectRequired reports a JSON value that is not an object.
	ErrJSONObjectRequired = errors.New("NDJSON line must contain a JSON object")
	// ErrMultipleJSONValues reports more than one JSON value on a line.
	ErrMultipleJSONValues = errors.New("NDJSON line contains multiple JSON values")
)

// NDJSONParser validates a single line of structured log input.
type NDJSONParser struct{}

// Parse parses one NDJSON line into a log entry without source metadata.
func (NDJSONParser) Parse(line SourceLine) (LogEntry, error) {
	return ParseNDJSONLine(line)
}

// ParseNDJSONLine parses one complete JSON object from a single NDJSON line.
func ParseNDJSONLine(line SourceLine) (LogEntry, error) {
	if !utf8.Valid(line.Data) {
		return LogEntry{}, ErrInvalidUTF8
	}
	if len(bytes.TrimSpace(line.Data)) == 0 {
		return LogEntry{}, ErrEmptyLine
	}

	decoder := json.NewDecoder(bytes.NewReader(line.Data))
	var object json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		return LogEntry{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(object) == 0 || object[0] != '{' {
		return LogEntry{}, ErrJSONObjectRequired
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(object, &fields); err != nil {
		return LogEntry{}, fmt.Errorf("invalid JSON object: %w", err)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return LogEntry{}, ErrMultipleJSONValues
		}
		return LogEntry{}, fmt.Errorf("invalid JSON after object: %w", err)
	}

	timestampText, err := requiredString(fields, "timestamp")
	if err != nil {
		return LogEntry{}, err
	}
	timestamp, err := time.Parse(time.RFC3339Nano, timestampText)
	if err != nil {
		return LogEntry{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	level, err := requiredString(fields, "level")
	if err != nil {
		return LogEntry{}, err
	}
	if strings.TrimSpace(level) == "" {
		return LogEntry{}, errors.New("level must not be empty")
	}

	message, err := requiredString(fields, "message")
	if err != nil {
		return LogEntry{}, err
	}
	if strings.TrimSpace(message) == "" {
		return LogEntry{}, errors.New("message must not be empty")
	}

	delete(fields, "timestamp")
	delete(fields, "level")
	delete(fields, "message")

	params := make(map[string]json.RawMessage, len(fields))
	for key, value := range fields {
		var compactValue bytes.Buffer
		if err := json.Compact(&compactValue, value); err != nil {
			return LogEntry{}, fmt.Errorf("invalid parameter %q: %w", key, err)
		}
		params[key] = append(json.RawMessage(nil), compactValue.Bytes()...)
	}

	return LogEntry{
		Timestamp:  timestamp,
		Level:      level,
		Message:    message,
		Params:     params,
		LineNumber: line.Number,
	}, nil
}

func requiredString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, found := fields[name]
	if !found {
		return "", fmt.Errorf("%s is required", name)
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", name)
	}

	return value, nil
}
