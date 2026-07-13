package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenAppliesVersionedMigrationsAndIsIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nested", "social-network.db")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()
	assertMigratedSchema(t, db)
	assertMigrationVersion(t, db, int(latestMigrationVersion), false)
	if err := db.Close(); err != nil {
		t.Fatalf("close first database: %v", err)
	}
	db = nil

	db, err = Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	assertMigratedSchema(t, db)
	assertMigrationVersion(t, db, int(latestMigrationVersion), false)
}

func TestMigrationsRunAllTheWayDownAndBackUp(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := migrateDown(db); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	for _, table := range []string{"sessions", "media", "users"} {
		if tableExists(t, db, table) {
			t.Fatalf("table %s still exists after down migrations", table)
		}
	}

	if err := migrateUp(db); err != nil {
		t.Fatalf("migrate back up: %v", err)
	}
	assertMigratedSchema(t, db)
	assertMigrationVersion(t, db, int(latestMigrationVersion), false)
}

func TestOpenRejectsDisposableLegacyBootstrapDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	if _, err := legacy.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)
	`); err != nil {
		_ = legacy.Close()
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	if db, err := Open(context.Background(), dbPath); err == nil {
		_ = db.Close()
		t.Fatal("expected legacy bootstrap database to require one-time removal")
	}
}

func assertMigratedSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	for _, table := range []string{"users", "media", "sessions", "schema_migrations"} {
		if !tableExists(t, db, table) {
			t.Fatalf("expected table %s", table)
		}
	}

	wantUserColumns := []string{
		"id",
		"email",
		"password_hash",
		"first_name",
		"last_name",
		"date_of_birth",
		"gender",
		"nickname",
		"about_me",
		"avatar_media_id",
		"created_at",
		"updated_at",
	}
	columns := tableColumns(t, db, "users")
	for _, column := range wantUserColumns {
		if _, ok := columns[column]; !ok {
			t.Fatalf("users column %s is missing", column)
		}
	}
	if !columns["id"].primaryKey {
		t.Fatal("users.id must be the primary key")
	}
	for _, column := range []string{"email", "password_hash", "first_name", "last_name", "date_of_birth", "created_at", "updated_at"} {
		if !columns[column].notNull {
			t.Fatalf("users.%s must be NOT NULL", column)
		}
	}
	for _, column := range []string{"gender", "nickname", "about_me", "avatar_media_id"} {
		if columns[column].notNull {
			t.Fatalf("users.%s must be nullable", column)
		}
	}
	for _, tableAndColumn := range []struct {
		table  string
		column string
	}{
		{table: "users", column: "date_of_birth"},
		{table: "users", column: "created_at"},
		{table: "users", column: "updated_at"},
		{table: "media", column: "created_at"},
		{table: "sessions", column: "expires_at"},
		{table: "sessions", column: "created_at"},
	} {
		definition := tableColumns(t, db, tableAndColumn.table)[tableAndColumn.column]
		if definition.columnType != "INTEGER" {
			t.Fatalf("%s.%s must store Unix seconds as INTEGER, got %s", tableAndColumn.table, tableAndColumn.column, definition.columnType)
		}
	}

	for _, table := range []string{
		"categories",
		"posts",
		"comments",
		"reactions",
		"moderation_reports",
		"auth_identities",
	} {
		if tableExists(t, db, table) {
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
	var busyTimeout int
	if err := db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout pragma: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("expected busy timeout 5000ms, got %dms", busyTimeout)
	}
	assertForeignKey(t, db, "media", "owner_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "users", "avatar_media_id", "media", "id", "SET NULL")
	assertForeignKey(t, db, "sessions", "user_id", "users", "id", "CASCADE")
}

func assertMigrationVersion(t *testing.T, db *sql.DB, wantVersion int, wantDirty bool) {
	t.Helper()
	var version int
	var dirty bool
	if err := db.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
		t.Fatalf("query migration version: %v", err)
	}
	if version != wantVersion || dirty != wantDirty {
		t.Fatalf("unexpected migration state version=%d dirty=%t", version, dirty)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		t.Fatalf("query table %s: %v", table, err)
	}
	return count == 1
}

type columnDefinition struct {
	columnType string
	notNull    bool
	primaryKey bool
}

func tableColumns(t *testing.T, db *sql.DB, table string) map[string]columnDefinition {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("query columns for %s: %v", table, err)
	}
	defer rows.Close()

	columns := make(map[string]columnDefinition)
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			t.Fatalf("scan column for %s: %v", table, err)
		}
		columns[name] = columnDefinition{
			columnType: columnType,
			notNull:    notNull != 0,
			primaryKey: primaryKey != 0,
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns for %s: %v", table, err)
	}
	return columns
}

func assertForeignKey(t *testing.T, db *sql.DB, table, fromColumn, targetTable, targetColumn, onDelete string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA foreign_key_list(` + table + `)`)
	if err != nil {
		t.Fatalf("query foreign keys for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id, sequence           int
			actualTable            string
			from, to               string
			onUpdate, actualDelete string
			match                  string
		)
		if err := rows.Scan(&id, &sequence, &actualTable, &from, &to, &onUpdate, &actualDelete, &match); err != nil {
			t.Fatalf("scan foreign key for %s: %v", table, err)
		}
		if from == fromColumn && actualTable == targetTable && to == targetColumn && actualDelete == onDelete {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign keys for %s: %v", table, err)
	}
	t.Fatalf("missing foreign key %s.%s -> %s.%s ON DELETE %s", table, fromColumn, targetTable, targetColumn, onDelete)
}
