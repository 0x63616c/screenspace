package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

var ValidCategories = []string{"nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal", "other"}

type WallpaperHandler struct {
	q     db.Querier
	store storage.Store
	video *service.VideoService
	auth  *service.AuthService
}

func NewWallpaperHandler(
	q db.Querier,
	store storage.Store,
	video *service.VideoService,
	auth *service.AuthService,
) *WallpaperHandler {
	return &WallpaperHandler{
		q:     q,
		store: store,
		video: video,
		auth:  auth,
	}
}

type createWallpaperRequest struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

type createWallpaperResponse struct {
	ID        string `json:"id"`
	UploadURL string `json:"upload_url"`
}

func (h *WallpaperHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req createWallpaperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Title) > 255 {
		http.Error(w, `{"error":"title must be 255 characters or fewer"}`, http.StatusBadRequest)
		return
	}

	if len(req.Tags) > 10 {
		http.Error(w, `{"error":"maximum 10 tags allowed"}`, http.StatusBadRequest)
		return
	}

	for _, tag := range req.Tags {
		if len(tag) > 50 {
			http.Error(w, `{"error":"each tag must be 50 characters or fewer"}`, http.StatusBadRequest)
			return
		}
	}

	if req.Category != "" {
		normalized := strings.ToLower(req.Category)
		valid := false
		for _, c := range ValidCategories {
			if c == normalized {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, `{"error":"invalid category"}`, http.StatusBadRequest)
			return
		}
		req.Category = normalized
	}

	uploaderID, err := parseUUID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", "pending")
	wp, err := h.q.CreateWallpaper(r.Context(), db.CreateWallpaperParams{
		Title:      req.Title,
		UploaderID: uploaderID,
		StorageKey: storageKey,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Update storage key with actual ID
	actualKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID.String())
	_, err = h.q.UpdateWallpaperAfterFinalize(r.Context(), db.UpdateWallpaperAfterFinalizeParams{
		Width:        0,
		Height:       0,
		Duration:     0,
		FileSize:     0,
		Format:       "",
		Resolution:   "",
		ThumbnailKey: "",
		PreviewKey:   "",
		Status:       "pending",
		ID:           wp.ID,
	})
	if err != nil {
		slog.Error("failed to update storage key after create", "wallpaper_id", wp.ID.String(), "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Update metadata if category/tags provided
	if req.Category != "" || len(req.Tags) > 0 {
		tags := req.Tags
		if tags == nil {
			tags = []string{}
		}
		cat := &req.Category
		if req.Category == "" {
			cat = nil
		}
		if err := h.q.UpdateWallpaperMetadata(r.Context(), db.UpdateWallpaperMetadataParams{
			Title:    req.Title,
			Category: cat,
			Tags:     tags,
			ID:       wp.ID,
		}); err != nil {
			slog.Error("failed to update metadata after create", "wallpaper_id", wp.ID.String(), "error", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
	}

	uploadURL, err := h.store.PreSignedUploadURL(r.Context(), actualKey, 2*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("wallpaper uploaded", "user_id", claims.UserID, "wallpaper_id", wp.ID.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createWallpaperResponse{
		ID:        wp.ID.String(),
		UploadURL: uploadURL,
	})
}

func (h *WallpaperHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	wp, err := h.q.GetWallpaperByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.UploaderID.String() != claims.UserID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Download from S3 to temp file
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID.String())
	reader, err := h.store.Get(r.Context(), storageKey)
	if err != nil {
		http.Error(w, `{"error":"video not found in storage"}`, http.StatusNotFound)
		return
	}
	defer reader.Close()

	tmpFile, err := os.CreateTemp("", "wallpaper-*.mp4")
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	tmpFile.Close()

	// Probe video
	info, err := h.video.Probe(r.Context(), tmpFile.Name())
	if err != nil {
		http.Error(w, `{"error":"failed to probe video"}`, http.StatusBadRequest)
		return
	}

	// Validate constraints
	if info.Size > 200*1024*1024 {
		http.Error(w, `{"error":"file too large, max 200MB"}`, http.StatusBadRequest)
		return
	}
	if info.Duration > 60 {
		http.Error(w, `{"error":"video too long, max 60 seconds"}`, http.StatusBadRequest)
		return
	}
	if info.Height < 1080 {
		http.Error(w, `{"error":"minimum resolution is 1080p"}`, http.StatusBadRequest)
		return
	}
	if info.Format != "h264" && info.Format != "h265" {
		http.Error(w, `{"error":"only h264 and h265 codecs are supported"}`, http.StatusBadRequest)
		return
	}

	// Generate thumbnail
	thumbPath := tmpFile.Name() + "_thumb.jpg"
	defer os.Remove(thumbPath)
	if err := h.video.GenerateThumbnail(r.Context(), tmpFile.Name(), thumbPath); err != nil {
		http.Error(w, `{"error":"failed to generate thumbnail"}`, http.StatusInternalServerError)
		return
	}

	// Generate preview
	previewPath := tmpFile.Name() + "_preview.mp4"
	defer os.Remove(previewPath)
	if err := h.video.GeneratePreview(r.Context(), tmpFile.Name(), previewPath); err != nil {
		http.Error(w, `{"error":"failed to generate preview"}`, http.StatusInternalServerError)
		return
	}

	// Upload thumbnail to S3
	thumbnailKey := fmt.Sprintf("wallpapers/%s/thumbnail.jpg", wp.ID.String())
	thumbFile, err := os.Open(thumbPath)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer thumbFile.Close()
	if err := h.store.Put(r.Context(), thumbnailKey, thumbFile, "image/jpeg"); err != nil {
		http.Error(w, `{"error":"failed to upload thumbnail"}`, http.StatusInternalServerError)
		return
	}

	// Upload preview to S3
	previewKey := fmt.Sprintf("wallpapers/%s/preview.mp4", wp.ID.String())
	prevFile, err := os.Open(previewPath)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer prevFile.Close()
	if err := h.store.Put(r.Context(), previewKey, prevFile, "video/mp4"); err != nil {
		http.Error(w, `{"error":"failed to upload preview"}`, http.StatusInternalServerError)
		return
	}

	// Compute resolution string
	resolution := fmt.Sprintf("%dx%d", info.Width, info.Height)

	// Update DB
	_, err = h.q.UpdateWallpaperAfterFinalize(r.Context(), db.UpdateWallpaperAfterFinalizeParams{
		Width:        int32(info.Width),
		Height:       int32(info.Height),
		Duration:     info.Duration,
		FileSize:     info.Size,
		Format:       info.Format,
		Resolution:   resolution,
		ThumbnailKey: thumbnailKey,
		PreviewKey:   previewKey,
		Status:       "pending_review",
		ID:           wp.ID,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "pending_review"})
}

type wallpaperResponse struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	UploaderID      string   `json:"uploader_id"`
	Status          string   `json:"status"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	Resolution      string   `json:"resolution"`
	Width           int      `json:"width"`
	Height          int      `json:"height"`
	Duration        float64  `json:"duration"`
	FileSize        int64    `json:"file_size"`
	Format          string   `json:"format"`
	DownloadCount   int64    `json:"download_count"`
	DownloadURL     string   `json:"download_url,omitempty"`
	ThumbnailURL    string   `json:"thumbnail_url,omitempty"`
	PreviewURL      string   `json:"preview_url,omitempty"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func timestamptzToString(t pgtype.Timestamptz) string {
	if t.Valid {
		return t.Time.Format(time.RFC3339)
	}
	return ""
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func wallpaperRowToResponse(w *db.GetWallpaperByIDRow) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:              w.ID.String(),
		Title:           w.Title,
		UploaderID:      w.UploaderID.String(),
		Status:          w.Status,
		Category:        w.Category,
		Tags:            tags,
		Resolution:      w.Resolution,
		Width:           int(w.Width),
		Height:          int(w.Height),
		Duration:        w.Duration,
		FileSize:        w.FileSize,
		Format:          w.Format,
		DownloadCount:   w.DownloadCount,
		RejectionReason: derefString(w.RejectionReason),
		CreatedAt:       timestamptzToString(w.CreatedAt),
		UpdatedAt:       timestamptzToString(w.UpdatedAt),
	}
}

func wallpaperRowToResponseWithURLs(ctx context.Context, s storage.Store, w *db.GetWallpaperByIDRow) wallpaperResponse {
	resp := wallpaperRowToResponse(w)
	if w.ThumbnailKey != "" {
		if url, err := s.PreSignedURL(ctx, w.ThumbnailKey, 1*time.Hour); err == nil {
			resp.ThumbnailURL = url
		}
	}
	if w.PreviewKey != "" {
		if url, err := s.PreSignedURL(ctx, w.PreviewKey, 1*time.Hour); err == nil {
			resp.PreviewURL = url
		}
	}
	return resp
}

func recentRowToResponse(w *db.ListWallpapersRecentRow) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:              w.ID.String(),
		Title:           w.Title,
		UploaderID:      w.UploaderID.String(),
		Status:          w.Status,
		Category:        w.Category,
		Tags:            tags,
		Resolution:      w.Resolution,
		Width:           int(w.Width),
		Height:          int(w.Height),
		Duration:        w.Duration,
		FileSize:        w.FileSize,
		Format:          w.Format,
		DownloadCount:   w.DownloadCount,
		RejectionReason: derefString(w.RejectionReason),
		CreatedAt:       timestamptzToString(w.CreatedAt),
		UpdatedAt:       timestamptzToString(w.UpdatedAt),
	}
}

func recentRowToResponseWithURLs(ctx context.Context, s storage.Store, w *db.ListWallpapersRecentRow) wallpaperResponse {
	resp := recentRowToResponse(w)
	if w.ThumbnailKey != "" {
		if url, err := s.PreSignedURL(ctx, w.ThumbnailKey, 1*time.Hour); err == nil {
			resp.ThumbnailURL = url
		}
	}
	if w.PreviewKey != "" {
		if url, err := s.PreSignedURL(ctx, w.PreviewKey, 1*time.Hour); err == nil {
			resp.PreviewURL = url
		}
	}
	return resp
}

func popularRowToResponse(w *db.ListWallpapersPopularRow) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:              w.ID.String(),
		Title:           w.Title,
		UploaderID:      w.UploaderID.String(),
		Status:          w.Status,
		Category:        w.Category,
		Tags:            tags,
		Resolution:      w.Resolution,
		Width:           int(w.Width),
		Height:          int(w.Height),
		Duration:        w.Duration,
		FileSize:        w.FileSize,
		Format:          w.Format,
		DownloadCount:   w.DownloadCount,
		RejectionReason: derefString(w.RejectionReason),
		CreatedAt:       timestamptzToString(w.CreatedAt),
		UpdatedAt:       timestamptzToString(w.UpdatedAt),
	}
}

