package sqlite

import (
	"context"
	"database/sql"
	"fmt"
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
	for _, table := range []string{"group_chat_read_states", "direct_chat_read_states", "chat_user_states", "notification_user_states", "notifications", "group_event_responses", "group_events", "chat_messages", "direct_conversations", "group_memberships", "groups", "post_comments", "post_selected_users", "posts", "follows", "sessions", "media", "users"} {
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

	for _, table := range []string{"users", "media", "sessions", "follows", "posts", "post_selected_users", "post_comments", "groups", "group_memberships", "group_events", "group_event_responses", "direct_conversations", "chat_messages", "notifications", "notification_user_states", "chat_user_states", "direct_chat_read_states", "group_chat_read_states", "schema_migrations"} {
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
		"is_private",
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
	for _, column := range []string{"email", "password_hash", "first_name", "last_name", "date_of_birth", "is_private", "created_at", "updated_at"} {
		if !columns[column].notNull {
			t.Fatalf("users.%s must be NOT NULL", column)
		}
	}
	for _, column := range []string{"gender", "nickname", "about_me", "avatar_media_id"} {
		if columns[column].notNull {
			t.Fatalf("users.%s must be nullable", column)
		}
	}
	dateOfBirth := columns["date_of_birth"]
	if dateOfBirth.columnType != "TEXT" {
		t.Fatalf("users.date_of_birth must store DD-MM-YYYY as TEXT, got %s", dateOfBirth.columnType)
	}
	for _, tableAndColumn := range []struct {
		table  string
		column string
	}{
		{table: "users", column: "created_at"},
		{table: "users", column: "updated_at"},
		{table: "media", column: "created_at"},
		{table: "sessions", column: "expires_at"},
		{table: "sessions", column: "created_at"},
		{table: "follows", column: "created_at"},
		{table: "follows", column: "updated_at"},
		{table: "groups", column: "created_at"},
		{table: "group_memberships", column: "created_at"},
		{table: "group_memberships", column: "updated_at"},
		{table: "group_events", column: "starts_at"},
		{table: "group_events", column: "created_at"},
		{table: "group_event_responses", column: "created_at"},
		{table: "group_event_responses", column: "updated_at"},
		{table: "direct_conversations", column: "created_at"},
		{table: "chat_messages", column: "created_at"},
		{table: "notifications", column: "resolved_at"},
		{table: "notifications", column: "read_at"},
		{table: "notifications", column: "created_at"},
	} {
		definition := tableColumns(t, db, tableAndColumn.table)[tableAndColumn.column]
		if definition.columnType != "INTEGER" {
			t.Fatalf("%s.%s must store Unix seconds as INTEGER, got %s", tableAndColumn.table, tableAndColumn.column, definition.columnType)
		}
	}

	for _, table := range []string{
		"categories",
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
	assertForeignKey(t, db, "follows", "follower_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "follows", "followed_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "posts", "author_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "posts", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "posts", "media_id", "media", "id", "SET NULL")
	assertForeignKey(t, db, "post_selected_users", "post_id", "posts", "id", "CASCADE")
	assertForeignKey(t, db, "post_selected_users", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "post_comments", "post_id", "posts", "id", "CASCADE")
	assertForeignKey(t, db, "post_comments", "author_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "post_comments", "media_id", "media", "id", "SET NULL")
	assertForeignKey(t, db, "groups", "owner_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "notifications", "recipient_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "notifications", "actor_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "notifications", "follow_id", "follows", "id", "SET NULL")
	assertForeignKey(t, db, "notifications", "group_id", "groups", "id", "SET NULL")
	assertForeignKey(t, db, "notifications", "event_id", "group_events", "id", "SET NULL")
	assertForeignKey(t, db, "notifications", "membership_id", "group_memberships", "id", "SET NULL")
	assertForeignKey(t, db, "notification_user_states", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "group_events", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "group_events", "creator_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "group_event_responses", "event_id", "group_events", "id", "CASCADE")
	assertForeignKey(t, db, "group_event_responses", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "direct_conversations", "user_low_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "direct_conversations", "user_high_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "direct_conversation_id", "direct_conversations", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "sender_user_id", "users", "id", "CASCADE")
	if !schemaObjectExists(t, db, "index", "idx_post_comments_media") {
		t.Fatal("expected unique comment media index")
	}
}

func TestGroupPostSchemaConstraintsAndIndex(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	userResult, err := db.Exec(`
		INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES ('group-post-schema@example.com', 'hash', 'Group', 'Poster', '22-07-1992', 1, 1)
	`)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ := userResult.LastInsertId()
	groupResult, err := db.Exec(`INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, 'Posts', 'Description', 1)`, userID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	groupID, _ := groupResult.LastInsertId()

	if _, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, created_at) VALUES (?, 'personal', 'public', 1)`, userID); err != nil {
		t.Fatalf("insert personal post: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (author_user_id, group_id, text, created_at) VALUES (?, ?, 'group', 1)`, userID, groupID); err != nil {
		t.Fatalf("insert group post: %v", err)
	}
	for name, statement := range map[string]string{
		"personal without privacy": `INSERT INTO posts (author_user_id, text, created_at) VALUES (?, 'invalid', 1)`,
		"group with privacy":       `INSERT INTO posts (author_user_id, group_id, text, privacy, created_at) VALUES (?, ?, 'invalid', 'public', 1)`,
	} {
		t.Run(name, func(t *testing.T) {
			var err error
			if name == "group with privacy" {
				_, err = db.Exec(statement, userID, groupID)
			} else {
				_, err = db.Exec(statement, userID)
			}
			if err == nil {
				t.Fatal("expected one-of constraint failure")
			}
		})
	}
	if !schemaObjectExists(t, db, "index", "idx_posts_group_created") {
		t.Fatal("missing group post pagination index")
	}
}

func TestMigration11PreservesPersonalPostGraphAndAutoincrement(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-11.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Migrate(10); err != nil {
		t.Fatalf("migrate to version 10: %v", err)
	}
	_ = sourceDriver.Close()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES
			(1, 'migration-author@example.com', 'hash', 'Migration', 'Author', '22-07-1992', 1, 1),
			(2, 'migration-reader@example.com', 'hash', 'Migration', 'Reader', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	mediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (1, 'image/png', 8, 'migration.png', 'migration.png', 2)
	`)
	if err != nil {
		t.Fatalf("seed media: %v", err)
	}
	mediaID, _ := mediaResult.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO posts (id, author_user_id, text, privacy, media_id, created_at)
		VALUES (41, 1, 'selected migration post', 'selected', ?, 3)
	`, mediaID); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_selected_users (post_id, user_id) VALUES (41, 2)`); err != nil {
		t.Fatalf("seed audience: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO post_comments (id, post_id, author_user_id, text, created_at)
		VALUES (71, 41, 2, 'preserved comment', 4)
	`); err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO posts (id, author_user_id, text, privacy, created_at)
		VALUES (100, 1, 'deleted high post', 'public', 4)
	`); err != nil {
		t.Fatalf("seed high post sequence: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO post_comments (id, post_id, author_user_id, text, created_at)
		VALUES (170, 100, 2, 'deleted high comment', 4)
	`); err != nil {
		t.Fatalf("seed high comment sequence: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM posts WHERE id = 100`); err != nil {
		t.Fatalf("delete high post while retaining sequences: %v", err)
	}

	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new version 11 migrator: %v", err)
	}
	if err := migrator.Migrate(11); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
	_ = sourceDriver.Close()
	assertMigrationVersion(t, db, 11, false)
	var (
		groupID       sql.NullInt64
		privacy       string
		gotMediaID    int64
		commentsCount int64
	)
	if err := db.QueryRow(`
		SELECT group_id, privacy, media_id,
			(SELECT COUNT(*) FROM post_comments WHERE post_id = posts.id)
		FROM posts WHERE id = 41
	`).Scan(&groupID, &privacy, &gotMediaID, &commentsCount); err != nil {
		t.Fatalf("read migrated post: %v", err)
	}
	if groupID.Valid || privacy != "selected" || gotMediaID != mediaID || commentsCount != 1 {
		t.Fatalf("migrated post mismatch: group=%+v privacy=%q media=%d comments=%d", groupID, privacy, gotMediaID, commentsCount)
	}
	var audienceUserID, commentID int64
	if err := db.QueryRow(`SELECT user_id FROM post_selected_users WHERE post_id = 41`).Scan(&audienceUserID); err != nil || audienceUserID != 2 {
		t.Fatalf("migrated audience: user=%d err=%v", audienceUserID, err)
	}
	if err := db.QueryRow(`SELECT id FROM post_comments WHERE post_id = 41`).Scan(&commentID); err != nil || commentID != 71 {
		t.Fatalf("migrated comment: id=%d err=%v", commentID, err)
	}
	result, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, created_at) VALUES (1, 'next post', 'public', 5)`)
	if err != nil {
		t.Fatalf("insert post after migration: %v", err)
	}
	nextID, _ := result.LastInsertId()
	if nextID <= 100 {
		t.Fatalf("post AUTOINCREMENT did not preserve deleted high ID: %d", nextID)
	}
	commentResult, err := db.Exec(`
		INSERT INTO post_comments (post_id, author_user_id, text, created_at)
		VALUES (?, 2, 'next comment', 5)
	`, nextID)
	if err != nil {
		t.Fatalf("insert comment after migration: %v", err)
	}
	nextCommentID, _ := commentResult.LastInsertId()
	if nextCommentID <= 170 {
		t.Fatalf("comment AUTOINCREMENT did not preserve deleted high ID: %d", nextCommentID)
	}
}

func TestMigration13BackfillsNotificationStateAndPhysicalMembershipIDs(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migration-13.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Migrate(12); err != nil {
		t.Fatalf("migrate to version 12: %v", err)
	}
	_ = sourceDriver.Close()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES
			(1, 'notification-owner@example.com', 'hash', 'Owner', 'User', '22-07-1992', 1, 1),
			(2, 'notification-member@example.com', 'hash', 'Member', 'User', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO groups (id, owner_user_id, title, description, created_at) VALUES (1, 1, 'Group', 'Description', 1)`); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at)
		VALUES (1, 1, 'owner', 1, 1), (1, 2, 'member', 2, 2)
	`); err != nil {
		t.Fatalf("seed memberships: %v", err)
	}

	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new version 13 migrator: %v", err)
	}
	if err := migrator.Migrate(13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}
	_ = sourceDriver.Close()

	rows, err := db.Query(`SELECT id, user_id, status FROM group_memberships ORDER BY user_id`)
	if err != nil {
		t.Fatalf("read memberships: %v", err)
	}
	defer rows.Close()
	seenIDs := map[int64]bool{}
	seenStatuses := map[int64]string{}
	for rows.Next() {
		var id, userID int64
		var status string
		if err := rows.Scan(&id, &userID, &status); err != nil {
			t.Fatalf("scan membership: %v", err)
		}
		if id <= 0 || seenIDs[id] {
			t.Fatalf("invalid physical membership id %d", id)
		}
		seenIDs[id] = true
		seenStatuses[userID] = status
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("membership rows: %v", err)
	}
	if seenStatuses[1] != "owner" || seenStatuses[2] != "member" {
		t.Fatalf("membership states not preserved: %+v", seenStatuses)
	}
	var stateRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification_user_states WHERE revision = 0`).Scan(&stateRows); err != nil || stateRows != 2 {
		t.Fatalf("notification state backfill: rows=%d err=%v", stateRows, err)
	}

	if _, err := db.Exec(`INSERT INTO group_memberships (id, group_id, user_id, status, created_at, updated_at) VALUES (100, 1, 2, 'member', 2, 2)`); err == nil {
		t.Fatal("expected unique group/user membership constraint")
	}
	if _, err := db.Exec(`DELETE FROM group_memberships WHERE user_id = 2`); err != nil {
		t.Fatalf("delete member: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_memberships (id, group_id, user_id, status, created_at, updated_at) VALUES (100, 1, 2, 'member', 3, 3)`); err != nil {
		t.Fatalf("insert high membership id: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM group_memberships WHERE id = 100`); err != nil {
		t.Fatalf("delete high membership id: %v", err)
	}
	result, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (1, 2, 'member', 4, 4)`)
	if err != nil {
		t.Fatalf("insert membership after deleted high id: %v", err)
	}
	nextID, _ := result.LastInsertId()
	if nextID <= 100 {
		t.Fatalf("membership AUTOINCREMENT reused a deleted lifecycle id: %d", nextID)
	}
}

func TestMigration14BackfillsChatReadStatesAndConstraints(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migration-14.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Migrate(13); err != nil {
		t.Fatalf("migrate to version 13: %v", err)
	}
	_ = sourceDriver.Close()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES
			(1, 'chat-read-one@example.com', 'hash', 'One', 'User', '22-07-1992', 1, 1),
			(2, 'chat-read-two@example.com', 'hash', 'Two', 'User', '22-07-1992', 1, 1),
			(3, 'chat-read-three@example.com', 'hash', 'Three', 'User', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO groups (id, owner_user_id, title, description, created_at)
		VALUES (1, 1, 'Chat group', 'Description', 1)
	`); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO group_memberships (id, group_id, user_id, status, created_at, updated_at)
		VALUES
			(10, 1, 1, 'owner', 1, 1),
			(11, 1, 2, 'member', 1, 1),
			(12, 1, 3, 'invited', 1, 1)
	`); err != nil {
		t.Fatalf("seed memberships: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO direct_conversations (id, user_low_id, user_high_id, created_at)
		VALUES (20, 1, 2, 1)
	`); err != nil {
		t.Fatalf("seed direct conversation: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO chat_messages (
			id, direct_conversation_id, group_id, sender_user_id, client_message_id, body, created_at
		) VALUES
			(30, 20, NULL, 1, 'direct-30', 'direct', 30),
			(31, NULL, 1, 2, 'group-31', 'group', 31)
	`); err != nil {
		t.Fatalf("seed chat messages: %v", err)
	}

	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new version 14 migrator: %v", err)
	}
	if err := migrator.Migrate(14); err != nil {
		t.Fatalf("migrate to version 14: %v", err)
	}
	_ = sourceDriver.Close()

	var userStates, directStates, groupStates int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_user_states WHERE revision = 0`).Scan(&userStates); err != nil || userStates != 3 {
		t.Fatalf("chat user-state backfill: count=%d err=%v", userStates, err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM direct_chat_read_states
		WHERE last_read_message_id = 30 AND unread_count = 0
	`).Scan(&directStates); err != nil || directStates != 2 {
		t.Fatalf("direct read-state backfill: count=%d err=%v", directStates, err)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM group_chat_read_states
		WHERE last_read_message_id = 31 AND unread_count = 0
	`).Scan(&groupStates); err != nil || groupStates != 2 {
		t.Fatalf("group read-state backfill: count=%d err=%v", groupStates, err)
	}
	var invitedStates int
	if err := db.QueryRow(`
		SELECT COUNT(*)
		FROM group_chat_read_states state
		JOIN group_memberships membership ON membership.id = state.membership_id
		WHERE membership.user_id = 3
	`).Scan(&invitedStates); err != nil || invitedStates != 0 {
		t.Fatalf("invited membership received read state: count=%d err=%v", invitedStates, err)
	}
	if _, err := db.Exec(`
		UPDATE direct_chat_read_states SET unread_count = -1
		WHERE user_id = 1 AND direct_conversation_id = 20
	`); err == nil {
		t.Fatal("expected direct unread_count check constraint")
	}
	for _, index := range []string{
		"idx_direct_chat_read_states_conversation_user",
		"idx_direct_chat_read_states_last_read",
		"idx_group_chat_read_states_last_read",
	} {
		var count int
		if err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?
		`, index).Scan(&count); err != nil || count != 1 {
			t.Fatalf("index %s: count=%d err=%v", index, count, err)
		}
	}
	if _, err := db.Exec(`DELETE FROM group_memberships WHERE id = 11`); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM group_chat_read_states WHERE membership_id = 11`).Scan(&groupStates); err != nil || groupStates != 0 {
		t.Fatalf("group read-state cascade: count=%d err=%v", groupStates, err)
	}
}

func TestMigration15PreservesCommentGraphAndAutoincrementUpAndDown(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migration-15.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Migrate(14); err != nil {
		t.Fatalf("migrate to version 14: %v", err)
	}
	_ = sourceDriver.Close()

	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES
			(1, 'comment-media-author@example.com', 'hash', 'Media', 'Author', '22-07-1992', 1, 1),
			(2, 'comment-media-commenter@example.com', 'hash', 'Media', 'Commenter', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	mediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (2, 'image/png', 8, 'comment.png', 'comment.png', 2)
	`)
	if err != nil {
		t.Fatalf("seed media: %v", err)
	}
	mediaID, _ := mediaResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO posts (id, author_user_id, text, privacy, created_at) VALUES (1, 1, 'post', 'public', 3)`); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_comments (id, post_id, author_user_id, text, created_at) VALUES (41, 1, 2, 'preserved', 4)`); err != nil {
		t.Fatalf("seed preserved comment: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_comments (id, post_id, author_user_id, text, created_at) VALUES (170, 1, 2, 'deleted high', 5)`); err != nil {
		t.Fatalf("seed high comment sequence: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM post_comments WHERE id = 170`); err != nil {
		t.Fatalf("delete high comment: %v", err)
	}

	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new version 15 migrator: %v", err)
	}
	if err := migrator.Migrate(15); err != nil {
		t.Fatalf("migrate to version 15: %v", err)
	}
	_ = sourceDriver.Close()
	assertMigrationVersion(t, db, 15, false)
	if _, exists := tableColumns(t, db, "post_comments")["media_id"]; !exists {
		t.Fatal("post_comments.media_id is missing after migration")
	}
	var (
		text      string
		createdAt int64
	)
	if err := db.QueryRow(`SELECT text, created_at FROM post_comments WHERE id = 41`).Scan(&text, &createdAt); err != nil || text != "preserved" || createdAt != 4 {
		t.Fatalf("preserved comment mismatch: text=%q created_at=%d err=%v", text, createdAt, err)
	}
	result, err := db.Exec(`
		INSERT INTO post_comments (post_id, author_user_id, text, media_id, created_at)
		VALUES (1, 2, 'after up', ?, 6)
	`, mediaID)
	if err != nil {
		t.Fatalf("insert comment after up: %v", err)
	}
	nextID, _ := result.LastInsertId()
	if nextID <= 170 {
		t.Fatalf("comment AUTOINCREMENT did not preserve deleted high ID after up: %d", nextID)
	}
	if _, err := db.Exec(`
		INSERT INTO post_comments (post_id, author_user_id, text, media_id, created_at)
		VALUES (1, 2, 'duplicate media', ?, 7)
	`, mediaID); err == nil {
		t.Fatal("expected partial unique comment media constraint")
	}
	if _, err := db.Exec(`UPDATE post_comments SET media_id = NULL WHERE id = ?`, nextID); err != nil {
		t.Fatalf("detach media before down: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_comments (id, post_id, author_user_id, text, created_at) VALUES (220, 1, 2, 'deleted high down', 8)`); err != nil {
		t.Fatalf("seed high down sequence: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM post_comments WHERE id = 220`); err != nil {
		t.Fatalf("delete high down comment: %v", err)
	}
	if err := guardCommentMediaDownMigration(db); err != nil {
		t.Fatalf("preflight version 15 down: %v", err)
	}
	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new down migrator: %v", err)
	}
	if err := migrator.Steps(-1); err != nil {
		t.Fatalf("migrate version 15 down: %v", err)
	}
	_ = sourceDriver.Close()
	assertMigrationVersion(t, db, 14, false)
	if _, exists := tableColumns(t, db, "post_comments")["media_id"]; exists {
		t.Fatal("post_comments.media_id remained after version 15 down")
	}
	result, err = db.Exec(`INSERT INTO post_comments (post_id, author_user_id, text, created_at) VALUES (1, 2, 'after down', 9)`)
	if err != nil {
		t.Fatalf("insert comment after down: %v", err)
	}
	nextID, _ = result.LastInsertId()
	if nextID <= 220 {
		t.Fatalf("comment AUTOINCREMENT did not preserve deleted high ID after down: %d", nextID)
	}
}

func TestMigration15DownRefusesCommentAttachmentsWithoutDirtyState(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "migration-15-down.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES (1, 'comment-down@example.com', 'hash', 'Comment', 'Down', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	mediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (1, 'image/png', 8, 'comment-down.png', 'comment-down.png', 2)
	`)
	if err != nil {
		t.Fatalf("seed media: %v", err)
	}
	mediaID, _ := mediaResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO posts (id, author_user_id, text, privacy, created_at) VALUES (1, 1, 'post', 'public', 3)`); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO post_comments (id, post_id, author_user_id, text, media_id, created_at)
		VALUES (1, 1, 1, 'attached', ?, 4)
	`, mediaID); err != nil {
		t.Fatalf("seed comment attachment: %v", err)
	}

	if err := migrateDown(db); err == nil {
		t.Fatal("expected down migration refusal")
	}
	assertMigrationVersion(t, db, 15, false)
	var gotMediaID int64
	if err := db.QueryRow(`SELECT media_id FROM post_comments WHERE id = 1`).Scan(&gotMediaID); err != nil || gotMediaID != mediaID {
		t.Fatalf("comment attachment changed after refused down: media=%d err=%v", gotMediaID, err)
	}
}

func TestMigration11DownRefusesGroupPostsWithoutDirtyState(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "migration-11-down.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Migrate(11); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
	_ = sourceDriver.Close()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES (1, 'down-owner@example.com', 'hash', 'Down', 'Owner', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO groups (id, owner_user_id, title, description, created_at) VALUES (1, 1, 'Down', 'Description', 1)`); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, author_user_id, group_id, text, created_at) VALUES (10, 1, 1, 'keep me', 1)`); err != nil {
		t.Fatalf("seed group post: %v", err)
	}

	if err := migrateDown(db); err == nil {
		t.Fatal("expected down migration refusal")
	}
	assertMigrationVersion(t, db, 11, false)
	var text string
	if err := db.QueryRow(`SELECT text FROM posts WHERE id = 10`).Scan(&text); err != nil || text != "keep me" {
		t.Fatalf("group post changed after refused down: text=%q err=%v", text, err)
	}
}

func TestMigration11DownPreservesPersonalPostGraph(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "migration-11-personal-down.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	migrator, sourceDriver, err := newMigrator(db)
	if err != nil {
		t.Fatalf("new version 11 migrator: %v", err)
	}
	if err := migrator.Migrate(11); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
	_ = sourceDriver.Close()
	if _, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
		VALUES
			(1, 'down-author@example.com', 'hash', 'Down', 'Author', '22-07-1992', 1, 1),
			(2, 'down-reader@example.com', 'hash', 'Down', 'Reader', '22-07-1992', 1, 1)
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, author_user_id, text, privacy, created_at) VALUES (31, 1, 'personal down', 'selected', 2)`); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_selected_users (post_id, user_id) VALUES (31, 2)`); err != nil {
		t.Fatalf("seed audience: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_comments (id, post_id, author_user_id, text, created_at) VALUES (61, 31, 2, 'down comment', 3)`); err != nil {
		t.Fatalf("seed comment: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO posts (id, author_user_id, text, privacy, created_at) VALUES (130, 1, 'deleted high down post', 'public', 4)`); err != nil {
		t.Fatalf("seed high down post sequence: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_comments (id, post_id, author_user_id, text, created_at) VALUES (190, 130, 2, 'deleted high down comment', 4)`); err != nil {
		t.Fatalf("seed high down comment sequence: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM posts WHERE id = 130`); err != nil {
		t.Fatalf("delete high down post while retaining sequences: %v", err)
	}
	if err := guardGroupPostsDownMigration(db); err != nil {
		t.Fatalf("preflight personal down: %v", err)
	}
	migrator, sourceDriver, err = newMigrator(db)
	if err != nil {
		t.Fatalf("new migrator: %v", err)
	}
	if err := migrator.Steps(-1); err != nil {
		t.Fatalf("migrate version 11 down: %v", err)
	}
	_ = sourceDriver.Close()
	assertMigrationVersion(t, db, 10, false)
	if _, exists := tableColumns(t, db, "posts")["group_id"]; exists {
		t.Fatal("posts.group_id remained after version 11 down")
	}
	var postID, audienceUserID, commentID int64
	if err := db.QueryRow(`SELECT id FROM posts WHERE id = 31 AND privacy = 'selected'`).Scan(&postID); err != nil || postID != 31 {
		t.Fatalf("personal post after down: id=%d err=%v", postID, err)
	}
	if err := db.QueryRow(`SELECT user_id FROM post_selected_users WHERE post_id = 31`).Scan(&audienceUserID); err != nil || audienceUserID != 2 {
		t.Fatalf("audience after down: user=%d err=%v", audienceUserID, err)
	}
	if err := db.QueryRow(`SELECT id FROM post_comments WHERE post_id = 31`).Scan(&commentID); err != nil || commentID != 61 {
		t.Fatalf("comment after down: id=%d err=%v", commentID, err)
	}
	postResult, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, created_at) VALUES (1, 'next down post', 'public', 5)`)
	if err != nil {
		t.Fatalf("insert post after down migration: %v", err)
	}
	nextPostID, _ := postResult.LastInsertId()
	if nextPostID <= 130 {
		t.Fatalf("post AUTOINCREMENT did not survive down migration: %d", nextPostID)
	}
	commentResult, err := db.Exec(`INSERT INTO post_comments (post_id, author_user_id, text, created_at) VALUES (?, 2, 'next down comment', 5)`, nextPostID)
	if err != nil {
		t.Fatalf("insert comment after down migration: %v", err)
	}
	nextCommentID, _ := commentResult.LastInsertId()
	if nextCommentID <= 190 {
		t.Fatalf("comment AUTOINCREMENT did not survive down migration: %d", nextCommentID)
	}
}

func TestGroupSchemaConstraintsIndexesAndCascades(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
			VALUES (?, 'hash', 'Group', 'User', '21-07-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	ownerID := insertUser("group-owner@example.com")
	memberID := insertUser("group-member@example.com")
	groupResult, err := db.Exec(`INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, 'Group', 'Description', 1)`, ownerID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	groupID, _ := groupResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'owner', 1, 1)`, groupID, ownerID); err != nil {
		t.Fatalf("insert owner membership: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'member', 2, 2)`, groupID, memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'invited', 3, 3)`, groupID, memberID); err == nil {
		t.Fatal("expected one membership state per group/user")
	}
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'invalid', 3, 3)`, groupID, insertUser("group-invalid@example.com")); err == nil {
		t.Fatal("expected membership status constraint")
	}
	for name, values := range map[string][2]string{
		"blank title":           {"   ", "Description"},
		"untrimmed title":       {" Group ", "Description"},
		"blank description":     {"Group", "   "},
		"untrimmed description": {"Group", " Description "},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec(`INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, ?, ?, 2)`, ownerID, values[0], values[1]); err == nil {
				t.Fatal("expected group text constraint")
			}
		})
	}
	for _, index := range []string{
		"idx_groups_created",
		"idx_group_memberships_group_status_updated",
		"idx_group_memberships_user_status_updated",
		"idx_group_memberships_user_status_created",
	} {
		if !schemaObjectExists(t, db, "index", index) {
			t.Fatalf("missing group index %s", index)
		}
	}
	if _, err := db.Exec(`DELETE FROM groups WHERE id = ?`, groupID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM group_memberships WHERE group_id = ?`, groupID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("group delete did not cascade memberships: count=%d err=%v", count, err)
	}
}

