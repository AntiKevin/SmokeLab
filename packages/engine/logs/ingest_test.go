package logs

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestIngestServiceBatchesSkipsInvalidLinesAndFlushesFinalBatch(t *testing.T) {
	t.Parallel()

	source := &fakeSource{
		descriptor: SourceDescriptor{ID: "terminal-1", Name: "local app", Kind: "terminal"},
		lines: []SourceLine{
			{Number: 1, Data: []byte(`{"timestamp":"2026-08-22T13:14:15Z","level":"info","message":"one"}`)},
			{Number: 2, Data: []byte(`not-json`)},
			{Number: 3, Data: []byte(`{"timestamp":"2026-08-22T13:14:17Z","level":"warn","message":"two"}`)},
			{Number: 4, Data: []byte(`{"timestamp":"2026-08-22T13:14:18Z","level":"error","message":"three"}`)},
		},
	}
	repository := &fakeRepository{}

	result, err := NewIngestService(repository).Ingest(context.Background(), source, IngestOptions{
		BatchSize:     2,
		InvalidPolicy: SkipInvalid,
	})
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}
	if result.LinesRead != 4 || result.EntriesAccepted != 3 || result.EntriesPersisted != 3 || result.InvalidCount != 1 || result.LinesSkipped != 1 || result.BatchesFlushed != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.InvalidLines) != 1 || result.InvalidLines[0].Line.Number != 2 {
		t.Fatalf("unexpected invalid lines: %#v", result.InvalidLines)
	}
	if len(repository.batches) != 2 || len(repository.batches[0]) != 2 || len(repository.batches[1]) != 1 {
		t.Fatalf("unexpected batches: %#v", repository.batches)
	}
	for _, batch := range repository.batches {
		for _, entry := range batch {
			if entry.Source != source.descriptor {
				t.Fatalf("entry source = %#v, want %#v", entry.Source, source.descriptor)
			}
		}
	}
}

func TestIngestServiceFailsOnInvalidLine(t *testing.T) {
	t.Parallel()

	source := &fakeSource{lines: []SourceLine{
		{Number: 1, Data: []byte(`{"timestamp":"2026-08-22T13:14:15Z","level":"info","message":"one"}`)},
		{Number: 2, Data: []byte(`{}`)},
	}}
	repository := &fakeRepository{}

	result, err := NewIngestService(repository).Ingest(context.Background(), source, IngestOptions{BatchSize: 10, InvalidPolicy: FailOnInvalid})
	if err == nil {
		t.Fatal("expected ingestion failure")
	}
	var invalid InvalidLine
	if !errors.As(err, &invalid) || invalid.Line.Number != 2 {
		t.Fatalf("expected InvalidLine for line 2, got %v", err)
	}
	if result.LinesRead != 2 || result.EntriesAccepted != 1 || len(result.InvalidLines) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(repository.batches) != 0 {
		t.Fatalf("pending batch should not be flushed after failure: %#v", repository.batches)
	}
}

func TestIngestServiceHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := &fakeSource{lines: []SourceLine{{Number: 1, Data: []byte(`{}`)}}}

	_, err := NewIngestService(&fakeRepository{}).Ingest(ctx, source, IngestOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
	if source.nextCalls != 0 {
		t.Fatalf("source should not be read after cancellation, got %d reads", source.nextCalls)
	}
}

type fakeSource struct {
	descriptor SourceDescriptor
	lines      []SourceLine
	nextCalls  int
}

func (s *fakeSource) Descriptor() SourceDescriptor {
	return s.descriptor
}

func (s *fakeSource) Next(ctx context.Context) (SourceLine, error) {
	s.nextCalls++
	if err := ctx.Err(); err != nil {
		return SourceLine{}, err
	}
	if len(s.lines) == 0 {
		return SourceLine{}, io.EOF
	}

	line := s.lines[0]
	s.lines = s.lines[1:]
	return line, nil
}

type fakeRepository struct {
	batches [][]LogEntry
	err     error
}

func (r *fakeRepository) Store(ctx context.Context, entries []LogEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.err != nil {
		return r.err
	}

	batch := append([]LogEntry(nil), entries...)
	r.batches = append(r.batches, batch)
	return nil
}
