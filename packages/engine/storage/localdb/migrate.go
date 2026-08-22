// Nome: migrate.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Implementa o mecanismo genérico de migrações do banco local, registrando
// versões aplicadas, executando scripts pendentes em transação e oferecendo uma base
// reutilizável para schemas mantidos pelo engine.
package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS localdb_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
)`

// Migration is one immutable, versioned local database schema change.
type Migration struct {
	Version    int
	Name       string
	Statements []string
}

// Migrate atomically applies migrations that have not been recorded yet.
// Migrations must be supplied in strictly increasing version order.
func Migrate(ctx context.Context, db *sql.DB, migrations []Migration) error {
	if ctx == nil {
		return errors.New("migration context is required")
	}
	if db == nil {
		return errors.New("local database is required")
	}
	if err := validateMigrations(migrations); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin local database migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}

	applied, err := loadAppliedMigrations(ctx, tx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if appliedName, ok := applied[migration.Version]; ok {
			if appliedName != migration.Name {
				return fmt.Errorf(
					"migration version %d is already registered as %q, not %q",
					migration.Version,
					appliedName,
					migration.Name,
				)
			}
			continue
		}

		for _, statement := range migration.Statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
			}
		}
		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO localdb_migrations (version, name) VALUES (?, ?)",
			migration.Version,
			migration.Name,
		); err != nil {
			return fmt.Errorf("record migration %d (%s): %w", migration.Version, migration.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit local database migrations: %w", err)
	}
	return nil
}

func validateMigrations(migrations []Migration) error {
	previousVersion := 0
	for _, migration := range migrations {
		if migration.Version <= 0 {
			return fmt.Errorf("migration version must be positive: %d", migration.Version)
		}
		if migration.Version <= previousVersion {
			return errors.New("migrations must be ordered by increasing unique version")
		}
		if strings.TrimSpace(migration.Name) == "" {
			return fmt.Errorf("migration %d name is required", migration.Version)
		}
		if len(migration.Statements) == 0 {
			return fmt.Errorf("migration %d must contain at least one statement", migration.Version)
		}
		for _, statement := range migration.Statements {
			if strings.TrimSpace(statement) == "" {
				return fmt.Errorf("migration %d contains an empty statement", migration.Version)
			}
		}
		previousVersion = migration.Version
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, tx *sql.Tx) (map[int]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT version, name FROM localdb_migrations")
	if err != nil {
		return nil, fmt.Errorf("read migration history: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]string)
	for rows.Next() {
		var version int
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return nil, fmt.Errorf("scan migration history: %w", err)
		}
		applied[version] = name
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration history: %w", err)
	}
	return applied, nil
}
