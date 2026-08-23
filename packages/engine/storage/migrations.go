// Package storage provides persistent engine repositories backed by local storage.
// Nome: migrations.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Declara as migrações específicas do domínio de armazenamento de logs,
// mantendo a evolução do schema local no engine e separada dos comandos de terminal
// ou da interface visual que apenas consomem esses dados.
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
	{
		Version: 2,
		Name:    "index log exploration filters",
		Statements: []string{
			"CREATE INDEX IF NOT EXISTS logs_level_idx ON logs (level)",
			"CREATE INDEX IF NOT EXISTS logs_source_descriptor_idx ON logs (source_kind, source_name, source_id)",
		},
	},
	{
		Version: 3,
		Name:    "index log exploration ordering",
		Statements: []string{
			"CREATE INDEX IF NOT EXISTS logs_timestamp_order_idx ON logs (" + logTimestampOrderExpressionV3 + ", id)",
			"CREATE INDEX IF NOT EXISTS logs_captured_at_order_idx ON logs (" + logCapturedAtOrderExpressionV3 + ", id)",
		},
	},
}

// Migrate applies all local storage schema migrations.
func Migrate(ctx context.Context, db *sql.DB) error {
	return localdb.Migrate(ctx, db, migrations)
}
