package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

var (
	testBirthDate = "14-03-1992"
	testCreatedAt = time.Date(2026, time.July, 13, 8, 30, 15, 0, time.UTC)
	testUpdatedAt = time.Date(2026, time.July, 13, 9, 45, 20, 0, time.UTC)
)

func TestUserRepoCreatesAndReadsFullUser(t *testing.T) {
	db := openUserTestDB(t)
	repository := NewUserRepo(db)
	gender := domain.GenderMale
	nickname := "  comet  "
	aboutMe := "  Learning Go and distributed systems.  "
	user := &domain.User{
		Email:        "  Alice@example.com  ",
		PasswordHash: "bcrypt-hash",
		FirstName:    "  Alice  ",
		LastName:     "  Example  ",
		DateOfBirth:  testBirthDate,
		Gender:       &gender,
		Nickname:     &nickname,
		AboutMe:      &aboutMe,
		CreatedAt:    testCreatedAt,
		UpdatedAt:    testUpdatedAt,
	}

	userID, err := repository.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	got, err := repository.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get user by ID: %v", err)
	}
	assertUserCore(t, got, domain.GenderMale, "comet", "Learning Go and distributed systems.")
	if got.Email != "Alice@example.com" || got.FirstName != "Alice" || got.LastName != "Example" {
		t.Fatalf("unexpected normalized user: %+v", got)
	}
	if got.AvatarMediaID != nil {
		t.Fatalf("avatar_media_id must default to NULL, got %d", *got.AvatarMediaID)
	}
	if got.DateOfBirth != testBirthDate || !got.CreatedAt.Equal(testCreatedAt) || !got.UpdatedAt.Equal(testUpdatedAt) {
		t.Fatalf("date round trip failed: birth=%v created=%v updated=%v", got.DateOfBirth, got.CreatedAt, got.UpdatedAt)
	}

	byEmail, err := repository.GetByEmail(context.Background(), "ALICE@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("get user by case-insensitive email: %v", err)
	}
	if byEmail.ID != userID {
		t.Fatalf("expected user %d, got %d", userID, byEmail.ID)
	}
}

func TestUserRepoStoresOptionalFieldsAsNull(t *testing.T) {
	db := openUserTestDB(t)
	repository := NewUserRepo(db)
	userID := createTestUser(t, repository, "minimal@example.com", nil)

	got, err := repository.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if got.Gender != nil || got.Nickname != nil || got.AboutMe != nil || got.AvatarMediaID != nil {
		t.Fatalf("optional fields must be NULL: %+v", got)
	}
}

func TestUserRepoStoresSupportedGenders(t *testing.T) {
	for _, testCase := range []struct {
		name string
		in   domain.Gender
	}{
		{name: "male", in: domain.GenderMale},
		{name: "female", in: domain.GenderFemale},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db := openUserTestDB(t)
			repository := NewUserRepo(db)
			userID := createTestUser(t, repository, testCase.name+"@example.com", &testCase.in)
			got, err := repository.GetByID(context.Background(), userID)
			if err != nil {
				t.Fatalf("get user: %v", err)
			}
			if got.Gender == nil || *got.Gender != testCase.in {
				t.Fatalf("expected gender %q, got %v", testCase.in, got.Gender)
			}
		})
	}
}

func TestUserRepoRejectsInvalidGender(t *testing.T) {
	for _, value := range []domain.Gender{"", "unknown", " male "} {
		t.Run(string(value), func(t *testing.T) {
			db := openUserTestDB(t)
			repository := NewUserRepo(db)
			user := newTestUser("invalid-gender@example.com", &value)

			if _, err := repository.Create(context.Background(), user); !errors.Is(err, domain.ErrInvalidGender) {
				t.Fatalf("expected invalid gender error for %q, got %v", value, err)
			}
			assertRowCount(t, db, "users", 0)
		})
	}
}

func TestUserRepoRejectsInvalidDateOfBirth(t *testing.T) {
	for _, value := range []string{"", "1992-03-14", "14/03/1992", "31-02-1992"} {
		t.Run(value, func(t *testing.T) {
			db := openUserTestDB(t)
			repository := NewUserRepo(db)
			user := newTestUser("invalid-date@example.com", nil)
			user.DateOfBirth = value

			if _, err := repository.Create(context.Background(), user); !errors.Is(err, domain.ErrInvalidDateOfBirth) {
				t.Fatalf("expected invalid date_of_birth error for %q, got %v", value, err)
			}
			assertRowCount(t, db, "users", 0)
		})
	}
}

