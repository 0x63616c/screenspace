package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/apperr"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
)

// WallpaperService handles all wallpaper business logic.
type WallpaperService struct {
	db    db.Querier
	store storage.Store
	video video.Prober
	cfg   *config.Config
}

// NewWallpaperService creates a new WallpaperService.
func NewWallpaperService(q db.Querier, s storage.Store, v video.Prober, cfg *config.Config) *WallpaperService {
	return &WallpaperService{db: q, store: s, video: v, cfg: cfg}
}

// Finalize runs the finalize flow: download, probe, validate, thumbnail,
// preview, upload assets, update DB status to pending_review.
//
// Status transition enforced: pending -> pending_review only.
func (s *WallpaperService) Finalize(ctx context.Context, wallpaperID, userID uuid.UUID) (*db.UpdateWallpaperAfterFinalizeRow, error) {
	wp, err := s.db.GetWallpaperByID(ctx, wallpaperID)
	if err != nil {
		return nil, apperr.NotFound("wallpaper not found")
	}
	if wp.UploaderID != userID {
		return nil, apperr.Forbidden("not your wallpaper")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPending {
		return nil, apperr.BadRequest("wallpaper is not in pending status")
	}

	tmpPath, info, err := s.downloadAndProbe(ctx, wp.ID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := s.validateVideo(info); err != nil {
		return nil, err
	}

	thumbnailKey, previewKey, err := s.generateAndUploadAssets(ctx, wp.ID, tmpPath)
	if err != nil {
		return nil, err
	}

	resolution := fmt.Sprintf("%dx%d", info.Width, info.Height)
	updated, err := s.db.UpdateWallpaperAfterFinalize(ctx, db.UpdateWallpaperAfterFinalizeParams{
		ID:           wp.ID,
		Width:        int32(info.Width),
		Height:       int32(info.Height),
		Duration:     info.Duration,
		FileSize:     info.Size,
		Format:       info.Format,
		Resolution:   resolution,
		ThumbnailKey: thumbnailKey,
		PreviewKey:   previewKey,
		Status:       string(types.StatusPendingReview),
	})
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("update after finalize: %w", err))
	}

	slog.Info("wallpaper finalized", //nolint:gosec // structured log, not user-controlled format
		"wallpaper_id", wp.ID,
		"user_id", userID,
		"resolution", resolution,
		"format", info.Format,
	)

	return &updated, nil
}

// downloadAndProbe downloads the original video from S3, writes it to a temp file, and probes it.
func (s *WallpaperService) downloadAndProbe(ctx context.Context, wpID uuid.UUID) (string, *video.ProbeResult, error) {
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wpID)

	reader, err := s.store.Get(ctx, storageKey)
	if err != nil {
		return "", nil, apperr.NotFound("video not found in storage")
	}
	defer func() { _ = reader.Close() }()

	tmpFile, err := os.CreateTemp("", "wallpaper-*.mp4")
	if err != nil {
		return "", nil, apperr.Internal(fmt.Errorf("create temp file: %w", err))
	}
	tmpPath := tmpFile.Name()

	limited := io.LimitReader(reader, s.cfg.MaxFileSize+1)
	if _, err := io.Copy(tmpFile, limited); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return "", nil, apperr.Internal(fmt.Errorf("write temp file: %w", err))
	}
	_ = tmpFile.Close()

	info, err := s.video.Probe(ctx, tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, apperr.BadRequest("failed to probe video")
	}

	return tmpPath, info, nil
}

// validateVideo checks file size, duration, resolution, and codec constraints.
func (s *WallpaperService) validateVideo(info *video.ProbeResult) error {
	if info.Size > s.cfg.MaxFileSize {
		return apperr.BadRequest(fmt.Sprintf("file too large, max %dMB", s.cfg.MaxFileSize/1024/1024))
	}
	if info.Duration > s.cfg.MaxDuration {
		return apperr.BadRequest(fmt.Sprintf("video too long, max %.0f seconds", s.cfg.MaxDuration))
	}
	if info.Height < s.cfg.MinHeight {
		return apperr.BadRequest(fmt.Sprintf("minimum resolution is %dp", s.cfg.MinHeight))
	}
	if info.Format != "h264" && info.Format != "h265" {
		return apperr.BadRequest("only h264 and h265 codecs are supported")
	}
	return nil
}

