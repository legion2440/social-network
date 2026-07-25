package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

func TestDemoSeedMigrationsAreOptInVersionedAndIdempotent(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var ignored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM seed_migrations`).Scan(&ignored); err == nil {
		t.Fatal("normal database startup unexpectedly created seed_migrations")
	}
	var schemaVersion uint
	var schemaDirty bool
	if err := db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&schemaVersion, &schemaDirty); err != nil {
		t.Fatal(err)
	}
	if schemaVersion != 16 || schemaDirty {
		t.Fatalf("schema migration state=(%d, %t)", schemaVersion, schemaDirty)
	}

	hash, err := service.HashPassword("LoopDemo123!")
	if err != nil {
		t.Fatal(err)
	}
	applied, err := sqlite.ApplyDemoSeedMigrations(ctx, db, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 1 || applied[0] != 1 {
		t.Fatalf("applied=%v", applied)
	}

	assertCount(t, db, `SELECT COUNT(*) FROM seed_migrations`, 1)
	assertCount(t, db, `SELECT COUNT(*) FROM users WHERE email LIKE '%.demo@example.com'`, 3)
	assertCount(t, db, `SELECT COUNT(*) FROM users WHERE email LIKE '%.demo@example.com' AND is_private = 1`, 1)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM notification_user_states state
		JOIN users u ON u.id = state.user_id
		WHERE u.email LIKE '%.demo@example.com' AND state.revision = 0
	`, 3)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM chat_user_states state
		JOIN users u ON u.id = state.user_id
		WHERE u.email LIKE '%.demo@example.com' AND state.revision = 0
	`, 3)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM follows f
		JOIN users follower ON follower.id = f.follower_user_id
		JOIN users followed ON followed.id = f.followed_user_id
		WHERE follower.email LIKE '%.demo@example.com'
		  AND followed.email = 'alice.demo@example.com'
		  AND f.status = 'accepted'
	`, 2)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM posts p
		JOIN users u ON u.id = p.author_user_id
		WHERE u.email LIKE '%.demo@example.com' AND p.group_id IS NULL
	`, 3)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM post_selected_users audience
		JOIN users u ON u.id = audience.user_id
		WHERE u.email = 'bob.demo@example.com'
	`, 1)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM group_memberships membership
		JOIN groups g ON g.id = membership.group_id
		WHERE g.title = 'Loop Demo Group'
		  AND membership.status IN ('owner', 'member')
	`, 3)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM group_chat_read_states state
		JOIN group_memberships membership ON membership.id = state.membership_id
		JOIN groups g ON g.id = membership.group_id
		WHERE g.title = 'Loop Demo Group'
		  AND state.last_read_message_id IS NULL
		  AND state.unread_count = 0
	`, 3)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM posts p
		JOIN groups g ON g.id = p.group_id
		WHERE g.title = 'Loop Demo Group' AND p.privacy IS NULL
	`, 1)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM group_events event
		JOIN groups g ON g.id = event.group_id
		WHERE g.title = 'Loop Demo Group'
	`, 1)
	assertCount(t, db, `
		SELECT COUNT(*)
		FROM group_event_responses response
		JOIN group_events event ON event.id = response.event_id
		JOIN groups g ON g.id = event.group_id
		WHERE g.title = 'Loop Demo Group' AND response.response = 'going'
	`, 1)

	var storedHash string
	if err := db.QueryRow(`
		SELECT password_hash FROM users WHERE email = 'alice.demo@example.com'
	`).Scan(&storedHash); err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyPassword(storedHash, "LoopDemo123!"); err != nil {
		t.Fatalf("demo password does not use the application bcrypt flow: %v", err)
	}

	reapplied, err := sqlite.ApplyDemoSeedMigrations(ctx, db, hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(reapplied) != 0 {
		t.Fatalf("second run applied versions %v", reapplied)
	}
	assertCount(t, db, `SELECT COUNT(*) FROM users WHERE email LIKE '%.demo@example.com'`, 3)

	if err := db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&schemaVersion, &schemaDirty); err != nil {
		t.Fatal(err)
	}
	if schemaVersion != 16 || schemaDirty {
		t.Fatalf("seed changed schema migration state=(%d, %t)", schemaVersion, schemaDirty)
	}
}

func TestDemoSeedMigrationsRejectExistingDemoEmailBeforeChanges(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "seed-collision.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		INSERT INTO users (
			email, password_hash, first_name, last_name, date_of_birth,
			gender, nickname, about_me, created_at, updated_at
		) VALUES (
			'alice.demo@example.com', 'existing-hash', 'Existing', 'User', '01-01-1990',
			NULL, NULL, NULL, unixepoch(), unixepoch()
		)
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := sqlite.ApplyDemoSeedMigrations(ctx, db, "new-seed-hash"); err == nil {
		t.Fatal("expected a demo email collision error")
	}
	var ignored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM seed_migrations`).Scan(&ignored); err == nil {
		t.Fatal("failed seed unexpectedly created seed_migrations")
	}
	assertCount(t, db, `SELECT COUNT(*) FROM users`, 1)
	assertCount(t, db, `SELECT COUNT(*) FROM groups`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM posts`, 0)
}

func assertCount(t *testing.T, db *sql.DB, query string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count=%d want=%d for %s", got, want, query)
	}
}
