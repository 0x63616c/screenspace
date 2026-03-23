package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

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
	h.wallpapers.UpdateAfterFinalize(r.Context(), wp.ID, repository.FinalizeParams{
		Status: "pending",
	})
	wp.StorageKey = actualKey

	// Update metadata if category/tags provided
	if req.Category != "" || len(req.Tags) > 0 {
		tags := req.Tags
		if tags == nil {
			tags = []string{}
		}
		h.wallpapers.UpdateMetadata(r.Context(), wp.ID, req.Title, req.Category, tags)
	}

	uploadURL, err := h.store.PreSignedUploadURL(r.Context(), actualKey, 15*time.Minute)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

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
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	UploaderID    string   `json:"uploader_id"`
	Status        string   `json:"status"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	Resolution    string   `json:"resolution"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Duration      float64  `json:"duration"`
	FileSize      int64    `json:"file_size"`
	Format        string   `json:"format"`
	DownloadCount int64    `json:"download_count"`
	DownloadURL   string   `json:"download_url,omitempty"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

func wallpaperToResponse(w *repository.Wallpaper) wallpaperResponse {
	tags := w.Tags
	if tags == nil {
		tags = []string{}
	}
	return wallpaperResponse{
		ID:            w.ID,
		Title:         w.Title,
		UploaderID:    w.UploaderID,
		Status:        w.Status,
		Category:      w.Category,
		Tags:          tags,
		Resolution:    w.Resolution,
		Width:         w.Width,
		Height:        w.Height,
		Duration:      w.Duration,
		FileSize:      w.FileSize,
		Format:        w.Format,
		DownloadCount: w.DownloadCount,
		CreatedAt:     w.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     w.UpdatedAt.Format(time.RFC3339),
	}
}

type listWallpapersResponse struct {
	Wallpapers []wallpaperResponse `json:"wallpapers"`
	Total      int                 `json:"total"`
}

func (h *WallpaperHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

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
		resp.Wallpapers = append(resp.Wallpapers, wallpaperToResponse(wp))
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

	// Generate download URL
	downloadURL, err := h.store.PreSignedURL(r.Context(), wp.StorageKey, 1*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	h.wallpapers.IncrementDownloadCount(r.Context(), wp.ID)

	resp := wallpaperToResponse(wp)
	resp.DownloadURL = downloadURL

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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
