package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
	"github.com/0x63616c/screenspace/server/storage"
)

// WallpaperService handles all wallpaper business logic.
type WallpaperService struct {
	db    db.Querier
	store storage.Store
	video video.Prober
	cfg   *config.Config
}

func NewWallpaperService(q db.Querier, s storage.Store, v video.Prober, cfg *config.Config) *WallpaperService {
	return &WallpaperService{db: q, store: s, video: v, cfg: cfg}
}

// Finalize runs the finalize flow: download, probe, validate, thumbnail,
// preview, upload assets, update DB status to pending_review.
//
// Status transition enforced: pending -> pending_review only.
func (s *WallpaperService) Finalize(ctx context.Context, wallpaperID, userID uuid.UUID) (*db.UpdateWallpaperAfterFinalizeRow, error) {
	// Step 1: Get wallpaper.
	wp, err := s.db.GetWallpaperByID(ctx, wallpaperID)
	if err != nil {
		return nil, handler.NotFound("wallpaper not found")
	}

	// Step 2: Ownership check.
	if wp.UploaderID != userID {
		return nil, handler.Forbidden("not your wallpaper")
	}

	// Step 3: Status must be pending.
	if types.WallpaperStatus(wp.Status) != types.StatusPending {
		return nil, handler.BadRequest("wallpaper is not in pending status")
	}

	// Steps 4-12 use deferred cleanup.
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID)

	// Step 4: Download original from S3.
	reader, err := s.store.Get(ctx, storageKey)
	if err != nil {
		return nil, handler.NotFound("video not found in storage")
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", "wallpaper-*.mp4")
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("create temp file: %w", err))
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	limited := io.LimitReader(reader, s.cfg.MaxFileSize+1)
	if _, err := io.Copy(tmpFile, limited); err != nil {
		return nil, handler.Internal(fmt.Errorf("write temp file: %w", err))
	}
	tmpFile.Close()

	// Step 5: Probe video.
	info, err := s.video.Probe(ctx, tmpFile.Name())
	if err != nil {
		return nil, handler.BadRequest("failed to probe video")
	}

	// Step 6: Validate constraints.
	if info.Size > s.cfg.MaxFileSize {
		return nil, handler.BadRequest(fmt.Sprintf("file too large, max %dMB", s.cfg.MaxFileSize/1024/1024))
	}
	if info.Duration > s.cfg.MaxDuration {
		return nil, handler.BadRequest(fmt.Sprintf("video too long, max %.0f seconds", s.cfg.MaxDuration))
	}
	if info.Height < s.cfg.MinHeight {
		return nil, handler.BadRequest(fmt.Sprintf("minimum resolution is %dp", s.cfg.MinHeight))
	}
	if info.Format != "h264" && info.Format != "h265" {
		return nil, handler.BadRequest("only h264 and h265 codecs are supported")
	}

	// Step 7: Generate thumbnail.
	thumbPath := tmpFile.Name() + "_thumb.jpg"
	defer os.Remove(thumbPath)
	if err := s.video.GenerateThumbnail(ctx, tmpFile.Name(), thumbPath); err != nil {
		return nil, handler.Internal(fmt.Errorf("generate thumbnail: %w", err))
	}

	// Step 8: Generate preview clip.
	previewPath := tmpFile.Name() + "_preview.mp4"
	defer os.Remove(previewPath)
	if err := s.video.GeneratePreview(ctx, tmpFile.Name(), previewPath); err != nil {
		return nil, handler.Internal(fmt.Errorf("generate preview: %w", err))
	}

	// Step 9: Upload thumbnail to S3.
	thumbnailKey := fmt.Sprintf("wallpapers/%s/thumbnail.jpg", wp.ID)
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("open thumbnail: %w", err))
	}
	defer thumbFile.Close()
	if err := s.store.Put(ctx, thumbnailKey, thumbFile, "image/jpeg"); err != nil {
		return nil, handler.Internal(fmt.Errorf("upload thumbnail: %w", err))
	}

	// Step 10: Upload preview to S3.
	previewKey := fmt.Sprintf("wallpapers/%s/preview.mp4", wp.ID)
	prevFile, err := os.Open(previewPath)
	if err != nil {
		return nil, handler.Internal(fmt.Errorf("open preview: %w", err))
	}
	defer prevFile.Close()
	if err := s.store.Put(ctx, previewKey, prevFile, "video/mp4"); err != nil {
		return nil, handler.Internal(fmt.Errorf("upload preview: %w", err))
	}

	// Step 11: Update DB.
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
		return nil, handler.Internal(fmt.Errorf("update after finalize: %w", err))
	}

	slog.Info("wallpaper finalized",
		"wallpaper_id", wp.ID,
		"user_id", userID,
		"resolution", resolution,
		"format", info.Format,
	)

	return &updated, nil
}

// GetApproved returns a wallpaper only if its status is approved.
func (s *WallpaperService) GetApproved(ctx context.Context, id uuid.UUID) (*db.GetWallpaperByIDRow, error) {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return nil, handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusApproved {
		return nil, handler.NotFound("wallpaper not found")
	}
	return &wp, nil
}

// Approve transitions a wallpaper from pending_review -> approved.
func (s *WallpaperService) Approve(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return handler.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateWallpaperStatus(ctx, db.UpdateWallpaperStatusParams{
		ID:     id,
		Status: string(types.StatusApproved),
	}); err != nil {
		return handler.Internal(fmt.Errorf("approve wallpaper: %w", err))
	}
	slog.Info("wallpaper approved", "wallpaper_id", id, "admin_id", adminID)
	return nil
}

// Reject transitions a wallpaper from pending_review -> rejected.
func (s *WallpaperService) Reject(ctx context.Context, id uuid.UUID, adminID uuid.UUID, reason string) error {
	wp, err := s.db.GetWallpaperByID(ctx, id)
	if err != nil {
		return handler.NotFound("wallpaper not found")
	}
	if types.WallpaperStatus(wp.Status) != types.StatusPendingReview {
		return handler.BadRequest("wallpaper is not in pending_review status")
	}
	if err := s.db.UpdateWallpaperStatusWithReason(ctx, db.UpdateWallpaperStatusWithReasonParams{
		ID:              id,
		Status:          string(types.StatusRejected),
		RejectionReason: &reason,
	}); err != nil {
		return handler.Internal(fmt.Errorf("reject wallpaper: %w", err))
	}
	slog.Info("wallpaper rejected", "wallpaper_id", id, "admin_id", adminID)
	return nil
}