func TestUserRepoRejectsDuplicateEmailIgnoringCase(t *testing.T) {
	db := openUserTestDB(t)
	repository := NewUserRepo(db)
	createTestUser(t, repository, "duplicate@example.com", nil)

	duplicate := newTestUser("DUPLICATE@EXAMPLE.COM", nil)
	if _, err := repository.Create(context.Background(), duplicate); !errors.Is(err, repo.ErrConflict) {
		t.Fatalf("expected case-insensitive conflict error, got %v", err)
	}
}

func TestUserMediaAvatarRelationAndUserDeleteCascades(t *testing.T) {
	db := openUserTestDB(t)
	users := NewUserRepo(db)
	media := NewMediaRepo(db)
	sessions := NewSessionRepo(db)
	userID := createTestUser(t, users, "avatar@example.com", nil)

	mediaID, err := media.Create(
		context.Background(),
		userID,
		"image/png",
		128,
		"avatar-storage-key.png",
		"avatar.png",
		testCreatedAt,
	)
	if err != nil {
		t.Fatalf("create media: %v", err)
	}
	if err := users.SetAvatarMediaID(context.Background(), userID, &mediaID, testUpdatedAt); err != nil {
		t.Fatalf("link avatar media: %v", err)
	}
	got, err := users.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get user with avatar: %v", err)
	}
	if got.AvatarMediaID == nil || *got.AvatarMediaID != mediaID {
		t.Fatalf("expected avatar media %d, got %v", mediaID, got.AvatarMediaID)
	}
	if _, err := db.Exec(`DELETE FROM media WHERE id = ?`, mediaID); err != nil {
		t.Fatalf("delete avatar media: %v", err)
	}
	got, err = users.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("get user after avatar delete: %v", err)
	}
	if got.AvatarMediaID != nil {
		t.Fatalf("avatar reference must be NULL after media delete, got %d", *got.AvatarMediaID)
	}

	mediaID, err = media.Create(
		context.Background(),
		userID,
		"image/png",
		256,
		"second-avatar-storage-key.png",
		"second-avatar.png",
		testUpdatedAt,
	)
	if err != nil {
		t.Fatalf("create second media: %v", err)
	}
	if err := users.SetAvatarMediaID(context.Background(), userID, &mediaID, testUpdatedAt); err != nil {
		t.Fatalf("link second avatar media: %v", err)
	}

	if err := sessions.Create(context.Background(), &domain.Session{
		Token:     "cascade-session",
		UserID:    userID,
		ExpiresAt: testCreatedAt.Add(24 * time.Hour),
		CreatedAt: testCreatedAt,
	}); err != nil {
		t.Fatalf("create session: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	assertRowCount(t, db, "media", 0)
	assertRowCount(t, db, "sessions", 0)
}

func openUserTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func createTestUser(t *testing.T, repository *UserRepo, email string, gender *domain.Gender) int64 {
	t.Helper()
	userID, err := repository.Create(context.Background(), newTestUser(email, gender))
	if err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return userID
}

func newTestUser(email string, gender *domain.Gender) *domain.User {
	return &domain.User{
		Email:        email,
		PasswordHash: "bcrypt-hash",
		FirstName:    "Test",
		LastName:     "User",
		DateOfBirth:  testBirthDate,
		Gender:       gender,
		CreatedAt:    testCreatedAt,
		UpdatedAt:    testUpdatedAt,
	}
}

func assertUserCore(t *testing.T, user *domain.User, gender domain.Gender, nickname, aboutMe string) {
	t.Helper()
	if user.Gender == nil || *user.Gender != gender {
		t.Fatalf("expected gender %q, got %v", gender, user.Gender)
	}
	if user.Nickname == nil || *user.Nickname != nickname {
		t.Fatalf("expected nickname %q, got %v", nickname, user.Nickname)
	}
	if user.AboutMe == nil || *user.AboutMe != aboutMe {
		t.Fatalf("expected about_me %q, got %v", aboutMe, user.AboutMe)
	}
}

func assertRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("expected %d rows in %s, got %d", want, table, count)
	}
}