func TestNotificationSchemaConstraintsIndexesAndCascades(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "notifications.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
			VALUES (?, 'hash', 'Notification', 'User', '22-07-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user: %v", err)
		}
		id, _ := result.LastInsertId()
		if _, err := db.Exec(`INSERT INTO notification_user_states (user_id, revision) VALUES (?, 0)`, id); err != nil {
			t.Fatalf("insert notification state: %v", err)
		}
		return id
	}
	actorID := insertUser("notification-actor@example.com")
	recipientID := insertUser("notification-recipient@example.com")
	followResult, err := db.Exec(`
		INSERT INTO follows (follower_user_id, followed_user_id, status, created_at, updated_at)
		VALUES (?, ?, 'pending', 1, 1)
	`, actorID, recipientID)
	if err != nil {
		t.Fatalf("insert follow: %v", err)
	}
	followID, _ := followResult.LastInsertId()
	groupResult, err := db.Exec(`INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, 'Notifications', 'Description', 1)`, actorID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	groupID, _ := groupResult.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, follow_id, created_at)
		VALUES (?, ?, 'follow_request', ?, 2)
	`, recipientID, actorID, followID); err != nil {
		t.Fatalf("insert notification: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, follow_id, created_at)
		VALUES (?, ?, 'follow_request', ?, 3)
	`, recipientID, actorID, followID); err == nil {
		t.Fatal("expected one notification per follow lifecycle")
	}
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, follow_id, created_at)
		VALUES (?, ?, 'follow_started', ?, 3)
	`, recipientID, actorID, followID); err == nil {
		t.Fatal("expected follow request and follow-started to share one lifecycle slot")
	}
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, follow_id, resolution, resolved_at, created_at)
		VALUES (?, ?, 'follow_started', ?, 'accepted', 3, 3)
	`, recipientID, actorID, followID); err == nil {
		t.Fatal("expected non-actionable resolution constraint")
	}
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, follow_id, group_id, created_at)
		VALUES (?, ?, 'group_event', ?, ?, 3)
	`, recipientID, actorID, followID, groupID); err == nil {
		t.Fatal("expected source union constraint")
	}
	if _, err := db.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, type, created_at)
		VALUES (?, ?, 'follow_started', 3)
	`, recipientID, recipientID); err == nil {
		t.Fatal("expected actor and recipient to differ")
	}
	for _, index := range []string{
		"idx_notifications_recipient_created",
		"idx_notifications_recipient_unread",
		"idx_notifications_unique_follow_lifecycle",
		"idx_notifications_unique_event_recipient",
		"idx_notifications_unique_membership_lifecycle",
	} {
		if !schemaObjectExists(t, db, "index", index) {
			t.Fatalf("missing notification index %s", index)
		}
	}
	if _, err := db.Exec(`DELETE FROM follows WHERE id = ?`, followID); err != nil {
		t.Fatalf("delete follow: %v", err)
	}
	var nullableFollowID sql.NullInt64
	if err := db.QueryRow(`SELECT follow_id FROM notifications WHERE recipient_user_id = ?`, recipientID).Scan(&nullableFollowID); err != nil || nullableFollowID.Valid {
		t.Fatalf("follow lifecycle FK was not cleared: value=%+v err=%v", nullableFollowID, err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, recipientID); err != nil {
		t.Fatalf("delete recipient: %v", err)
	}
	var notificationRows, stateRows int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE recipient_user_id = ?`, recipientID).Scan(&notificationRows)
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_user_states WHERE user_id = ?`, recipientID).Scan(&stateRows)
	if notificationRows != 0 || stateRows != 0 {
		t.Fatalf("recipient cascade failed: notifications=%d state=%d", notificationRows, stateRows)
	}
}

