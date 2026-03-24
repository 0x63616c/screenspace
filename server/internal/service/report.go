package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/apperr"
)

// ReportService handles report creation and admin dismissal.
type ReportService struct {
	db  db.Querier
	cfg *config.Config
}

func NewReportService(q db.Querier, cfg *config.Config) *ReportService {
	return &ReportService{db: q, cfg: cfg}
}

// Create validates and persists a new report.
func (s *ReportService) Create(ctx context.Context, wallpaperID, reporterID uuid.UUID, reason string) (*db.Report, error) {
	if reason == "" {
		return nil, apperr.BadRequest("reason is required")
	}
	if len(reason) > s.cfg.MaxReportLength {
		return nil, apperr.BadRequest(fmt.Sprintf("reason must be %d characters or fewer", s.cfg.MaxReportLength))
	}

	report, err := s.db.CreateReport(ctx, db.CreateReportParams{
		WallpaperID: wallpaperID,
		ReporterID:  reporterID,
		Reason:      reason,
	})
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("create report: %w", err))
	}
	return &report, nil
}

// Dismiss marks a report as dismissed.
func (s *ReportService) Dismiss(ctx context.Context, reportID uuid.UUID) error {
	if err := s.db.DismissReport(ctx, reportID); err != nil {
		return apperr.Internal(fmt.Errorf("dismiss report: %w", err))
	}
	return nil
}
