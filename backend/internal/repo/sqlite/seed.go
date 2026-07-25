package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const demoPasswordHashPlaceholder = "{{DEMO_PASSWORD_HASH}}"

var seedMigrationName = regexp.MustCompile(`^([0-9]{6})_.+\.up\.sql$`)

//go:embed seedmigrations/*.up.sql
var seedMigrationFiles embed.FS

type seedMigration struct {
	version uint
	name    string
	sql     string
}

// ApplyDemoSeedMigrations applies the opt-in demo dataset without changing
// schema_migrations. Each embedded seed version and its data commit together.
func ApplyDemoSeedMigrations(ctx context.Context, db *sql.DB, passwordHash string) ([]uint, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	passwordHash = strings.TrimSpace(passwordHash)
	if passwordHash == "" {
		return nil, errors.New("demo password hash is required")
	}
	migrations, err := embeddedSeedMigrations()
	if err != nil {
		return nil, err
	}
	hasSeedTable, err := seedMigrationsTableExists(ctx, db)
	if err != nil {
		return nil, err
	}
	applied := make(map[uint]struct{})
	if hasSeedTable {
		applied, err = appliedSeedVersions(ctx, db)
		if err != nil {
			return nil, err
		}
	}
	if len(applied) == 0 {
		conflictingEmail, err := existingDemoEmail(ctx, db)
		if err != nil {
			return nil, err
		}
		if conflictingEmail != "" {
			return nil, fmt.Errorf("demo seed requires an empty demo namespace: user %q already exists", conflictingEmail)
		}
	}
	if !hasSeedTable {
		if _, err := db.ExecContext(ctx, `
			CREATE TABLE seed_migrations (
				version INTEGER PRIMARY KEY,
				applied_at INTEGER NOT NULL
			)
		`); err != nil {
			return nil, fmt.Errorf("create seed_migrations: %w", err)
		}
	}

	known := make(map[uint]struct{}, len(migrations))
	for _, migration := range migrations {
		known[migration.version] = struct{}{}
	}
	for version := range applied {
		if _, ok := known[version]; !ok {
			return nil, fmt.Errorf("seed migration version %d has no embedded source", version)
		}
	}

	escapedHash := strings.ReplaceAll(passwordHash, "'", "''")
	newlyApplied := make([]uint, 0, len(migrations))
	for _, migration := range migrations {
		if _, ok := applied[migration.version]; ok {
			continue
		}
		rendered := strings.ReplaceAll(migration.sql, demoPasswordHashPlaceholder, escapedHash)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return newlyApplied, fmt.Errorf("begin seed migration %d: %w", migration.version, err)
		}
		if _, err := tx.ExecContext(ctx, rendered); err != nil {
			_ = tx.Rollback()
			return newlyApplied, fmt.Errorf("apply seed migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO seed_migrations (version, applied_at) VALUES (?, unixepoch())`,
			migration.version,
		); err != nil {
			_ = tx.Rollback()
			return newlyApplied, fmt.Errorf("record seed migration %d: %w", migration.version, err)
		}
		if err := tx.Commit(); err != nil {
			return newlyApplied, fmt.Errorf("commit seed migration %d: %w", migration.version, err)
		}
		newlyApplied = append(newlyApplied, migration.version)
	}
	return newlyApplied, nil
}

func seedMigrationsTableExists(ctx context.Context, db *sql.DB) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sqlite_master
			WHERE type = 'table' AND name = 'seed_migrations'
		)
	`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check seed_migrations table: %w", err)
	}
	return exists, nil
}

func existingDemoEmail(ctx context.Context, db *sql.DB) (string, error) {
	var email string
	err := db.QueryRowContext(ctx, `
		SELECT email
		FROM users
		WHERE email IN (
			'alice.demo@example.com',
			'bob.demo@example.com',
			'carol.demo@example.com'
		)
		COLLATE NOCASE
		ORDER BY email
		LIMIT 1
	`).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("check demo email namespace: %w", err)
	}
	return email, nil
}

func embeddedSeedMigrations() ([]seedMigration, error) {
	names, err := fs.Glob(seedMigrationFiles, "seedmigrations/*.up.sql")
	if err != nil {
		return nil, fmt.Errorf("list embedded seed migrations: %w", err)
	}
	sort.Strings(names)
	migrations := make([]seedMigration, 0, len(names))
	versions := make(map[uint]string, len(names))
	for _, name := range names {
		base := path.Base(name)
		match := seedMigrationName.FindStringSubmatch(base)
		if match == nil {
			return nil, fmt.Errorf("invalid seed migration filename %q", base)
		}
		value, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil || value == 0 {
			return nil, fmt.Errorf("invalid seed migration version in %q", base)
		}
		version := uint(value)
		if previous, ok := versions[version]; ok {
			return nil, fmt.Errorf("duplicate seed migration version %d in %q and %q", version, previous, base)
		}
		content, err := seedMigrationFiles.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("read seed migration %q: %w", base, err)
		}
		versions[version] = base
		migrations = append(migrations, seedMigration{version: version, name: base, sql: string(content)})
	}
	if len(migrations) == 0 {
		return nil, errors.New("no embedded seed migrations")
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	return migrations, nil
}

func appliedSeedVersions(ctx context.Context, db *sql.DB) (map[uint]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM seed_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("read seed migration versions: %w", err)
	}
	defer rows.Close()
	versions := make(map[uint]struct{})
	for rows.Next() {
		var version uint
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan seed migration version: %w", err)
		}
		versions[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate seed migration versions: %w", err)
	}
	return versions, nil
}
