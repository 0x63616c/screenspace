package config_test

import (
	"testing"

	"github.com/0x63616c/screenspace/server/internal/config"
)

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("S3_ENDPOINT", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_KEY", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}

func TestLoad_JWTSecretTooShort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "tooshort")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET, got nil")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "a-very-long-secret-that-is-at-least-32-chars!!")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DBMaxConns != 25 {
		t.Errorf("expected default DBMaxConns 25, got %d", cfg.DBMaxConns)
	}
	if cfg.MaxFileSize != 200*1024*1024 {
		t.Errorf("expected default MaxFileSize 200MB, got %d", cfg.MaxFileSize)
	}
	if cfg.UploadRateLimit != 5 {
		t.Errorf("expected default UploadRateLimit 5, got %d", cfg.UploadRateLimit)
	}
}

func TestLoad_OverrideDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("JWT_SECRET", "a-very-long-secret-that-is-at-least-32-chars!!")
	t.Setenv("S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("S3_ACCESS_KEY", "key")
	t.Setenv("S3_SECRET_KEY", "secret")
	t.Setenv("PORT", "9090")
	t.Setenv("DB_MAX_CONNS", "50")
	t.Setenv("UPLOAD_RATE_LIMIT", "10")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DBMaxConns != 50 {
		t.Errorf("expected DBMaxConns 50, got %d", cfg.DBMaxConns)
	}
	if cfg.UploadRateLimit != 10 {
		t.Errorf("expected UploadRateLimit 10, got %d", cfg.UploadRateLimit)
	}
}
