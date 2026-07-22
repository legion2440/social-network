package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

const latestMigrationVersion uint = 11

//go:embed migrations/*.sql
var migrationFiles embed.FS

func migrateUp(db *sql.DB) error {
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer sourceDriver.Close()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func migrateDown(db *sql.DB) error {
	if err := guardGroupPostsDownMigration(db); err != nil {
		return err
	}
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		return err
	}
	defer sourceDriver.Close()

	if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func guardGroupPostsDownMigration(db *sql.DB) error {
	if db == nil {
		return errors.New("database is required")
	}
	var version uint
	var dirty bool
	err := db.QueryRow(`SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty)
	if errors.Is(err, sql.ErrNoRows) || strings.Contains(fmt.Sprint(err), "no such table") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if dirty {
		return errors.New("database migration state is dirty")
	}
	if version < 11 {
		return nil
	}
	var groupPosts int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts WHERE group_id IS NOT NULL`).Scan(&groupPosts); err != nil {
		return fmt.Errorf("check group posts before down migration: %w", err)
	}
	if groupPosts > 0 {
		return errors.New("cannot migrate down while group posts exist")
	}
	return nil
}

func newMigrator(db *sql.DB) (*migrate.Migrate, source.Driver, error) {
	if db == nil {
		return nil, nil, errors.New("database is required")
	}

	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, nil, fmt.Errorf("open embedded migrations: %w", err)
	}
	databaseDriver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{
		MigrationsTable: "schema_migrations",
	})
	if err != nil {
		_ = sourceDriver.Close()
		return nil, nil, fmt.Errorf("open SQLite migration driver: %w", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", databaseDriver)
	if err != nil {
		_ = sourceDriver.Close()
		return nil, nil, fmt.Errorf("create migrator: %w", err)
	}
	return migrator, sourceDriver, nil
}