func TestGroupEventSchemaConstraintsIndexesAndCascades(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
			VALUES (?, 'hash', 'Event', 'User', '22-07-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	ownerID := insertUser("event-schema-owner@example.com")
	memberID := insertUser("event-schema-member@example.com")
	groupResult, err := db.Exec(`INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, 'Events', 'Description', 1)`, ownerID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	groupID, _ := groupResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'owner', 1, 1)`, groupID, ownerID); err != nil {
		t.Fatalf("insert owner membership: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_memberships (group_id, user_id, status, created_at, updated_at) VALUES (?, ?, 'member', 1, 1)`, groupID, memberID); err != nil {
		t.Fatalf("insert member membership: %v", err)
	}
	eventResult, err := db.Exec(`
		INSERT INTO group_events (group_id, creator_user_id, title, description, starts_at, created_at)
		VALUES (?, ?, 'Planning', 'Plan the next meetup', 10, 1)
	`, groupID, memberID)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}
	eventID, _ := eventResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO group_event_responses (event_id, user_id, response, created_at, updated_at) VALUES (?, ?, 'going', 2, 2)`, eventID, memberID); err != nil {
		t.Fatalf("insert response: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO group_event_responses (event_id, user_id, response, created_at, updated_at) VALUES (?, ?, 'not_going', 3, 3)`, eventID, memberID); err == nil {
		t.Fatal("expected one response per event/user")
	}
	if _, err := db.Exec(`UPDATE group_event_responses SET response = 'maybe' WHERE event_id = ?`, eventID); err == nil {
		t.Fatal("expected response value constraint")
	}
	for name, values := range map[string][2]string{
		"blank title":           {"   ", "Description"},
		"untrimmed title":       {" Event ", "Description"},
		"blank description":     {"Event", "   "},
		"untrimmed description": {"Event", " Description "},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec(`INSERT INTO group_events (group_id, creator_user_id, title, description, starts_at, created_at) VALUES (?, ?, ?, ?, 10, 1)`, groupID, ownerID, values[0], values[1]); err == nil {
				t.Fatal("expected event text constraint")
			}
		})
	}
	for _, index := range []string{"idx_group_events_group_starts", "idx_group_event_responses_user_event"} {
		if !schemaObjectExists(t, db, "index", index) {
			t.Fatalf("missing group event index %s", index)
		}
	}

	if _, err := db.Exec(`DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?`, groupID, memberID); err != nil {
		t.Fatalf("delete membership: %v", err)
	}
	assertDBCount := func(table string, want int) {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
			t.Fatalf("%s count=%d err=%v want=%d", table, count, err, want)
		}
	}
	assertDBCount("group_events", 1)
	assertDBCount("group_event_responses", 1)
	if _, err := db.Exec(`DELETE FROM groups WHERE id = ?`, groupID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
	assertDBCount("group_events", 0)
	assertDBCount("group_event_responses", 0)
}

