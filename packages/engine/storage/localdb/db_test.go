package localdb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenConfiguresLocalDatabase(t *testing.T) {
	t.Parallel()

	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "storage.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	})

	assertPragmaInteger(t, db, "foreign_keys", 1)
	assertPragmaInteger(t, db, "busy_timeout", 5000)

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want %q", journalMode, "wal")
	}
}

func TestDefaultPathUsesUserConfigDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))

	path := DefaultPath()
	if !strings.HasSuffix(path, filepath.Join("SmokeLab", "smokelab.db")) {
		t.Fatalf("DefaultPath() = %q, want SmokeLab database path", path)
	}
	if !strings.HasPrefix(path, os.Getenv("XDG_CONFIG_HOME")) {
		t.Fatalf("DefaultPath() = %q, want under XDG_CONFIG_HOME", path)
	}
}

func assertPragmaInteger(t *testing.T, db *sql.DB, name string, want int) {
	t.Helper()

	var got int
	if err := db.QueryRow("PRAGMA " + name).Scan(&got); err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %d, want %d", name, got, want)
	}
}
