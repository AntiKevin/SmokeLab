// Package localdb provides the engine's generic local database infrastructure.
package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultDatabaseFile = "smokelab.db"

// DefaultPath returns the shared local database path for SmokeLab.
func DefaultPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		return defaultDatabaseFile
	}

	return filepath.Join(configDir, "SmokeLab", defaultDatabaseFile)
}

// Open opens and configures a local database stored at path.
// The special SQLite path ":memory:" is supported for short-lived databases.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if ctx == nil {
		return nil, errors.New("local database context is required")
	}
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("local database path is required")
	}
	if err := ensureParentDirectory(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open local database: %w", err)
	}

	// These pragmas are connection-local. Keeping one reusable connection also
	// preserves the contents of an in-memory database across operations.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to local database: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure local database: %w", err)
		}
	}

	return db, nil
}

func ensureParentDirectory(path string) error {
	if path == ":memory:" || strings.HasPrefix(path, "file:") {
		return nil
	}

	directory := filepath.Dir(path)
	if directory == "." || directory == "" {
		return nil
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create local database directory: %w", err)
	}

	return nil
}
