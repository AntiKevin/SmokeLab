// Nome: ingest.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Coordena o fluxo de ingestão de logs estruturados no engine, lendo
// fontes abstratas, aplicando parsing e política de registros inválidos, formando
// lotes e delegando a persistência para o repositório configurado.
package logs

import (
	"context"
	"errors"
	"fmt"
)

// IngestService coordinates parsing and batched persistence of a Source.
type IngestService struct {
	repository Repository
	parser     NDJSONParser
}

// NewIngestService creates an ingestion service backed by repository.
func NewIngestService(repository Repository) *IngestService {
	return &IngestService{repository: repository}
}

// Ingest reads, validates, and persists logs from source according to options.
func (s *IngestService) Ingest(ctx context.Context, source Source, options IngestOptions) (IngestResult, error) {
	if ctx == nil {
		return IngestResult{}, errors.New("ingest context is required")
	}
	if source == nil {
		return IngestResult{}, errors.New("log source is required")
	}
	if s == nil || s.repository == nil {
		return IngestResult{}, errors.New("log repository is required")
	}

	options, err := options.normalized()
	if err != nil {
		return IngestResult{}, err
	}

	result := IngestResult{}
	descriptor := source.Descriptor()
	batch := make([]LogEntry, 0, options.BatchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries := append([]LogEntry(nil), batch...)
		if err := s.repository.Store(ctx, entries); err != nil {
			return fmt.Errorf("store log batch: %w", err)
		}
		result.EntriesPersisted += len(batch)
		result.BatchesFlushed++
		batch = batch[:0]
		return nil
	}

	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		line, err := source.Next(ctx)
		if isEndOfSource(err) {
			if err := flush(); err != nil {
				return result, err
			}
			return result, nil
		}
		if err != nil {
			return result, fmt.Errorf("read log source: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}

		result.LinesRead++
		entry, err := s.parser.Parse(line)
		if err != nil {
			invalid := InvalidLine{Source: descriptor, Line: line, Cause: err}
			result.InvalidLines = append(result.InvalidLines, invalid)
			result.InvalidCount++
			if options.InvalidPolicy == InvalidPolicyFail {
				return result, invalid
			}
			result.LinesSkipped++
			continue
		}

		entry.Application = options.Application
		entry.Source = descriptor
		batch = append(batch, entry)
		result.EntriesAccepted++
		if len(batch) == options.BatchSize {
			if err := flush(); err != nil {
				return result, err
			}
		}
	}
}
