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

const progressRefreshInterval = 100 * time.Millisecond

type progressRepository struct {
	repository logs.Repository
	progress   *ingestProgress
}

func (r *progressRepository) Store(ctx context.Context, entries []logs.LogEntry) error {
	if err := r.repository.Store(ctx, entries); err != nil {
		return err
	}
	r.progress.addPersisted(len(entries))
	return nil
}

type ingestProgress struct {
	writer       io.Writer
	enabled      bool
	now          func() time.Time
	lastRendered time.Time
	persisted    int
	writeErr     error
}

func newIngestProgress(writer io.Writer) *ingestProgress {
	return &ingestProgress{
		writer:  writer,
		enabled: isTerminalWriter(writer),
		now:     time.Now,
	}
}

func (p *ingestProgress) start() error {
	if !p.enabled {
		return nil
	}
	p.lastRendered = p.now()
	p.render()
	return p.writeErr
}

func (p *ingestProgress) addPersisted(count int) {
	p.persisted += count
	if !p.enabled || p.writeErr != nil {
		return
	}

	now := p.now()
	if p.persisted == count || now.Sub(p.lastRendered) >= progressRefreshInterval {
		p.lastRendered = now
		p.render()
	}
}

func (p *ingestProgress) render() {
	_, p.writeErr = fmt.Fprintf(p.writer, "\ringesting logs: persisted=%d", p.persisted)
}

func (p *ingestProgress) finish(result logs.IngestResult) error {
	if p.writeErr != nil {
		return p.writeErr
	}
	prefix := ""
	if p.enabled {
		prefix = "\r"
	}
	_, err := fmt.Fprintf(p.writer, prefix+"ingested: read=%d accepted=%d persisted=%d invalid=%d skipped=%d batches=%d\n",
		result.LinesRead,
		result.EntriesAccepted,
		result.EntriesPersisted,
		result.InvalidCount,
		result.LinesSkipped,
		result.BatchesFlushed,
	)
	return err
}

func (p *ingestProgress) abort() {
	if p.enabled {
		_, _ = fmt.Fprintln(p.writer)
	}
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
		batchSize:    logs.DefaultBatchSize,
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
	flags.IntVar(&config.batchSize, "batch-size", logs.DefaultBatchSize, "number of entries per database batch")
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
	progress := newIngestProgress(stderr)
	if err := progress.start(); err != nil {
		return err
	}
	repository = &progressRepository{repository: repository, progress: progress}

	result, err := logs.NewIngestService(repository).Ingest(ctx, input, logs.IngestOptions{
		BatchSize:     config.batchSize,
		InvalidPolicy: policy,
	})
	if err != nil {
		progress.abort()
		return err
	}

	return progress.finish(result)
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
