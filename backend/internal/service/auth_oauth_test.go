package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"
)

func TestNormalizeOAuthNextAllowsOnlySameOriginRelativePaths(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  string
	}{
		{input: "/", want: "/"},
		{input: "/groups", want: "/groups"},
		{input: "/users/12", want: "/users/12"},
		{input: "/messages/direct/5", want: "/messages/direct/5"},
		{input: "https://example.com", want: "/"},
		{input: "//example.com", want: "/"},
		{input: "javascript:alert(1)", want: "/"},
		{input: `\example.com`, want: "/"},
		{input: `/\example.com`, want: "/"},
		{input: "/groups\nLocation: https://example.com", want: "/"},
	} {
		if got := normalizeOAuthNext(testCase.input); got != testCase.want {
			t.Fatalf("normalizeOAuthNext(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}

type cleanupTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *cleanupTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *cleanupTestClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type cleanupTestFlowRepo struct {
	mu          sync.Mutex
	deleteCalls int
	deleteErr   error
}

func (*cleanupTestFlowRepo) Create(context.Context, *domain.AuthFlow) error {
	return nil
}

func (*cleanupTestFlowRepo) GetByToken(context.Context, string) (*domain.AuthFlow, error) {
	return nil, repo.ErrNotFound
}

func (*cleanupTestFlowRepo) TakeByToken(context.Context, string) (*domain.AuthFlow, error) {
	return nil, repo.ErrNotFound
}

func (r *cleanupTestFlowRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleteCalls++
	return 0, r.deleteErr
}

func (r *cleanupTestFlowRepo) Calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.deleteCalls
}

func TestOAuthCleanupThrottleIsRaceSafeAndReportsErrors(t *testing.T) {
	cleanupErr := errors.New("cleanup failed")
	appClock := &cleanupTestClock{now: time.Unix(1_700_000_000, 0).UTC()}
	flows := &cleanupTestFlowRepo{deleteErr: cleanupErr}
	var errorMu sync.Mutex
	errorCalls := 0
	auth := &AuthService{
		authFlows: flows,
		clock:     appClock,
		oauthCleanupErrorHandler: func(err error) {
			if !errors.Is(err, cleanupErr) {
				t.Errorf("unexpected cleanup error: %v", err)
			}
			errorMu.Lock()
			errorCalls++
			errorMu.Unlock()
		},
	}

	var group sync.WaitGroup
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			auth.cleanupExpiredOAuthFlows(context.Background())
		}()
	}
	group.Wait()
	if calls := flows.Calls(); calls != 1 {
		t.Fatalf("concurrent cleanup calls=%d want=1", calls)
	}

	appClock.Advance(59 * time.Second)
	auth.cleanupExpiredOAuthFlows(context.Background())
	if calls := flows.Calls(); calls != 1 {
		t.Fatalf("cleanup retried before throttle elapsed: %d", calls)
	}

	appClock.Advance(time.Second)
	auth.cleanupExpiredOAuthFlows(context.Background())
	if calls := flows.Calls(); calls != 2 {
		t.Fatalf("cleanup calls after throttle=%d want=2", calls)
	}
	errorMu.Lock()
	defer errorMu.Unlock()
	if errorCalls != 2 {
		t.Fatalf("cleanup error callbacks=%d want=2", errorCalls)
	}
}
