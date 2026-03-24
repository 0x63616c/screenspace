package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/video"
)

type seeder struct {
	pool        *pgxpool.Pool
	store       storage.Store
	authService *service.AuthService
	prober      video.Prober
}

func (s *seeder) run(ctx context.Context, adminEmail string) error {
	// Seed users
	adminID, err := s.seedUser(ctx, adminEmail, "password", "admin")
	if err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	slog.Info("admin user", "email", adminEmail, "id", adminID)

	userEmail := "user@screenspace.dev"
	userID, err := s.seedUser(ctx, userEmail, "password", "user")
	if err != nil {
		return fmt.Errorf("seed regular user: %w", err)
	}
	slog.Info("regular user", "email", userEmail, "id", userID)

	// Seed wallpapers
	var wallpaperIDs []string
	for _, v := range videos {
		id, err := s.seedWallpaper(ctx, v, adminID)
		if err != nil {
			slog.Warn("failed to seed wallpaper", "title", v.Title, "error", err)
			continue
		}
		wallpaperIDs = append(wallpaperIDs, id)
		slog.Info("seeded wallpaper", "title", v.Title)
	}

	// Add favorites for the regular user (first 3 wallpapers)
	for i, wpID := range wallpaperIDs {
		if i >= 3 {
			break
		}
		if err := s.addFavorite(ctx, userID, wpID); err != nil {
			slog.Warn("failed to add favorite", "error", err)
		}
	}
	slog.Info("added favorites", "count", min(3, len(wallpaperIDs)), "user", userEmail)

	return nil
}

func (s *seeder) seedUser(ctx context.Context, email, password, role string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1`, email,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("check user: %w", err)
	}

	hash, err := s.authService.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	err = s.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3) RETURNING id`,
		email, hash, role,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert user: %w", err)
	}
	return id, nil
}

func (s *seeder) seedWallpaper(ctx context.Context, v SeedVideo, uploaderID string) (string, error) {
	// Idempotency check
	var existingID string
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM wallpapers WHERE title = $1`, v.Title,
	).Scan(&existingID)
	if err == nil {
		slog.Info("skipping (already exists)", "title", v.Title)
		return existingID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("check existing: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "seed-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	videoPath := filepath.Join(tmpDir, "video.mp4")

	// Download from Pexels CDN, fall back to generated video
	if err := downloadFile(ctx, v.URL, videoPath); err != nil {
		slog.Warn("download failed, generating placeholder", "title", v.Title, "error", err)
		if err := generateVideo(ctx, v, videoPath); err != nil {
			return "", fmt.Errorf("generate video: %w", err)
		}
	}

	// Generate thumbnail and preview
	thumbPath := filepath.Join(tmpDir, "thumb.jpg")
	previewPath := filepath.Join(tmpDir, "preview.mp4")

	if err := s.prober.GenerateThumbnail(ctx, videoPath, thumbPath); err != nil {
		return "", fmt.Errorf("generate thumbnail: %w", err)
	}
	if err := s.prober.GeneratePreview(ctx, videoPath, previewPath); err != nil {
		return "", fmt.Errorf("generate preview: %w", err)
	}

	// Probe actual video metadata
	info, err := s.prober.Probe(ctx, videoPath)
	if err != nil {
		return "", fmt.Errorf("probe video: %w", err)
	}

	// Upload to S3
	wpID := uuid.New().String()
	prefix := fmt.Sprintf("wallpapers/%s", wpID)

	storageKey := prefix + "/original.mp4"
	thumbKey := prefix + "/thumb.jpg"
	previewKey := prefix + "/preview.mp4"

	if err := s.uploadFile(ctx, storageKey, videoPath, "video/mp4"); err != nil {
		return "", fmt.Errorf("upload video: %w", err)
	}
	if err := s.uploadFile(ctx, thumbKey, thumbPath, "image/jpeg"); err != nil {
		return "", fmt.Errorf("upload thumbnail: %w", err)
	}
	if err := s.uploadFile(ctx, previewKey, previewPath, "video/mp4"); err != nil {
		return "", fmt.Errorf("upload preview: %w", err)
	}

	resolution := fmt.Sprintf("%dx%d", info.Width, info.Height)
	format := info.Format
	if format == "" {
		format = "h264"
	}

	var insertedID string
	err = s.pool.QueryRow(ctx,
		`INSERT INTO wallpapers (id, title, uploader_id, status, category, tags, resolution, width, height, duration, file_size, format, download_count, storage_key, thumbnail_key, preview_key)
		 VALUES ($1, $2, $3, 'approved', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id`,
		wpID, v.Title, uploaderID, v.Category, v.Tags,
		resolution, info.Width, info.Height, info.Duration, info.Size, format,
		v.Downloads, storageKey, thumbKey, previewKey,
	).Scan(&insertedID)
	if err != nil {
		return "", fmt.Errorf("insert wallpaper: %w", err)
	}

	return insertedID, nil
}

func (s *seeder) uploadFile(ctx context.Context, key, filePath, contentType string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return s.store.Put(ctx, key, f, contentType)
}

func (s *seeder) addFavorite(ctx context.Context, userID, wallpaperID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO favorites (user_id, wallpaper_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, wallpaperID,
	)
	return err
}

func downloadFile(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d for %s", resp.StatusCode, url)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, resp.Body)
	return err
}

// generateVideo creates a placeholder video with ffmpeg when CDN download fails.
func generateVideo(ctx context.Context, v SeedVideo, outputPath string) error {
	bg := categoryColor(v.Category)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=%s:s=1920x1080:d=10:r=24", bg),
		"-vf", fmt.Sprintf(
			"drawtext=text='%s':fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2-40:shadowcolor=black:shadowx=2:shadowy=2,"+
				"drawtext=text='%s':fontsize=28:fontcolor=white@0.7:x=(w-text_w)/2:y=(h-text_h)/2+30:shadowcolor=black:shadowx=1:shadowy=1",
			strings.ReplaceAll(v.Title, "'", ""),
			strings.ToUpper(v.Category),
		),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		outputPath,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		// drawtext may not be available, try without text overlay
		cmd2 := exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-f", "lavfi",
			"-i", fmt.Sprintf("color=c=%s:s=1920x1080:d=10:r=24", bg),
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-pix_fmt", "yuv420p",
			outputPath,
		)
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("ffmpeg generate: %w: %s", err2, string(out2))
		}
		return nil
	}
	_ = out
	return nil
}

func categoryColor(category string) string {
	switch category {
	case "nature":
		return "0x2D5016"
	case "abstract":
		return "0x6B1D7B"
	case "space":
		return "0x0D0D2B"
	case "urban":
		return "0x2C3E50"
	case "underwater":
		return "0x004D6B"
	default:
		return "0x333333"
	}
}