// generateAndUploadAssets creates thumbnail and preview, uploads them, and returns their storage keys.
func (s *WallpaperService) generateAndUploadAssets(ctx context.Context, wpID uuid.UUID, tmpPath string) (string, string, error) {
	thumbPath := tmpPath + "_thumb.jpg"
	if err := s.video.GenerateThumbnail(ctx, tmpPath, thumbPath); err != nil {
		return "", "", apperr.Internal(fmt.Errorf("generate thumbnail: %w", err))
	}
	defer func() { _ = os.Remove(thumbPath) }()

	previewPath := tmpPath + "_preview.mp4"
	if err := s.video.GeneratePreview(ctx, tmpPath, previewPath); err != nil {
		return "", "", apperr.Internal(fmt.Errorf("generate preview: %w", err))
	}
	defer func() { _ = os.Remove(previewPath) }()

	thumbnailKey := fmt.Sprintf("wallpapers/%s/thumbnail.jpg", wpID)
	thumbFile, err := os.Open(thumbPath) //nolint:gosec // path constructed from temp file, not user input
	if err != nil {
		return "", "", apperr.Internal(fmt.Errorf("open thumbnail: %w", err))
	}
	defer func() { _ = thumbFile.Close() }()
	if err := s.store.Put(ctx, thumbnailKey, thumbFile, "image/jpeg"); err != nil {
		return "", "", apperr.Internal(fmt.Errorf("upload thumbnail: %w", err))
	}

	previewKey := fmt.Sprintf("wallpapers/%s/preview.mp4", wpID)
	prevFile, err := os.Open(previewPath) //nolint:gosec // path constructed from temp file, not user input
	if err != nil {
		return "", "", apperr.Internal(fmt.Errorf("open preview: %w", err))
	}
	defer func() { _ = prevFile.Close() }()
	if err := s.store.Put(ctx, previewKey, prevFile, "video/mp4"); err != nil {
		return "", "", apperr.Internal(fmt.Errorf("upload preview: %w", err))
	}

	return thumbnailKey, previewKey, nil
}

// GetApproved returns a wallpaper only if its status is approved.
func (s *WallpaperService) GetApproved(ctx context.Context, id uuid.UUID) (*db.GetWallpaperByIDRow, error) {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return nil, apperr.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusApproved {
		return nil, apperr.NotFound("wallpaper not found")
	}
	return &wp, nil
}

// Approve transitions a wallpaper from pending_review -> approved.
func (s *WallpaperService) Approve(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return apperr.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return apperr.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateWallpaperStatus(ctx, db.UpdateWallpaperStatusParams{
		ID:     id,
		Status: string(types.StatusApproved),
	}); err != nil {
		return apperr.Internal(fmt.Errorf("approve wallpaper: %w", err))
	}
	slog.Info("wallpaper approved", "wallpaper_id", id, "admin_id", adminID)
	return nil
}

// Reject transitions a wallpaper from pending_review -> rejected.
func (s *WallpaperService) Reject(ctx context.Context, id uuid.UUID, adminID uuid.UUID, reason string) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return apperr.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return apperr.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateWallpaperStatusWithReason(ctx, db.UpdateWallpaperStatusWithReasonParams{
		ID:              id,
		Status:          string(types.StatusRejected),
		RejectionReason: &reason,
	}); err != nil {
		return apperr.Internal(fmt.Errorf("reject wallpaper: %w", err))
	}
	slog.Info("wallpaper rejected", "wallpaper_id", id, "admin_id", adminID)
	return nil
}
