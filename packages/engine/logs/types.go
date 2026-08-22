// Nome: types.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define os contratos centrais do engine para logs estruturados, incluindo
// entradas validadas, descritores de fonte, linhas brutas, políticas de invalidez,
// repositórios e métricas compartilhadas pelo fluxo de ingestão.
// Package logs contains the engine contracts for structured log ingestion.
package logs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const defaultBatchSize = 100

// LogEntry is one validated structured log record.
type LogEntry struct {
	Timestamp  time.Time                  `json:"timestamp"`
	Level      string                     `json:"level"`
	Message    string                     `json:"message"`
	Params     map[string]json.RawMessage `json:"params,omitempty"`
	Source     SourceDescriptor           `json:"source"`
	LineNumber int                        `json:"lineNumber"`
}

// SourceLine is one raw line emitted by a Source. Number is one-based when
// the source can provide it.
type SourceLine struct {
	Number int
	Data   []byte
}

// RawLine is kept as an alias for callers that use the raw-input terminology.
type RawLine = SourceLine

// SourceDescriptor identifies where a log entry came from. Its fields are
// deliberately generic so terminal, file, and future sources can share it.
type SourceDescriptor struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Kind string `json:"kind,omitempty"`
}

// Source supplies NDJSON lines to be ingested. It must return io.EOF when no
// more lines are available.
type Source interface {
	Descriptor() SourceDescriptor
	Next(context.Context) (SourceLine, error)
}

// Repository persists validated entries in batches.
type Repository interface {
	Store(context.Context, []LogEntry) error
}

// InvalidPolicy controls how IngestService handles a malformed input line.
type InvalidPolicy int

const (
	// InvalidPolicyFail stops ingestion on the first invalid line.
	InvalidPolicyFail InvalidPolicy = iota
	// InvalidPolicySkip records invalid lines and continues ingestion.
	InvalidPolicySkip
)

const (
	// FailOnInvalid is the readable alias for InvalidPolicyFail.
	FailOnInvalid = InvalidPolicyFail
	// SkipInvalid is the readable alias for InvalidPolicySkip.
	SkipInvalid = InvalidPolicySkip
)

// IngestOptions configures one ingestion run. A zero BatchSize uses 100.
// The zero InvalidPolicy fails on invalid input.
type IngestOptions struct {
	BatchSize     int
	InvalidPolicy InvalidPolicy
}

// IngestResult describes the work completed before ingestion returned.
type IngestResult struct {
	LinesRead        int
	EntriesAccepted  int
	EntriesPersisted int
	InvalidCount     int
	LinesSkipped     int
	BatchesFlushed   int
	InvalidLines     []InvalidLine
}

// InvalidLine identifies a source line rejected by the NDJSON parser.
type InvalidLine struct {
	Source SourceDescriptor
	Line   SourceLine
	Cause  error
}

func (e InvalidLine) Error() string {
	if e.Line.Number > 0 {
		return fmt.Sprintf("invalid log line %d: %v", e.Line.Number, e.Cause)
	}

	return fmt.Sprintf("invalid log line: %v", e.Cause)
}

// Unwrap exposes the parser error that caused this line to be invalid.
func (e InvalidLine) Unwrap() error {
	return e.Cause
}

func (o IngestOptions) normalized() (IngestOptions, error) {
	if o.BatchSize <= 0 {
		o.BatchSize = defaultBatchSize
	}

	if o.InvalidPolicy != InvalidPolicyFail && o.InvalidPolicy != InvalidPolicySkip {
		return IngestOptions{}, fmt.Errorf("invalid invalid-line policy: %d", o.InvalidPolicy)
	}

	return o, nil
}

func isEndOfSource(err error) bool {
	return errors.Is(err, io.EOF)
}