func TestChatSchemaConstraintsIndexesAndCascades(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	insertUser := func(email string) int64 {
		result, err := db.Exec("INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at) VALUES (?, 'hash', 'Chat', 'User', '22-07-1992', 1, 1)", email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	firstID := insertUser("chat-schema-first@example.com")
	secondID := insertUser("chat-schema-second@example.com")
	thirdID := insertUser("chat-schema-third@example.com")
	if _, err := db.Exec("INSERT INTO direct_conversations (user_low_id, user_high_id, created_at) VALUES (?, ?, 1)", firstID, firstID); err == nil {
		t.Fatal("expected normalized pair check")
	}
	conversationResult, err := db.Exec("INSERT INTO direct_conversations (user_low_id, user_high_id, created_at) VALUES (?, ?, 1)", firstID, secondID)
	if err != nil {
		t.Fatalf("insert direct conversation: %v", err)
	}
	conversationID, _ := conversationResult.LastInsertId()
	if _, err := db.Exec("INSERT INTO direct_conversations (user_low_id, user_high_id, created_at) VALUES (?, ?, 2)", firstID, secondID); err == nil {
		t.Fatal("expected unique direct pair")
	}
	groupResult, err := db.Exec("INSERT INTO groups (owner_user_id, title, description, created_at) VALUES (?, 'Chat group', 'Description', 1)", firstID)
	if err != nil {
		t.Fatalf("insert group: %v", err)
	}
	groupID, _ := groupResult.LastInsertId()

	validID := "47cd9266-b43f-4a89-9338-4f9c197ff12a"
	if _, err := db.Exec("INSERT INTO chat_messages (direct_conversation_id, sender_user_id, client_message_id, body, created_at) VALUES (?, ?, ?, 'hello', 1)", conversationID, firstID, validID); err != nil {
		t.Fatalf("insert direct message: %v", err)
	}
	if _, err := db.Exec("INSERT INTO chat_messages (sender_user_id, client_message_id, body, created_at) VALUES (?, 'missing-target', 'hello', 1)", firstID); err == nil {
		t.Fatal("expected exactly one chat target")
	}
	if _, err := db.Exec("INSERT INTO chat_messages (direct_conversation_id, group_id, sender_user_id, client_message_id, body, created_at) VALUES (?, ?, ?, 'two-targets', 'hello', 1)", conversationID, groupID, firstID); err == nil {
		t.Fatal("expected direct/group exclusivity")
	}
	for name, body := range map[string]string{
		"empty":      "",
		"whitespace": " ",
		"untrimmed":  " message ",
		"too long":   string(make([]byte, 2001)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec("INSERT INTO chat_messages (group_id, sender_user_id, client_message_id, body, created_at) VALUES (?, ?, ?, ?, 1)", groupID, firstID, "invalid-"+name, body); err == nil {
				t.Fatal("expected body constraint")
			}
		})
	}
	if _, err := db.Exec("INSERT INTO chat_messages (group_id, sender_user_id, client_message_id, body, created_at) VALUES (?, ?, ?, 'duplicate', 1)", groupID, firstID, validID); err == nil {
		t.Fatal("expected sender/client_message_id uniqueness")
	}
	if _, err := db.Exec("INSERT INTO chat_messages (group_id, sender_user_id, client_message_id, body, created_at) VALUES (?, ?, ?, 'same key other sender', 1)", groupID, thirdID, validID); err != nil {
		t.Fatalf("same client id for another sender: %v", err)
	}
	for _, index := range []string{
		"idx_chat_messages_direct_created",
		"idx_chat_messages_group_created",
		"idx_chat_messages_sender_client",
	} {
		if !schemaObjectExists(t, db, "index", index) {
			t.Fatalf("missing chat index %s", index)
		}
	}
	if _, err := db.Exec("DELETE FROM direct_conversations WHERE id = ?", conversationID); err != nil {
		t.Fatalf("delete direct conversation: %v", err)
	}
	var directMessages int
	if err := db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE direct_conversation_id = ?", conversationID).Scan(&directMessages); err != nil || directMessages != 0 {
		t.Fatalf("direct cascade count=%d err=%v", directMessages, err)
	}
	if _, err := db.Exec("DELETE FROM groups WHERE id = ?", groupID); err != nil {
		t.Fatalf("delete chat group: %v", err)
	}
	var groupMessages int
	if err := db.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE group_id = ?", groupID).Scan(&groupMessages); err != nil || groupMessages != 0 {
		t.Fatalf("group cascade count=%d err=%v", groupMessages, err)
	}
}

