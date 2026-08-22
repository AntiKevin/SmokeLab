package logs

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestParseNDJSONLineParsesRequiredFieldsAndRawParams(t *testing.T) {
	t.Parallel()

	entry, err := ParseNDJSONLine(SourceLine{
		Number: 7,
		Data:   []byte(`{"timestamp":"2026-08-22T13:14:15.123456789Z","level":"info","message":"started","service_name":"api","context":{"request_id":"req-1","attempt":2},"tags":["a",true]}`),
	})
	if err != nil {
		t.Fatalf("ParseNDJSONLine returned error: %v", err)
	}

	wantTimestamp := time.Date(2026, time.August, 22, 13, 14, 15, 123456789, time.UTC)
	if !entry.Timestamp.Equal(wantTimestamp) {
		t.Fatalf("timestamp = %s, want %s", entry.Timestamp, wantTimestamp)
	}
	if entry.Level != "info" || entry.Message != "started" || entry.LineNumber != 7 {
		t.Fatalf("unexpected entry fields: %#v", entry)
	}
	if string(entry.Params["service_name"]) != `"api"` {
		t.Fatalf("service_name = %s", entry.Params["service_name"])
	}
	if string(entry.Params["context"]) != `{"request_id":"req-1","attempt":2}` {
		t.Fatalf("context was not preserved as JSON: %s", entry.Params["context"])
	}
	if string(entry.Params["tags"]) != `["a",true]` {
		t.Fatalf("tags was not preserved as JSON: %s", entry.Params["tags"])
	}
	if _, found := entry.Params["timestamp"]; found {
		t.Fatal("fixed fields must not remain in params")
	}

	var context map[string]any
	if err := json.Unmarshal(entry.Params["context"], &context); err != nil {
		t.Fatalf("context is not valid JSON: %v", err)
	}
}

func TestParseNDJSONLineRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want error
	}{
		{name: "invalid UTF-8", data: []byte{0xff}, want: ErrInvalidUTF8},
		{name: "empty", data: []byte(" \t "), want: ErrEmptyLine},
		{name: "array", data: []byte(`[]`), want: ErrJSONObjectRequired},
		{name: "null", data: []byte(`null`), want: ErrJSONObjectRequired},
		{name: "multiple values", data: []byte(`{} {}`), want: ErrMultipleJSONValues},
		{name: "missing timestamp", data: []byte(`{"level":"info","message":"x"}`)},
		{name: "non-string level", data: []byte(`{"timestamp":"2026-08-22T13:14:15Z","level":1,"message":"x"}`)},
		{name: "blank message", data: []byte(`{"timestamp":"2026-08-22T13:14:15Z","level":"info","message":" \t "}`)},
		{name: "invalid timestamp", data: []byte(`{"timestamp":"not-a-time","level":"info","message":"x"}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseNDJSONLine(SourceLine{Data: test.data})
			if err == nil {
				t.Fatal("expected error")
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want wrapped %v", err, test.want)
			}
		})
	}
}
