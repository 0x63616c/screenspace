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

	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

var ValidCategories = []string{"nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal", "other"}

type WallpaperHandler struct {
	wallpapers *repository.WallpaperRepo
	store      storage.Store
	video      *service.VideoService
	auth       *service.AuthService
}

func NewWallpaperHandler(
	wallpapers *repository.WallpaperRepo,
	store storage.Store,
	video *service.VideoService,
	auth *service.AuthService,
) *WallpaperHandler {
	return &WallpaperHandler{
		wallpapers: wallpapers,
		store:      store,
		video:      video,
		auth:       auth,
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

	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", "pending")
	wp, err := h.wallpapers.Create(r.Context(), repository.CreateParams{
		Title:      req.Title,
		UploaderID: claims.UserID,
		StorageKey: storageKey,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Update storage key with actual ID
	actualKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID)
	if err := h.wallpapers.UpdateAfterFinalize(r.Context(), wp.ID, repository.FinalizeParams{
		Status: "pending",
	}); err != nil {
		slog.Error("failed to update storage key after create", "wallpaper_id", wp.ID, "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	wp.StorageKey = actualKey

	// Update metadata if category/tags provided
	if req.Category != "" || len(req.Tags) > 0 {
		tags := req.Tags
		if tags == nil {
			tags = []string{}
		}
		if err := h.wallpapers.UpdateMetadata(r.Context(), wp.ID, req.Title, req.Category, tags); err != nil {
			slog.Error("failed to update metadata after create", "wallpaper_id", wp.ID, "error", err)
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
	}

	uploadURL, err := h.store.PreSignedUploadURL(r.Context(), actualKey, 2*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("wallpaper uploaded", "user_id", claims.UserID, "wallpaper_id", wp.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createWallpaperResponse{
		ID:        wp.ID,
		UploadURL: uploadURL,
	})
}

func (h *WallpaperHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	id := r.PathValue("id")
	wp, err := h.wallpapers.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.UploaderID != claims.UserID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	// Download from S3 to temp file
	storageKey := fmt.Sprintf("wallpapers/%s/original.mp4", wp.ID)
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
	thumbnailKey := fmt.Sprintf("wallpapers/%s/thumbnail.jpg", wp.ID)
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
	previewKey := fmt.Sprintf("wallpapers/%s/preview.mp4", wp.ID)
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
	if err := h.wallpapers.UpdateAfterFinalize(r.Context(), wp.ID, repository.FinalizeParams{
		Width:        info.Width,
		Height:       info.Height,
		Duration:     info.Duration,
		FileSize:     info.Size,
		Format:       info.Format,
		Resolution:   resolution,
		ThumbnailKey: thumbnailKey,
		PreviewKey:   previewKey,
		Status:       "pending_review",
	}); err != nil {
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

func wallpaperToResponse(w *repository.Wallpaper) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:              w.ID,
		Title:           w.Title,
		UploaderID:      w.UploaderID,
		Status:          w.Status,
		Category:        w.Category,
		Tags:            tags,
		Resolution:      w.Resolution,
		Width:           w.Width,
		Height:          w.Height,
		Duration:        w.Duration,
		FileSize:        w.FileSize,
		Format:          w.Format,
		DownloadCount:   w.DownloadCount,
		RejectionReason: w.RejectionReason,
		CreatedAt:       w.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       w.UpdatedAt.Format(time.RFC3339),
	}
}

func wallpaperToResponseWithURLs(ctx context.Context, s storage.Store, w *repository.Wallpaper) wallpaperResponse {
	resp := wallpaperToResponse(w)
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

	wallpapers, total, err := h.wallpapers.List(r.Context(), repository.ListParams{
		Status:   "approved",
		Category: q.Get("category"),
		Query:    q.Get("q"),
		Sort:     sort,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      total,
	}
	for _, wp := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, wallpaperToResponseWithURLs(r.Context(), h.store, wp))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *WallpaperHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	wp, err := h.wallpapers.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.Status != "approved" {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	resp := wallpaperToResponseWithURLs(r.Context(), h.store, wp)

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

	id := r.PathValue("id")
	wp, err := h.wallpapers.GetByID(r.Context(), id)
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

	h.wallpapers.IncrementDownloadCount(r.Context(), wp.ID)

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

	id := r.PathValue("id")
	wp, err := h.wallpapers.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"wallpaper not found"}`, http.StatusNotFound)
		return
	}

	if wp.UploaderID != claims.UserID && claims.Role != "admin" {
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

	if err := h.wallpapers.Delete(ctx, wp.ID); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
