package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// WallpaperHandler handles HTTP requests for the wallpaper resource.
type WallpaperHandler struct {
	q         db.Querier
	store     storage.Store
	svc       *service.WallpaperService
	auth      *service.AuthService
	cfg       *config.Config
	startTime time.Time
}

// NewWallpaperHandler creates a new WallpaperHandler.
func NewWallpaperHandler(q db.Querier, s storage.Store, svc *service.WallpaperService, auth *service.AuthService, cfg *config.Config) *WallpaperHandler {
	return &WallpaperHandler{q: q, store: s, svc: svc, auth: auth, cfg: cfg, startTime: time.Now()}
}

// Health checks database connectivity and reports uptime.
func (h *WallpaperHandler) Health(w http.ResponseWriter, r *http.Request) error {
	dbStatus := "ok"
	type pinger interface {
		Ping(context.Context) error
	}
	if p, ok := h.q.(pinger); ok {
		if err := p.Ping(r.Context()); err != nil {
			dbStatus = "unavailable"
		}
	}
	uptime := time.Since(h.startTime).Truncate(time.Second).String()
	return respond.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"db":     dbStatus,
		"uptime": uptime,
	})
}

// ListCategories returns all valid category values.
func ListCategories(w http.ResponseWriter, r *http.Request) error {
	cats := types.AllCategories()
	strs := make([]string, len(cats))
	for i, c := range cats {
		strs[i] = string(c)
	}
	return respond.JSON(w, http.StatusOK, strs)
}

// Get returns a single approved wallpaper by ID.
func (h *WallpaperHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	wp, err := h.svc.GetApproved(r.Context(), id)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, h.wallpaperRowToResponse(r, wp))
}

// List returns paginated wallpapers with optional sort/filter.
func (h *WallpaperHandler) List(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, "")
}

// Popular returns wallpapers sorted by download count.
func (h *WallpaperHandler) Popular(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, string(types.SortPopular))
}

// Recent returns wallpapers sorted by creation date.
func (h *WallpaperHandler) Recent(w http.ResponseWriter, r *http.Request) error {
	return h.list(w, r, string(types.SortRecent))
}

func (h *WallpaperHandler) list(w http.ResponseWriter, r *http.Request, forceSort string) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)

	sort := forceSort
	if sort == "" {
		s := q.Get("sort")
		if types.SortOrder(s).Valid() {
			sort = s
		} else {
			sort = string(types.SortRecent)
		}
	}

	category := ptrOrNil(q.Get("category"))
	query := ptrOrNil(q.Get("q"))
	status := string(types.StatusApproved)

	// Get total count.
	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status:   status,
		Category: category,
		Query:    query,
	})
	if err != nil {
		return Internal(fmt.Errorf("count wallpapers: %w", err))
	}

	var items []map[string]any
	if sort == string(types.SortPopular) {
		rows, err := h.q.ListWallpapersPopular(r.Context(), db.ListWallpapersPopularParams{
			Status:   status,
			Category: category,
			Query:    query,
			Off:      int32(pg.Offset),
			Lim:      int32(pg.Limit),
		})
		if err != nil {
			return Internal(fmt.Errorf("list wallpapers: %w", err))
		}
		items = make([]map[string]any, len(rows))
		for i := range rows {
			items[i] = popularRowToMap(&rows[i], h, r)
		}
	} else {
		rows, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
			Status:   status,
			Category: category,
			Query:    query,
			Off:      int32(pg.Offset),
			Lim:      int32(pg.Limit),
		})
		if err != nil {
			return Internal(fmt.Errorf("list wallpapers: %w", err))
		}
		items = make([]map[string]any, len(rows))
		for i := range rows {
			items[i] = recentRowToMap(&rows[i], h, r)
		}
	}

	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

