package service_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

func TestGroupEventCommitFailuresLeaveNoEventOrResponse(t *testing.T) {
	db, err := sqlite.Open(context.Background(), filepath.Join(t.TempDir(), "social-network.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	now := time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)
	users := sqlite.NewUserRepo(db)
	ownerID := createPostTestUser(t, ctx, users, "event-rollback-owner@example.com", now)
	transactions := sqlite.NewTransactionManager(db)
	group, err := service.NewGroupService(transactions, fixedPostClock{now: now}).Create(ctx, ownerID, "Rollback events", "Description")
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	failing := service.NewGroupEventService(failAfterCallbackTransactions{delegate: transactions}, fixedPostClock{now: now})
	event, err := failing.Create(ctx, ownerID, group.ID, service.CreateGroupEventInput{
		Title: "Rolled back", Description: "This event must not remain", StartsAt: now.Add(time.Hour),
	})
	if !errors.Is(err, errForcedCommitFailure) || event != nil {
		t.Fatalf("create commit failure: event=%+v err=%v", event, err)
	}
	assertServiceTableCount(t, db, "group_events", 0)

	working := service.NewGroupEventService(transactions, fixedPostClock{now: now})
	event, err = working.Create(ctx, ownerID, group.ID, service.CreateGroupEventInput{
		Title: "Persisted", Description: "This event remains", StartsAt: now.Add(2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create persisted event: %v", err)
	}
	responded, err := failing.Respond(ctx, ownerID, group.ID, event.ID, domain.GroupEventGoing)
	if !errors.Is(err, errForcedCommitFailure) || responded != nil {
		t.Fatalf("response commit failure: event=%+v err=%v", responded, err)
	}
	assertServiceTableCount(t, db, "group_event_responses", 0)
}

func assertServiceTableCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil || count != want {
		t.Fatalf("%s count=%d err=%v want=%d", table, count, err, want)
	}
}