func TestUserPrivacyAndFollowConstraints(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (
				email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at
			) VALUES (?, 'hash', 'Test', 'User', '14-03-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("last insert ID: %v", err)
		}
		return id
	}
	firstID := insertUser("first@example.com")
	secondID := insertUser("second@example.com")

	var defaultPrivacy int
	if err := db.QueryRow(`SELECT is_private FROM users WHERE id = ?`, firstID).Scan(&defaultPrivacy); err != nil {
		t.Fatalf("query default privacy: %v", err)
	}
	if defaultPrivacy != 0 {
		t.Fatalf("new user must default to public, got %d", defaultPrivacy)
	}
	if _, err := db.Exec(`UPDATE users SET is_private = 2 WHERE id = ?`, firstID); err == nil {
		t.Fatal("expected invalid is_private value to fail")
	}

	if _, err := db.Exec(`
		INSERT INTO follows (follower_user_id, followed_user_id, status, created_at, updated_at)
		VALUES (?, ?, 'pending', 1, 1)
	`, firstID, secondID); err != nil {
		t.Fatalf("insert valid follow: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO follows (follower_user_id, followed_user_id, status, created_at, updated_at)
		VALUES (?, ?, 'accepted', 1, 1)
	`, firstID, secondID); err == nil {
		t.Fatal("expected duplicate follow relation to fail")
	}
	if _, err := db.Exec(`
		INSERT INTO follows (follower_user_id, followed_user_id, status, created_at, updated_at)
		VALUES (?, ?, 'pending', 1, 1)
	`, firstID, firstID); err == nil {
		t.Fatal("expected self-follow to fail")
	}
	if _, err := db.Exec(`
		INSERT INTO follows (follower_user_id, followed_user_id, status, created_at, updated_at)
		VALUES (?, ?, 'rejected', 1, 1)
	`, secondID, firstID); err == nil {
		t.Fatal("expected unsupported follow status to fail")
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, secondID); err != nil {
		t.Fatalf("delete followed user: %v", err)
	}
	var followCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM follows`).Scan(&followCount); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if followCount != 0 {
		t.Fatalf("user delete must cascade follows, got %d rows", followCount)
	}
}

