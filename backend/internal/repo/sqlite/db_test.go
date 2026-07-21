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
	for _, table := range []string{"group_memberships", "groups", "post_comments", "post_selected_users", "posts", "follows", "sessions", "media", "users"} {
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

	for _, table := range []string{"users", "media", "sessions", "follows", "posts", "post_selected_users", "post_comments", "groups", "group_memberships", "schema_migrations"} {
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
	assertForeignKey(t, db, "posts", "media_id", "media", "id", "SET NULL")
	assertForeignKey(t, db, "post_selected_users", "post_id", "posts", "id", "CASCADE")
	assertForeignKey(t, db, "post_selected_users", "user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "post_comments", "post_id", "posts", "id", "CASCADE")
	assertForeignKey(t, db, "post_comments", "author_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "groups", "owner_user_id", "users", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "group_id", "groups", "id", "CASCADE")
	assertForeignKey(t, db, "group_memberships", "user_id", "users", "id", "CASCADE")
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