func popularRowToResponseWithURLs(ctx context.Context, s storage.Store, w *db.ListWallpapersPopularRow) wallpaperResponse {
	resp := popularRowToResponse(w)
	if w.ThumbnailKey != "" {
		if url, err := s.PreSignedURL(ctx, w.ThumbnailKey, 1*time.Hour); err == nil {
			resp.ThumbnailURL = url
		}
	}
	if w.PreviewKey != "" {
		if url, err := s.PreSignedURL(ctx, w.PreviewKey, 1*time.Hour); err == nil {
			resp.PreviewURL = url
		}
	}
	return resp
}

func favoriteRowToResponse(w *db.ListFavoritesByUserRow) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:              w.ID.String(),
		Title:           w.Title,
		UploaderID:      w.UploaderID.String(),
		Status:          w.Status,
		Category:        w.Category,
		Tags:            tags,
		Resolution:      w.Resolution,
		Width:           int(w.Width),
		Height:          int(w.Height),
		Duration:        w.Duration,
		FileSize:        w.FileSize,
		Format:          w.Format,
		DownloadCount:   w.DownloadCount,
		RejectionReason: derefString(w.RejectionReason),
		CreatedAt:       timestamptzToString(w.CreatedAt),
		UpdatedAt:       timestamptzToString(w.UpdatedAt),
	}
}

