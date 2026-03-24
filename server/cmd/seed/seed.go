package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

type seeder struct {
	db           *sql.DB
	store        storage.Store
	authService  *service.AuthService
	videoService *service.VideoService
	pexelsKey    string
}

func (s *seeder) run(ctx context.Context, adminEmail string) error {
	// Seed users
	adminID, err := s.seedUser(ctx, adminEmail, "password", "admin")
	if err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	log.Printf("admin user: %s (%s)", adminEmail, adminID)

	userEmail := "user@screenspace.dev"
	userID, err := s.seedUser(ctx, userEmail, "password", "user")
	if err != nil {
		return fmt.Errorf("seed regular user: %w", err)
	}
	log.Printf("regular user: %s (%s)", userEmail, userID)

	// Seed wallpapers
	var wallpaperIDs []string
	for _, v := range videos {
		id, err := s.seedWallpaper(ctx, v, adminID)
		if err != nil {
			log.Printf("warning: failed to seed %q: %v", v.Title, err)
			continue
		}
		wallpaperIDs = append(wallpaperIDs, id)
		log.Printf("seeded wallpaper: %s", v.Title)
	}

	// Add favorites for the regular user (first 3 wallpapers)
	for i, wpID := range wallpaperIDs {
		if i >= 3 {
			break
		}
		if err := s.addFavorite(ctx, userID, wpID); err != nil {
			log.Printf("warning: failed to add favorite: %v", err)
		}
	}
	log.Printf("added %d favorites for %s", min(3, len(wallpaperIDs)), userEmail)

	return nil
}

func (s *seeder) seedUser(ctx context.Context, email, password, role string) (string, error) {
	// Check if user already exists
	var id string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM users WHERE email = $1`, email,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check user: %w", err)
	}

	hash, err := s.authService.HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
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
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM wallpapers WHERE title = $1`, v.Title,
	).Scan(&existingID)
	if err == nil {
		log.Printf("  skipping %q (already exists)", v.Title)
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("check existing: %w", err)
	}

	// Create temp dir for this video
	tmpDir, err := os.MkdirTemp("", "seed-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	videoPath := filepath.Join(tmpDir, "video.mp4")

	// Get the video file
	if s.pexelsKey != "" {
		if err := s.downloadFromPexels(ctx, v.PexelsID, videoPath); err != nil {
			log.Printf("  pexels download failed, generating video: %v", err)
			if err := s.generateVideo(ctx, v, videoPath); err != nil {
				return "", fmt.Errorf("generate video: %w", err)
			}
		}
	} else {
		if err := s.generateVideo(ctx, v, videoPath); err != nil {
			return "", fmt.Errorf("generate video: %w", err)
		}
	}

	// Generate thumbnail and preview
	thumbPath := filepath.Join(tmpDir, "thumb.jpg")
	previewPath := filepath.Join(tmpDir, "preview.mp4")

	if err := s.videoService.GenerateThumbnail(ctx, videoPath, thumbPath); err != nil {
		return "", fmt.Errorf("generate thumbnail: %w", err)
	}
	if err := s.videoService.GeneratePreview(ctx, videoPath, previewPath); err != nil {
		return "", fmt.Errorf("generate preview: %w", err)
	}

	// Get actual file info
	info, err := s.videoService.Probe(ctx, videoPath)
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

	// Use probed values for width/height/duration, fall back to manifest
	width := info.Width
	height := info.Height
	duration := info.Duration
	fileSize := info.Size
	format := info.Format
	resolution := fmt.Sprintf("%dx%d", width, height)

	if width == 0 {
		width = v.Width
		height = v.Height
		resolution = v.Resolution
	}
	if duration == 0 {
		duration = v.Duration
	}
	if format == "" {
		format = "h264"
	}

	// Insert wallpaper record
	var insertedID string
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO wallpapers (id, title, uploader_id, status, category, tags, resolution, width, height, duration, file_size, format, download_count, storage_key, thumbnail_key, preview_key)
		 VALUES ($1, $2, $3, 'approved', $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		 RETURNING id`,
		wpID, v.Title, uploaderID, v.Category, pq.Array(v.Tags),
		resolution, width, height, duration, fileSize, format,
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
	defer f.Close()
	return s.store.Put(ctx, key, f, contentType)
}

func (s *seeder) addFavorite(ctx context.Context, userID, wallpaperID string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO favorites (user_id, wallpaper_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, wallpaperID,
	)
	return err
}

// downloadFromPexels fetches a video from the Pexels API by video ID.
func (s *seeder) downloadFromPexels(ctx context.Context, pexelsID int, outputPath string) error {
	url := fmt.Sprintf("https://api.pexels.com/videos/videos/%d", pexelsID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", s.pexelsKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("pexels API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("pexels API returned %d", resp.StatusCode)
	}

	var result struct {
		VideoFiles []struct {
			Quality string `json:"quality"`
			Link    string `json:"link"`
			Width   int    `json:"width"`
			Height  int    `json:"height"`
			FileType string `json:"file_type"`
		} `json:"video_files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode pexels response: %w", err)
	}

	// Pick the best file: prefer HD (1920x1080) to keep downloads fast
	var downloadURL string
	for _, f := range result.VideoFiles {
		if f.FileType != "video/mp4" {
			continue
		}
		if f.Width == 1920 || f.Quality == "hd" {
			downloadURL = f.Link
			break
		}
	}
	// Fall back to first mp4
	if downloadURL == "" {
		for _, f := range result.VideoFiles {
			if f.FileType == "video/mp4" {
				downloadURL = f.Link
				break
			}
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no suitable video file found for pexels ID %d", pexelsID)
	}

	return downloadFile(ctx, downloadURL, outputPath)
}

