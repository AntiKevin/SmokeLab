// Package storage provides persistent engine repositories backed by local storage.
package storage

import (
	"context"
	"database/sql"

	"SmokeLab/packages/engine/storage/localdb"
)

var migrations = []localdb.Migration{
	{
		Version: 1,
		Name:    "create logs",
		Statements: []string{
			`CREATE TABLE IF NOT EXISTS logs (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                timestamp TEXT NOT NULL,
                level TEXT NOT NULL,
                message TEXT NOT NULL,
                source_kind TEXT NOT NULL,
                source_name TEXT NOT NULL,
                source_id TEXT NOT NULL,
                line_number INTEGER NOT NULL,
                captured_at TEXT NOT NULL,
                params TEXT NOT NULL CHECK (json_valid(params))
            )`,
			"CREATE INDEX IF NOT EXISTS logs_timestamp_idx ON logs (timestamp)",
			"CREATE INDEX IF NOT EXISTS logs_source_idx ON logs (source_kind, source_id)",
		},
	},
}

// Migrate applies all local storage schema migrations.
func Migrate(ctx context.Context, db *sql.DB) error {
	return localdb.Migrate(ctx, db, migrations)
}
