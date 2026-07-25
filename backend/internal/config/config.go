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
	FrontendDir     string
	CookieSecure    bool
	SessionTTL      time.Duration
	ShutdownTimeout time.Duration
	GitHubOAuth     GitHubOAuthConfig
}

type GitHubOAuthConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenvOrDefault("SOCIAL_NETWORK_HTTP_ADDR", "127.0.0.1:8080"),
		DBPath:          getenvOrDefault("SOCIAL_NETWORK_DB_PATH", "var/social-network.db"),
		UploadDir:       getenvOrDefault("SOCIAL_NETWORK_UPLOAD_DIR", "var/uploads"),
		FrontendDir:     getenvOrDefault("SOCIAL_NETWORK_FRONTEND_DIR", "../frontend/dist"),
		SessionTTL:      24 * time.Hour,
		ShutdownTimeout: 10 * time.Second,
	}

	secure, err := boolEnv("SOCIAL_NETWORK_COOKIE_SECURE", false)
	if err != nil {
		return Config{}, err
	}
	cfg.CookieSecure = secure

	githubValues := []string{
		strings.TrimSpace(os.Getenv("SOCIAL_NETWORK_GITHUB_CLIENT_ID")),
		strings.TrimSpace(os.Getenv("SOCIAL_NETWORK_GITHUB_CLIENT_SECRET")),
		strings.TrimSpace(os.Getenv("SOCIAL_NETWORK_GITHUB_REDIRECT_URL")),
	}
	configured := 0
	for _, value := range githubValues {
		if value != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(githubValues) {
		return Config{}, fmt.Errorf(
			"SOCIAL_NETWORK_GITHUB_CLIENT_ID, SOCIAL_NETWORK_GITHUB_CLIENT_SECRET and SOCIAL_NETWORK_GITHUB_REDIRECT_URL must be set together",
		)
	}
	if configured == len(githubValues) {
		cfg.GitHubOAuth = GitHubOAuthConfig{
			Enabled:      true,
			ClientID:     githubValues[0],
			ClientSecret: githubValues[1],
			RedirectURL:  githubValues[2],
		}
	}

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