func TestPostSchemaConstraintsAndIndexes(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	postColumns := tableColumns(t, db, "posts")
	for _, column := range []string{"id", "author_user_id", "text", "privacy", "media_id", "created_at"} {
		if _, exists := postColumns[column]; !exists {
			t.Fatalf("posts.%s is missing", column)
		}
	}
	if postColumns["text"].columnType != "TEXT" || postColumns["privacy"].columnType != "TEXT" || postColumns["created_at"].columnType != "INTEGER" {
		t.Fatalf("unexpected post storage types: %+v", postColumns)
	}
	for _, index := range []string{
		"idx_posts_created",
		"idx_posts_author_created",
		"idx_posts_media_unique",
		"idx_post_selected_users_user_post",
	} {
		if !schemaObjectExists(t, db, "index", index) {
			t.Fatalf("expected index %s", index)
		}
	}

	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (
				email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at
			) VALUES (?, 'hash', 'Post', 'User', '18-07-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	authorID := insertUser("post-schema-author@example.com")
	selectedID := insertUser("post-schema-selected@example.com")
	mediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (?, 'image/png', 8, 'post-schema.png', 'post.png', 1)
	`, authorID)
	if err != nil {
		t.Fatalf("insert media: %v", err)
	}
	mediaID, _ := mediaResult.LastInsertId()
	postResult, err := db.Exec(`
		INSERT INTO posts (author_user_id, text, privacy, media_id, created_at)
		VALUES (?, 'selected post', 'selected', ?, 1)
	`, authorID, mediaID)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
	postID, _ := postResult.LastInsertId()
	if _, err := db.Exec(`INSERT INTO post_selected_users (post_id, user_id) VALUES (?, ?)`, postID, selectedID); err != nil {
		t.Fatalf("insert selected audience: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO post_selected_users (post_id, user_id) VALUES (?, ?)`, postID, selectedID); err == nil {
		t.Fatal("expected duplicate selected audience to fail")
	}
	if _, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, media_id, created_at) VALUES (?, 'reuse media', 'public', ?, 2)`, authorID, mediaID); err == nil {
		t.Fatal("expected one media row to be used by at most one post")
	}
	for name, test := range map[string]struct {
		text    string
		privacy string
	}{
		"blank text":        {text: "   ", privacy: "public"},
		"untrimmed text":    {text: " post ", privacy: "public"},
		"unsupported state": {text: "post", privacy: "private"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, created_at) VALUES (?, ?, ?, 2)`, authorID, test.text, test.privacy); err == nil {
				t.Fatal("expected post constraint failure")
			}
		})
	}
	if _, err := db.Exec(`DELETE FROM posts WHERE id = ?`, postID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	var audienceCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM post_selected_users WHERE post_id = ?`, postID).Scan(&audienceCount); err != nil || audienceCount != 0 {
		t.Fatalf("post delete did not cascade audience: count=%d err=%v", audienceCount, err)
	}
}

func TestPostCommentSchemaConstraintsIndexesAndCascades(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	columns := tableColumns(t, db, "post_comments")
	for _, column := range []string{"id", "post_id", "author_user_id", "text", "media_id", "created_at"} {
		if _, exists := columns[column]; !exists {
			t.Fatalf("post_comments.%s is missing", column)
		}
	}
	if columns["text"].columnType != "TEXT" || columns["created_at"].columnType != "INTEGER" {
		t.Fatalf("unexpected comment storage types: %+v", columns)
	}
	if !schemaObjectExists(t, db, "index", "idx_post_comments_post_created") {
		t.Fatal("expected post comment pagination index")
	}
	if !schemaObjectExists(t, db, "index", "idx_post_comments_media") {
		t.Fatal("expected unique post comment media index")
	}

	insertUser := func(email string) int64 {
		result, err := db.Exec(`
			INSERT INTO users (email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at)
			VALUES (?, 'hash', 'Comment', 'User', '21-07-1992', 1, 1)
		`, email)
		if err != nil {
			t.Fatalf("insert user %s: %v", email, err)
		}
		id, _ := result.LastInsertId()
		return id
	}
	authorID := insertUser("comment-schema-author@example.com")
	commenterID := insertUser("comment-schema-commenter@example.com")
	postResult, err := db.Exec(`INSERT INTO posts (author_user_id, text, privacy, created_at) VALUES (?, 'post', 'public', 1)`, authorID)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
	postID, _ := postResult.LastInsertId()
	mediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (?, 'image/png', 8, 'comment-schema.png', 'comment-schema.png', 1)
	`, commenterID)
	if err != nil {
		t.Fatalf("insert comment media: %v", err)
	}
	mediaID, _ := mediaResult.LastInsertId()
	commentResult, err := db.Exec(`
		INSERT INTO post_comments (post_id, author_user_id, text, media_id, created_at)
		VALUES (?, ?, 'comment', ?, 1)
	`, postID, commenterID, mediaID)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}
	commentID, _ := commentResult.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO post_comments (post_id, author_user_id, text, media_id, created_at)
		VALUES (?, ?, 'duplicate attachment', ?, 1)
	`, postID, commenterID, mediaID); err == nil {
		t.Fatal("expected unique comment media constraint failure")
	}
	if _, err := db.Exec(`DELETE FROM media WHERE id = ?`, mediaID); err != nil {
		t.Fatalf("delete attached media: %v", err)
	}
	var linkedMediaID sql.NullInt64
	if err := db.QueryRow(`SELECT media_id FROM post_comments WHERE id = ?`, commentID).Scan(&linkedMediaID); err != nil || linkedMediaID.Valid {
		t.Fatalf("media delete did not clear comment link: media=%+v err=%v", linkedMediaID, err)
	}
	remainingMediaResult, err := db.Exec(`
		INSERT INTO media (owner_user_id, mime, size, storage_key, original_name, created_at)
		VALUES (?, 'image/png', 8, 'comment-cascade.png', 'comment-cascade.png', 1)
	`, commenterID)
	if err != nil {
		t.Fatalf("insert cascade media: %v", err)
	}
	remainingMediaID, _ := remainingMediaResult.LastInsertId()
	if _, err := db.Exec(`UPDATE post_comments SET media_id = ? WHERE id = ?`, remainingMediaID, commentID); err != nil {
		t.Fatalf("restore comment media link: %v", err)
	}
	for name, text := range map[string]string{"blank": "   ", "untrimmed": " comment "} {
		t.Run(name, func(t *testing.T) {
			if _, err := db.Exec(`INSERT INTO post_comments (post_id, author_user_id, text, created_at) VALUES (?, ?, ?, 2)`, postID, commenterID, text); err == nil {
				t.Fatal("expected comment text constraint failure")
			}
		})
	}
	if _, err := db.Exec(`DELETE FROM posts WHERE id = ?`, postID); err != nil {
		t.Fatalf("delete post: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM post_comments WHERE post_id = ?`, postID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("post delete did not cascade comments: count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM media WHERE id = ?`, remainingMediaID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("post delete unexpectedly removed media metadata: count=%d err=%v", count, err)
	}
}

