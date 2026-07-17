package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsUseBackendRelativeRuntimePaths(t *testing.T) {
	t.Setenv("SOCIAL_NETWORK_HTTP_ADDR", "")
	t.Setenv("SOCIAL_NETWORK_DB_PATH", "")
	t.Setenv("SOCIAL_NETWORK_UPLOAD_DIR", "")
	t.Setenv("SOCIAL_NETWORK_FRONTEND_DIR", "")
	t.Setenv("SOCIAL_NETWORK_COOKIE_SECURE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("unexpected HTTP address %q", cfg.HTTPAddr)
	}
	if filepath.IsAbs(cfg.DBPath) || filepath.Clean(cfg.DBPath) != filepath.Join("var", "social-network.db") {
		t.Fatalf("unexpected DB path %q", cfg.DBPath)
	}
	if filepath.IsAbs(cfg.UploadDir) || filepath.Clean(cfg.UploadDir) != filepath.Join("var", "uploads") {
		t.Fatalf("unexpected upload path %q", cfg.UploadDir)
	}
	if filepath.IsAbs(cfg.FrontendDir) || filepath.Clean(cfg.FrontendDir) != filepath.Join("..", "frontend") {
		t.Fatalf("unexpected frontend path %q", cfg.FrontendDir)
	}
	if cfg.CookieSecure {
		t.Fatal("local HTTP cookie must not be secure by default")
	}
}

func TestLoadUsesConfiguredFrontendDir(t *testing.T) {
	t.Setenv("SOCIAL_NETWORK_FRONTEND_DIR", "./web")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if filepath.Clean(cfg.FrontendDir) != "web" {
		t.Fatalf("unexpected frontend path %q", cfg.FrontendDir)
	}
}

func TestLoadRejectsInvalidCookieSecureValue(t *testing.T) {
	t.Setenv("SOCIAL_NETWORK_COOKIE_SECURE", "sometimes")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid boolean error")
	}
}
