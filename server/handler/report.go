package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/0x63616c/screenspace/server/repository"
)

type ReportHandler struct {
	reports *repository.ReportRepo
}

func NewReportHandler(reports *repository.ReportRepo) *ReportHandler {
	return &ReportHandler{reports: reports}
}

type createReportRequest struct {
	Reason string `json:"reason"`
}

type reportResponse struct {
	ID          string `json:"id"`
	WallpaperID string `json:"wallpaper_id"`
	ReporterID  string `json:"reporter_id"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

func reportToResponse(r *repository.Report) reportResponse {
	return reportResponse{
		ID:          r.ID,
		WallpaperID: r.WallpaperID,
		ReporterID:  r.ReporterID,
		Reason:      r.Reason,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}
}

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	wallpaperID := r.PathValue("id")
	if wallpaperID == "" {
		http.Error(w, `{"error":"wallpaper id is required"}`, http.StatusBadRequest)
		return
	}

	var req createReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		http.Error(w, `{"error":"reason is required"}`, http.StatusBadRequest)
		return
	}

	report, err := h.reports.Create(r.Context(), wallpaperID, claims.UserID, req.Reason)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reportToResponse(report))
}
