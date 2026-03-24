package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

func main() {
	log.SetFlags(log.Ltime)

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3Bucket := envOr("S3_BUCKET", "screenspace")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}
	adminEmail := envOr("ADMIN_EMAIL", "admin@screenspace.dev")

	// Database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("database ping: %v", err)
	}

	// S3 Storage
	store, err := storage.NewS3Store(s3Endpoint, s3Bucket, s3AccessKey, s3SecretKey)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	if err := store.EnsureBucket(context.Background()); err != nil {
		log.Printf("warning: could not ensure bucket: %v", err)
	}

	log.Printf("seeding %d wallpapers from Pexels CDN (ffmpeg fallback if download fails)", len(videos))

	s := &seeder{
		db:           db,
		store:        store,
		authService:  service.NewAuthService(jwtSecret),
		videoService: service.NewVideoService(),
	}

	ctx := context.Background()
	if err := s.run(ctx, adminEmail); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Println("seed complete")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
