package main

import (
	"os"
	"testing"
)

func TestLoadConfig_RequiresDatabaseURL(t *testing.T) {
	os.Setenv("JWT_SECRET", "test")
	os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("JWT_SECRET")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when DATABASE_URL is missing")
	}
}

func TestLoadConfig_RequiresJWTSecret(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DATABASE_URL")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET is missing")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost/test")
	os.Setenv("JWT_SECRET", "secret")
	defer os.Unsetenv("DATABASE_URL")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.S3Bucket != "screenspace" {
		t.Errorf("expected default bucket screenspace, got %s", cfg.S3Bucket)
	}
}
