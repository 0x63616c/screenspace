package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/video"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3Bucket := envOr("S3_BUCKET", "screenspace")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Error("JWT_SECRET is required")
		os.Exit(1)
	}
	adminEmail := envOr("ADMIN_EMAIL", "admin@screenspace.dev")

	ctx := context.Background()

	// Database
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping", "error", err)
		os.Exit(1)
	}

	// S3 Storage
	store, err := storage.NewS3Store(s3Endpoint, s3Bucket, s3AccessKey, s3SecretKey)
	if err != nil {
		slog.Error("storage", "error", err)
		os.Exit(1)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		slog.Warn("could not ensure bucket", "error", err)
	}

	slog.Info("seeding wallpapers from Pexels CDN (ffmpeg fallback if download fails)", "count", len(videos))

	cfg := &config.Config{
		JWTSecret:  jwtSecret,
		BcryptCost: 10,
	}

	s := &seeder{
		pool:        pool,
		store:       store,
		authService: service.NewAuthService(cfg),
		prober:      video.NewFFProber(),
	}

	if err := s.run(ctx, adminEmail); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}

	slog.Info("seed complete")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
