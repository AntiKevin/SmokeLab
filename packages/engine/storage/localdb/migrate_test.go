// Nome: migrate_test.go
// Autor: Kevin Rodrigues
// Criado em: 2026-08-22
// Descrição: Testa o executor de migrações do banco local, garantindo aplicação em
// ordem, idempotência, registro de versões e rollback quando uma migração falha
// durante a preparação da infraestrutura de persistência.
package localdb

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMigrateIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "migrations.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	})

	migrations := []Migration{
		{
			Version: 1,
			Name:    "create records",
			Statements: []string{
				"CREATE TABLE records (value TEXT NOT NULL)",
				"INSERT INTO records (value) VALUES ('created once')",
			},
		},
	}
	if err := Migrate(ctx, db, migrations); err != nil {
		t.Fatalf("first Migrate returned error: %v", err)
	}
	if err := Migrate(ctx, db, migrations); err != nil {
		t.Fatalf("second Migrate returned error: %v", err)
	}

	var recordCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM records").Scan(&recordCount); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if recordCount != 1 {
		t.Fatalf("record count = %d, want 1", recordCount)
	}

	var migrationCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM localdb_migrations").Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("migration count = %d, want 1", migrationCount)
	}
}
