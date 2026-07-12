package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const SessionCookieName = "social_network_session"

type Config struct {
	HTTPAddr        string
	DBPath          string
	UploadDir       string
	CookieSecure    bool
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenvOrDefault("SOCIAL_NETWORK_HTTP_ADDR", "127.0.0.1:8080"),
		DBPath:          getenvOrDefault("SOCIAL_NETWORK_DB_PATH", "var/social-network.db"),
		UploadDir:       getenvOrDefault("SOCIAL_NETWORK_UPLOAD_DIR", "var/uploads"),
		SessionTTL:      24 * time.Hour,
		ShutdownTimeout: 10 * time.Second,
	}

	secure, err := boolEnv("SOCIAL_NETWORK_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	cfg.CookieSecure = secure

	return cfg, nil
}

func getenvOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
