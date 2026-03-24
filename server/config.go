package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	ShutdownTimeout     time.Duration
	DatabaseURL         string
	DBMaxConns          int
	DBMinConns          int
	DBMaxConnLifetime   time.Duration
	DBHealthCheckPeriod time.Duration
	S3Endpoint          string
	S3Bucket            string
	S3AccessKey         string
	S3SecretKey         string
	JWTSecret           string
	AdminEmail          string
	UploadRateLimit     int
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		ShutdownTimeout:     25 * time.Second,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DBMaxConns:          envInt("DB_MAX_CONNS", 25),
		DBMinConns:          envInt("DB_MIN_CONNS", 5),
		DBMaxConnLifetime:   5 * time.Minute,
		DBHealthCheckPeriod: 30 * time.Second,
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Bucket:            getEnv("S3_BUCKET", "screenspace"),
		S3AccessKey:         os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:         os.Getenv("S3_SECRET_KEY"),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		AdminEmail:          os.Getenv("ADMIN_EMAIL"),
		UploadRateLimit:     envInt("UPLOAD_RATE_LIMIT", 5),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