func downloadFile(ctx context.Context, url, outputPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// generateVideo creates a placeholder video with ffmpeg using colored gradients
// and text overlays. Used when PEXELS_API_KEY is not set.
func (s *seeder) generateVideo(ctx context.Context, v SeedVideo, outputPath string) error {
	// Map categories to color schemes (background gradient colors)
	bg1, bg2 := categoryColors(v.Category)
	dur := fmt.Sprintf("%.1f", v.Duration)

	// Generate a gradient video with category name and title overlay
	filter := fmt.Sprintf(
		"gradients=s=1920x1080:c0=%s:c1=%s:duration=%s:speed=1,"+
			"drawtext=text='%s':fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2-40:shadowcolor=black:shadowx=2:shadowy=2,"+
			"drawtext=text='%s':fontsize=28:fontcolor=white@0.7:x=(w-text_w)/2:y=(h-text_h)/2+30:shadowcolor=black:shadowx=1:shadowy=1",
		bg1, bg2, dur,
		strings.ReplaceAll(v.Title, "'", ""),
		strings.ToUpper(v.Category),
	)

	// Try gradients filter first (ffmpeg 6+), fall back to color source
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=%s:s=1920x1080:d=%s", bg1, dur),
		"-vf", fmt.Sprintf(
			"drawtext=text='%s':fontsize=48:fontcolor=white:x=(w-text_w)/2:y=(h-text_h)/2-40:shadowcolor=black:shadowx=2:shadowy=2,"+
				"drawtext=text='%s':fontsize=28:fontcolor=white@0.7:x=(w-text_w)/2:y=(h-text_h)/2+30:shadowcolor=black:shadowx=1:shadowy=1",
			strings.ReplaceAll(v.Title, "'", ""),
			strings.ToUpper(v.Category),
		),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-pix_fmt", "yuv420p",
		"-t", dur,
		outputPath,
	)
	_ = filter // gradients filter used in future ffmpeg versions

	out, err := cmd.CombinedOutput()
	if err != nil {
		// drawtext may not be available, try without text
		cmd2 := exec.CommandContext(ctx, "ffmpeg",
			"-y",
			"-f", "lavfi",
			"-i", fmt.Sprintf("color=c=%s:s=1920x1080:d=%s:r=24", bg1, dur),
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-pix_fmt", "yuv420p",
			"-t", dur,
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

func categoryColors(category string) (string, string) {
	switch category {
	case "nature":
		return "0x2D5016", "0x1A3A4A"
	case "abstract":
		return "0x6B1D7B", "0xC2185B"
	case "space":
		return "0x0D0D2B", "0x1A1A4E"
	case "urban":
		return "0x2C3E50", "0xE67E22"
	case "underwater":
		return "0x004D6B", "0x00838F"
	default:
		return "0x333333", "0x666666"
	}
}
