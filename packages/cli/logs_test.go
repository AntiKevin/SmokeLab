// Nome: logs_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Exercita os cenários da CLI de ingestão de logs, cobrindo validação
// de configuração, seleção de fonte, escrita de resumo e integração básica com
// persistência local sem duplicar as regras centrais do engine.
package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"SmokeLab/packages/engine/logs"
)

func TestIngestProgressReportsCommittedEntriesInInteractiveTerminal(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	progress := &ingestProgress{
		writer:  &output,
		enabled: true,
		now:     func() time.Time { return now },
	}
	if err := progress.start(); err != nil {
		t.Fatalf("start returned error: %v", err)
	}
	progress.addPersisted(1)
	progress.addPersisted(1)
	now = now.Add(progressRefreshInterval)
	progress.addPersisted(1)
	if err := progress.finish(logs.IngestResult{LinesRead: 3, EntriesAccepted: 3, EntriesPersisted: 3, BatchesFlushed: 3}); err != nil {
		t.Fatalf("finish returned error: %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"\ringesting logs: persisted=0",
		"\ringesting logs: persisted=1",
		"\ringesting logs: persisted=3",
		"\ringested: read=3 accepted=3 persisted=3 invalid=0 skipped=0 batches=3\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output %q does not contain %q", got, want)
		}
	}
}

func TestValidateIngestConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  ingestConfig
		want    logs.InvalidPolicy
		wantErr bool
	}{
		{name: "stdin fail", config: ingestConfig{stdin: true, onInvalid: "fail", batchSize: 1, maxLineBytes: 1}, want: logs.FailOnInvalid},
		{name: "file skip", config: ingestConfig{file: "logs.ndjson", onInvalid: "skip", batchSize: 1, maxLineBytes: 1}, want: logs.SkipInvalid},
		{name: "both inputs", config: ingestConfig{stdin: true, file: "logs.ndjson", onInvalid: "fail", batchSize: 1, maxLineBytes: 1}, wantErr: true},
		{name: "follow stdin", config: ingestConfig{stdin: true, follow: true, onInvalid: "fail", batchSize: 1, maxLineBytes: 1}, wantErr: true},
		{name: "invalid policy", config: ingestConfig{stdin: true, onInvalid: "ignore", batchSize: 1, maxLineBytes: 1}, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateIngestConfig(test.config)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateIngestConfig() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("policy = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLogsIngestFileWritesSummaryToStderr(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	inputPath := filepath.Join(directory, "input.ndjson")
	databasePath := filepath.Join(directory, "logs.db")
	data := `{"timestamp":"2026-08-22T13:14:15Z","level":"info","message":"started"}`
	if err := os.WriteFile(inputPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	command := newRootCommand()
	var stdout, stderr bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stderr)
	command.SetArgs([]string{"logs", "ingest", "--file", inputPath, "--db", databasePath})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no progress output", stdout.String())
	}
	if got, want := stderr.String(), "ingested: read=1 accepted=1 persisted=1 invalid=0 skipped=0 batches=1\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}
