package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	db "github.com/0x63616c/screenspace/server/db/generated"
)

type ReportHandler struct {
	q db.Querier
}

// NewReportHandler creates a handler for report operations.
func NewReportHandler(q db.Querier) *ReportHandler {
	return &ReportHandler{q: q}
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

func reportToResponse(r *db.Report) reportResponse {
	return reportResponse{
		ID:          r.ID.String(),
		WallpaperID: r.WallpaperID.String(),
		ReporterID:  r.ReporterID.String(),
		Reason:      r.Reason,
		Status:      r.Status,
		CreatedAt:   timestamptzToString(r.CreatedAt),
	}
}

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	wallpaperID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"wallpaper id is required"}`, http.StatusBadRequest)
		return
	}

	reporterID, err := parseUUID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
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

	if len(req.Reason) > 500 {
		http.Error(w, `{"error":"reason must be 500 characters or fewer"}`, http.StatusBadRequest)
		return
	}

	report, err := h.q.CreateReport(r.Context(), db.CreateReportParams{
		WallpaperID: wallpaperID,
		ReporterID:  reporterID,
		Reason:      req.Reason,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("wallpaper reported", "reporter_id", claims.UserID, "wallpaper_id", wallpaperID.String()) //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(reportToResponse(&report))
}
