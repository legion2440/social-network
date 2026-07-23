package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultHealthURL = "http://127.0.0.1:8080/api/health"
	healthTimeout    = 2 * time.Second
	maxHealthBody    = 4 << 10
)

type healthResponse struct {
	OK bool `json:"ok"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	client := &http.Client{Timeout: healthTimeout}
	if err := checkHealth(ctx, client, defaultHealthURL); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
		os.Exit(1)
	}
}

func checkHealth(ctx context.Context, client *http.Client, endpoint string) error {
	if ctx == nil || client == nil || endpoint == "" {
		return errors.New("healthcheck configuration is incomplete")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request health endpoint: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	decoder := json.NewDecoder(io.LimitReader(response.Body, maxHealthBody))
	var payload healthResponse
	if err := decoder.Decode(&payload); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("response contains trailing JSON")
		}
		return fmt.Errorf("decode trailing response: %w", err)
	}
	if !payload.OK {
		return errors.New("database is not healthy")
	}
	return nil
}
