package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"SmokeLab/packages/engine/logs"
)

const insertLog = `
INSERT INTO logs (
    timestamp,
    level,
    message,
    source_kind,
    source_name,
    source_id,
    line_number,
    captured_at,
    params
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

// LogRepository persists structured log entries in the local database.
type LogRepository struct {
	db  *sql.DB
	now func() time.Time
}

var _ logs.Repository = (*LogRepository)(nil)

// NewLogRepository applies pending storage migrations and creates a log repository.
func NewLogRepository(ctx context.Context, db *sql.DB) (*LogRepository, error) {
	if ctx == nil {
		return nil, errors.New("log repository context is required")
	}
	if db == nil {
		return nil, errors.New("local database is required")
	}
	if err := Migrate(ctx, db); err != nil {
		return nil, fmt.Errorf("prepare log repository: %w", err)
	}

	return &LogRepository{db: db, now: time.Now}, nil
}

// Store persists entries atomically as one batch.
func (r *LogRepository) Store(ctx context.Context, entries []logs.LogEntry) error {
	if ctx == nil {
		return errors.New("store context is required")
	}
	if r == nil || r.db == nil {
		return errors.New("log repository is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin log batch: %w", err)
	}
	defer tx.Rollback()

	statement, err := tx.PrepareContext(ctx, insertLog)
	if err != nil {
		return fmt.Errorf("prepare log insert: %w", err)
	}
	defer statement.Close()

	capturedAt := r.now().UTC().Format(time.RFC3339Nano)
	for index, entry := range entries {
		params, err := marshalParams(entry.Params)
		if err != nil {
			return fmt.Errorf("encode params for log %d: %w", index, err)
		}

		if _, err := statement.ExecContext(
			ctx,
			entry.Timestamp.UTC().Format(time.RFC3339Nano),
			entry.Level,
			entry.Message,
			entry.Source.Kind,
			entry.Source.Name,
			entry.Source.ID,
			entry.LineNumber,
			capturedAt,
			string(params),
		); err != nil {
			return fmt.Errorf("insert log %d: %w", index, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit log batch: %w", err)
	}
	return nil
}

func marshalParams(params map[string]json.RawMessage) ([]byte, error) {
	if params == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(params)
}