// Create creates a new wallpaper record and returns a presigned upload URL.
func (h *WallpaperHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())

	var req struct {
		Title    string `json:"title"`
		Category string `json:"category"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Title == "" {
		return BadRequest("title is required")
	}
	if len(req.Title) > h.cfg.MaxTitleLength {
		return BadRequest(fmt.Sprintf("title must be %d characters or fewer", h.cfg.MaxTitleLength))
	}
	if req.Category != "" && !types.Category(req.Category).Valid() {
		return BadRequest("invalid category")
	}

	uploaderID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return Internal(fmt.Errorf("parse user id: %w", err))
	}

	var category *string
	if req.Category != "" {
		category = &req.Category
	}

	// Insert with a placeholder storage key; we need the DB-generated ID first.
	wp, err := h.q.CreateWallpaper(r.Context(), db.CreateWallpaperParams{
		Title:      req.Title,
		UploaderID: uploaderID,
		StorageKey: "pending",
		Category:   category,
	})
	if err != nil {
		return Internal(fmt.Errorf("create wallpaper: %w", err))
	}

	// Use the wallpaper ID as the storage key so finalize can find the file.
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID.String())
	if err := h.q.UpdateWallpaperStorageKey(r.Context(), db.UpdateWallpaperStorageKeyParams{
		StorageKey: storageKey,
		ID:         wp.ID,
	}); err != nil {
		return Internal(fmt.Errorf("update storage key: %w", err))
	}

	uploadURL, err := h.store.PreSignedUploadURL(r.Context(), storageKey, h.cfg.PresignedUploadExpiry)
	if err != nil {
		return Internal(fmt.Errorf("presign upload url: %w", err))
	}

	return respond.JSON(w, http.StatusCreated, map[string]string{
		"id":         wp.ID.String(),
		"upload_url": uploadURL,
	})
}

// Finalize triggers the finalize flow for a wallpaper.
func (h *WallpaperHandler) Finalize(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return Internal(fmt.Errorf("parse user id: %w", err))
	}

	wp, err := h.svc.Finalize(r.Context(), id, userID)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": wp.Status})
}

// Download returns a presigned download URL for the wallpaper.
func (h *WallpaperHandler) Download(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	wp, err := h.svc.GetApproved(r.Context(), id)
	if err != nil {
		return err
	}

	url, err := h.store.PreSignedURL(r.Context(), wp.StorageKey, h.cfg.PresignedDownloadExpiry)
	if err != nil {
		return Internal(fmt.Errorf("presign download url: %w", err))
	}

	if err := h.q.IncrementDownloadCount(r.Context(), wp.ID); err != nil {
		slog.Error("increment download count", "wallpaper_id", wp.ID, "error", err) //nolint:gosec // structured log
	}

	return respond.JSON(w, http.StatusOK, map[string]string{"download_url": url})
}

func (h *WallpaperHandler) wallpaperRowToResponse(r *http.Request, wp *db.GetWallpaperByIDRow) map[string]any {
	resp := map[string]any{
		"id":             wp.ID.String(),
		"title":          wp.Title,
		"uploader_id":    wp.UploaderID.String(),
		"status":         wp.Status,
		"category":       wp.Category,
		"tags":           wp.Tags,
		"resolution":     wp.Resolution,
		"width":          wp.Width,
		"height":         wp.Height,
		"duration":       wp.Duration,
		"file_size":      wp.FileSize,
		"format":         wp.Format,
		"download_count": wp.DownloadCount,
		"created_at":     formatTimestamp(wp.CreatedAt),
		"updated_at":     formatTimestamp(wp.UpdatedAt),
	}
	if wp.RejectionReason != nil {
		resp["rejection_reason"] = *wp.RejectionReason
	}
	if wp.ThumbnailKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), wp.ThumbnailKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["thumbnail_url"] = url
		}
	}
	if wp.PreviewKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), wp.PreviewKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["preview_url"] = url
		}
	}
	return resp
}

func popularRowToMap(row *db.ListWallpapersPopularRow, h *WallpaperHandler, r *http.Request) map[string]any {
	resp := map[string]any{
		"id":             row.ID.String(),
		"title":          row.Title,
		"uploader_id":    row.UploaderID.String(),
		"status":         row.Status,
		"category":       row.Category,
		"tags":           row.Tags,
		"resolution":     row.Resolution,
		"width":          row.Width,
		"height":         row.Height,
		"duration":       row.Duration,
		"file_size":      row.FileSize,
		"format":         row.Format,
		"download_count": row.DownloadCount,
		"created_at":     formatTimestamp(row.CreatedAt),
		"updated_at":     formatTimestamp(row.UpdatedAt),
	}
	if row.ThumbnailKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), row.ThumbnailKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["thumbnail_url"] = url
		}
	}
	if row.PreviewKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), row.PreviewKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["preview_url"] = url
		}
	}
	return resp
}

func recentRowToMap(row *db.ListWallpapersRecentRow, h *WallpaperHandler, r *http.Request) map[string]any {
	resp := map[string]any{
		"id":             row.ID.String(),
		"title":          row.Title,
		"uploader_id":    row.UploaderID.String(),
		"status":         row.Status,
		"category":       row.Category,
		"tags":           row.Tags,
		"resolution":     row.Resolution,
		"width":          row.Width,
		"height":         row.Height,
		"duration":       row.Duration,
		"file_size":      row.FileSize,
		"format":         row.Format,
		"download_count": row.DownloadCount,
		"created_at":     formatTimestamp(row.CreatedAt),
		"updated_at":     formatTimestamp(row.UpdatedAt),
	}
	if row.ThumbnailKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), row.ThumbnailKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["thumbnail_url"] = url
		}
	}
	if row.PreviewKey != "" {
		if url, err := h.store.PreSignedURL(r.Context(), row.PreviewKey, h.cfg.PresignedDownloadExpiry); err == nil {
			resp["preview_url"] = url
		}
	}
	return resp
}

func formatTimestamp(ts pgtype.Timestamptz) string {
	if ts.Valid {
		return ts.Time.Format(time.RFC3339)
	}
	return ""
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
