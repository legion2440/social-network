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
	for _, table := range []string{"chat_messages", "direct_conversations", "group_memberships", "groups", "post_comments", "post_selected_users", "posts", "follows", "sessions", "media", "users"} {
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

	for _, table := range []string{"users", "media", "sessions", "follows", "posts", "post_selected_users", "post_comments", "groups", "group_memberships", "direct_conversations", "chat_messages", "schema_migrations"} {
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
		{table: "direct_conversations", column: "created_at"},
		{table: "chat_messages", column: "created_at"},
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
	assertForeignKey(t, db, "groups", "owner_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "direct_conversations", "user_low_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "direct_conversations", "user_high_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "direct_conversation_id", "direct_conversations", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "chat_messages", "sender_user_id", "users", "id", "CASCADE")
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

	if err := migrateUp(db); err != nil {
		t.Fatalf("migrate to version 11: %v", err)
	}
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
	if nextID <= 41 {
		t.Fatalf("AUTOINCREMENT did not advance beyond migrated IDs: %d", nextID)
	}
}

func TestMigration11DownRefusesGroupPostsWithoutDirtyState(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "migration-11-down.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
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
	if err := guardGroupPostsDownMigration(db); err != nil {
		t.Fatalf("preflight personal down: %v", err)
	}
	migrator, sourceDriver, err := newMigrator(db)
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
	for _, column := range []string{"id", "post_id", "author_user_id", "text", "created_at"} {
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
	if _, err := db.Exec(`INSERT INTO post_comments (post_id, author_user_id, text, created_at) VALUES (?, ?, 'comment', 1)`, postID, commenterID); err != nil {
		t.Fatalf("insert comment: %v", err)
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
