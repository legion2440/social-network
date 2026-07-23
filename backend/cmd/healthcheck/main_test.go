package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCheckHealthAcceptsHealthyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	if err := checkHealth(context.Background(), server.Client(), server.URL); err != nil {
		t.Fatalf("check health: %v", err)
	}
}

func TestCheckHealthRejectsUnavailableResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"ok":false}`))
	}))
	t.Cleanup(server.Close)

	err := checkHealth(context.Background(), server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestCheckHealthRejectsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":`))
	}))
	t.Cleanup(server.Close)

	if err := checkHealth(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestCheckHealthHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := checkHealth(ctx, server.Client(), server.URL); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCheckHealthRejectsConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	endpoint := "http://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := checkHealth(ctx, &http.Client{Timeout: time.Second}, endpoint); err == nil {
		t.Fatal("expected connection refused error")
	}
}
