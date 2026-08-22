// Nome: log_repository_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Testa a persistência de logs estruturados no repositório SQL, validando
// criação de registros, serialização de parâmetros, associação de fonte e leitura
// dos dados gravados para proteger o contrato de armazenamento.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"SmokeLab/packages/engine/logs"
	"SmokeLab/packages/engine/storage/localdb"
)

func TestLogMigrationIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDatabase(t, ctx)

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("first Migrate returned error: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate returned error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM localdb_migrations WHERE name = ?", "create logs").Scan(&count); err != nil {
		t.Fatalf("count log migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("log migration count = %d, want 1", count)
	}

	if _, err := db.Exec(`INSERT INTO logs (
        timestamp, level, message, source_kind, source_name, source_id,
        line_number, captured_at, params
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"2026-08-22T12:00:00Z", "info", "ready", "file", "app.log", "source-1",
		1, "2026-08-22T12:00:01Z", "{}",
	); err != nil {
		t.Fatalf("insert into migrated logs table: %v", err)
	}
}

func TestLogRepositoryPersistsBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDatabase(t, ctx)
	repository, err := NewLogRepository(ctx, db)
	if err != nil {
		t.Fatalf("NewLogRepository returned error: %v", err)
	}
	capturedAt := time.Date(2026, time.August, 22, 14, 30, 0, 123456789, time.FixedZone("local", -3*60*60))
	repository.now = func() time.Time { return capturedAt }

	entry := logs.LogEntry{
		Timestamp: time.Date(2026, time.August, 22, 17, 29, 59, 987654321, time.UTC),
		Level:     "warn",
		Message:   "request failed",
		Params: map[string]json.RawMessage{
			"attempt": json.RawMessage(`2`),
			"details": json.RawMessage(`{"retry":true}`),
		},
		Source: logs.SourceDescriptor{
			Kind: "file",
			Name: "service.ndjson",
			ID:   "source-42",
		},
		LineNumber: 17,
	}
	if err := repository.Store(ctx, []logs.LogEntry{entry}); err != nil {
		t.Fatalf("Store returned error: %v", err)
	}

	var got struct {
		timestamp  string
		level      string
		message    string
		sourceKind string
		sourceName string
		sourceID   string
		lineNumber int
		capturedAt string
		params     string
	}
	err = db.QueryRow(`SELECT
        timestamp, level, message, source_kind, source_name, source_id,
        line_number, captured_at, params
    FROM logs`).Scan(
		&got.timestamp,
		&got.level,
		&got.message,
		&got.sourceKind,
		&got.sourceName,
		&got.sourceID,
		&got.lineNumber,
		&got.capturedAt,
		&got.params,
	)
	if err != nil {
		t.Fatalf("query stored log: %v", err)
	}

	if got.timestamp != entry.Timestamp.Format(time.RFC3339Nano) ||
		got.level != entry.Level ||
		got.message != entry.Message ||
		got.sourceKind != entry.Source.Kind ||
		got.sourceName != entry.Source.Name ||
		got.sourceID != entry.Source.ID ||
		got.lineNumber != entry.LineNumber ||
		got.capturedAt != capturedAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("unexpected stored columns: %#v", got)
	}

	var gotParams map[string]json.RawMessage
	if err := json.Unmarshal([]byte(got.params), &gotParams); err != nil {
		t.Fatalf("decode stored params: %v", err)
	}
	if !reflect.DeepEqual(gotParams, entry.Params) {
		t.Fatalf("stored params = %#v, want %#v", gotParams, entry.Params)
	}
}

func TestLogRepositoryRollsBackBatchOnEncodingError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openTestDatabase(t, ctx)
	repository, err := NewLogRepository(ctx, db)
	if err != nil {
		t.Fatalf("NewLogRepository returned error: %v", err)
	}

	entries := []logs.LogEntry{
		{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "valid entry",
		},
		{
			Timestamp: time.Now(),
			Level:     "error",
			Message:   "invalid params",
			Params: map[string]json.RawMessage{
				"broken": json.RawMessage(`{"missing":"brace"`),
			},
		},
	}
	if err := repository.Store(ctx, entries); err == nil {
		t.Fatal("Store should reject invalid params JSON")
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count); err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if count != 0 {
		t.Fatalf("stored log count = %d, want 0 after rollback", count)
	}
}

func openTestDatabase(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()

	db, err := localdb.Open(ctx, filepath.Join(t.TempDir(), "storage.db"))
	if err != nil {
		t.Fatalf("localdb.Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	})
	return db
}
