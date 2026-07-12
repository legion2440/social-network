package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenBootstrapsMinimalSchemaAndIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "social-network.db")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	assertBootstrapSchema(t, db)
	if err := db.Close(); err != nil {
		t.Fatalf("close first database: %v", err)
	}

	db, err = Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db.Close()
	assertBootstrapSchema(t, db)
}

func assertBootstrapSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, table := range []string{"users", "sessions", "media"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected table %s", table)
		}
	}

	var migrations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&migrations); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if migrations != 0 {
		t.Fatal("schema_migrations must not be created")
	}
	for _, table := range []string{
		"categories",
		"posts",
		"comments",
		"reactions",
		"moderation_reports",
		"auth_identities",
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatalf("query excluded table %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("forum-specific table %s must not be created", table)
		}
	}

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatal("foreign keys must be enabled")
	}
}
