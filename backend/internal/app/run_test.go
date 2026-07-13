package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"social-network/backend/internal/config"
)

func TestRunWithContextShutsDownCleanly(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runWithContext(ctx, config.Config{
			HTTPAddr:        "127.0.0.1:0",
			DBPath:          filepath.Join(root, "social-network.db"),
			UploadDir:       filepath.Join(root, "uploads"),
			SessionTTL:      24 * time.Hour,
			ShutdownTimeout: 2 * time.Second,
		})
	}()

	deadline := time.Now().Add(5 * time.Second)
	started := false
	for time.Now().Before(deadline) {
		if _, err := os.Stat(filepath.Join(root, "social-network.db")); err == nil {
			started = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if !started {
		t.Fatal("database bootstrap did not start")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run with cancellation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after cancellation")
	}
}
