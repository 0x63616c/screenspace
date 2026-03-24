package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

// ReportHandler handles wallpaper reports.
type ReportHandler struct {
	svc *service.ReportService
}

// NewReportHandler creates a new ReportHandler.
func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// Create submits a new report for a wallpaper.
func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return BadRequest("invalid user id")
	}
	wallpaperID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}

	report, err := h.svc.Create(r.Context(), wallpaperID, userID, req.Reason)
	if err != nil {
		return err
	}

	return respond.JSON(w, http.StatusCreated, map[string]any{
		"id":           report.ID.String(),
		"wallpaper_id": report.WallpaperID.String(),
		"reporter_id":  report.ReporterID.String(),
		"reason":       report.Reason,
		"status":       report.Status,
		"created_at":   formatTimestamp(report.CreatedAt),
	})
}