type listWallpapersResponse struct {
	Wallpapers []wallpaperResponse `json:"wallpapers"`
	Total      int                 `json:"total"`
}

func (h *WallpaperHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, offset := parseLimitOffset(q)

	sort := "recent"
	if s := q.Get("sort"); s == "popular" || s == "recent" {
		sort = s
	}

	category := q.Get("category")
	query := q.Get("q")

	var catPtr *string
	if category != "" {
		catPtr = &category
	}
	var queryPtr *string
	if query != "" {
		search := "%" + query + "%"
		queryPtr = &search
	}

	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status:   "approved",
		Category: catPtr,
		Query:    queryPtr,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0),
		Total:      int(total),
	}

	if sort == "popular" {
		wallpapers, err := h.q.ListWallpapersPopular(r.Context(), db.ListWallpapersPopularParams{
			Status:   "approved",
			Category: catPtr,
			Query:    queryPtr,
			Lim:      int32(limit),
			Off:      int32(offset),
		})
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		for i := range wallpapers {
			resp.Wallpapers = append(resp.Wallpapers, popularRowToResponseWithURLs(r.Context(), h.store, &wallpapers[i]))
		}
	} else {
		wallpapers, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
			Status:   "approved",
			Category: catPtr,
			Query:    queryPtr,
			Lim:      int32(limit),
			Off:      int32(offset),
		})
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		for i := range wallpapers {
			resp.Wallpapers = append(resp.Wallpapers, recentRowToResponseWithURLs(r.Context(), h.store, &wallpapers[i]))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *WallpaperHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	wp, err := h.q.GetWallpaperByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.Status != "approved" {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	resp := wallpaperRowToResponseWithURLs(r.Context(), h.store, &wp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type downloadResponse struct {
	DownloadURL string `json:"download_url"`
}

func (h *WallpaperHandler) Download(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	wp, err := h.q.GetWallpaperByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.Status != "approved" {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	downloadURL, err := h.store.PreSignedURL(r.Context(), wp.StorageKey, 1*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	_ = h.q.IncrementDownloadCount(r.Context(), wp.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(downloadResponse{DownloadURL: downloadURL})
}

func (h *WallpaperHandler) Popular(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("sort", "popular")
	r.URL.RawQuery = q.Encode()
	h.List(w, r)
}

func (h *WallpaperHandler) Recent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	q.Set("sort", "recent")
	r.URL.RawQuery = q.Encode()
	h.List(w, r)
}

func (h *WallpaperHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	wp, err := h.q.GetWallpaperByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.UploaderID.String() != claims.UserID && claims.Role != "admin" {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Delete from S3 (best effort)
	ctx := r.Context()
	h.store.Delete(ctx, wp.StorageKey)
	if wp.ThumbnailKey != "" {
		h.store.Delete(ctx, wp.ThumbnailKey)
	}
	if wp.PreviewKey != "" {
		h.store.Delete(ctx, wp.PreviewKey)
	}

	if err := h.q.DeleteWallpaper(ctx, wp.ID); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