func TestUsersDateOfBirthConstraintAcceptsOnlyRealDDMMYYYYDates(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	insert := func(email, dateOfBirth string) error {
		_, err := db.Exec(`
			INSERT INTO users (
				email, password_hash, first_name, last_name, date_of_birth, created_at, updated_at
			) VALUES (?, 'hash', 'Test', 'User', ?, 1, 1)
		`, email, dateOfBirth)
		return err
	}

	for index, value := range []string{"14-03-1992", "29-02-2000"} {
		if err := insert(fmt.Sprintf("valid-%d@example.com", index), value); err != nil {
			t.Fatalf("expected valid date_of_birth %q: %v", value, err)
		}
	}
	for index, value := range []string{
		"31-02-1992",
		"29-02-1900",
		"1992-03-14",
		"14/03/1992",
		"1-03-1992",
		"14-3-1992",
		"00-01-1992",
		"14-00-1992",
		"14-03-0000",
	} {
		if err := insert(fmt.Sprintf("invalid-%d@example.com", index), value); err == nil {
			t.Fatalf("expected SQLite to reject date_of_birth %q", value)
		}
	}
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

func schemaObjectExists(t *testing.T, db *sql.DB, objectType, name string) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?`, objectType, name).Scan(&count); err != nil {
		t.Fatalf("query schema object %s %s: %v", objectType, name, err)
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
