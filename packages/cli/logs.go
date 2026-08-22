// Nome: logs.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Define os comandos de terminal para ingestão de logs estruturados,
// traduzindo flags, fontes de entrada e opções operacionais em chamadas para o
// engine, mantendo a CLI como camada de adaptação e apresentação.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"SmokeLab/packages/cli/internal/source"
	"SmokeLab/packages/engine/logs"
	"SmokeLab/packages/engine/storage"
	"SmokeLab/packages/engine/storage/localdb"
	"github.com/spf13/cobra"
)

type ingestConfig struct {
	stdin        bool
	file         string
	follow       bool
	databasePath string
	onInvalid    string
	batchSize    int
	maxLineBytes int
}

func newLogsCommand() *cobra.Command {
	logsCommand := &cobra.Command{Use: "logs", Short: "Manage structured logs"}
	logsCommand.AddCommand(newLogsIngestCommand())
	return logsCommand
}

func newLogsIngestCommand() *cobra.Command {
	defaultDB := localdb.DefaultPath()
	config := ingestConfig{
		databasePath: defaultDB,
		onInvalid:    "fail",
		batchSize:    100,
		maxLineBytes: source.DefaultMaxLineBytes,
	}

	command := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest NDJSON logs from stdin or a file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogsIngest(cmd.Context(), cmd.ErrOrStderr(), config)
		},
	}

	flags := command.Flags()
	flags.BoolVar(&config.stdin, "stdin", false, "read NDJSON from standard input")
	flags.StringVar(&config.file, "file", "", "read NDJSON from a file")
	flags.BoolVar(&config.follow, "follow", false, "follow appended lines in a file")
	flags.StringVar(&config.databasePath, "db", defaultDB, "local database path")
	flags.StringVar(&config.onInvalid, "on-invalid", "fail", "invalid line policy: fail or skip")
	flags.IntVar(&config.batchSize, "batch-size", 100, "number of entries per database batch")
	flags.IntVar(&config.maxLineBytes, "max-line-bytes", source.DefaultMaxLineBytes, "maximum NDJSON line size in bytes")
	return command
}

func runLogsIngest(ctx context.Context, stderr io.Writer, config ingestConfig) error {
	policy, err := validateIngestConfig(config)
	if err != nil {
		return err
	}

	input, closeInput, err := newIngestSource(config)
	if err != nil {
		return err
	}
	defer closeInput()

	repository, closeRepository, err := newLogRepository(ctx, config.databasePath)
	if err != nil {
		return err
	}
	defer closeRepository()

	result, err := logs.NewIngestService(repository).Ingest(ctx, input, logs.IngestOptions{
		BatchSize:     config.batchSize,
		InvalidPolicy: policy,
	})
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stderr, "ingested: read=%d accepted=%d persisted=%d invalid=%d skipped=%d batches=%d\n",
		result.LinesRead,
		result.EntriesAccepted,
		result.EntriesPersisted,
		result.InvalidCount,
		result.LinesSkipped,
		result.BatchesFlushed,
	)
	return err
}

func validateIngestConfig(config ingestConfig) (logs.InvalidPolicy, error) {
	if config.stdin == (config.file != "") {
		return logs.FailOnInvalid, errors.New("provide exactly one of --stdin or --file")
	}
	if config.follow && config.file == "" {
		return logs.FailOnInvalid, errors.New("--follow requires --file")
	}
	if config.batchSize <= 0 {
		return logs.FailOnInvalid, errors.New("--batch-size must be positive")
	}
	if config.maxLineBytes <= 0 {
		return logs.FailOnInvalid, errors.New("--max-line-bytes must be positive")
	}

	switch strings.ToLower(config.onInvalid) {
	case "fail":
		return logs.FailOnInvalid, nil
	case "skip":
		return logs.SkipInvalid, nil
	default:
		return logs.FailOnInvalid, fmt.Errorf("invalid --on-invalid value %q: use fail or skip", config.onInvalid)
	}
}

func newIngestSource(config ingestConfig) (logs.Source, func(), error) {
	if config.stdin {
		input, err := source.NewStdin(os.Stdin, config.maxLineBytes)
		return input, func() {}, err
	}
	if config.follow {
		input, err := source.OpenFollow(config.file, config.maxLineBytes, 250*time.Millisecond)
		return input, func() {}, err
	}

	input, err := source.OpenFile(config.file, config.maxLineBytes)
	if err != nil {
		return nil, nil, err
	}
	return input, func() { _ = input.Close() }, nil
}

// newLogRepository is the sole CLI-to-storage integration point.
func newLogRepository(ctx context.Context, databasePath string) (logs.Repository, func(), error) {
	database, err := localdb.Open(ctx, databasePath)
	if err != nil {
		return nil, func() {}, err
	}
	repository, err := storage.NewLogRepository(ctx, database)
	if err != nil {
		_ = database.Close()
		return nil, func() {}, err
	}
	return repository, func() { _ = database.Close() }, nil
}
