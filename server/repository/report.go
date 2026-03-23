package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Report struct {
	ID          string
	WallpaperID string
	ReporterID  string
	Reason      string
	Status      string
	CreatedAt   time.Time
}

type ReportRepo struct {
	db *sql.DB
}

func NewReportRepo(db *sql.DB) *ReportRepo {
	return &ReportRepo{db: db}
}

func (r *ReportRepo) Create(ctx context.Context, wallpaperID, reporterID, reason string) (*Report, error) {
	rpt := &Report{}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO reports (wallpaper_id, reporter_id, reason)
		 VALUES ($1, $2, $3)
		 RETURNING id, wallpaper_id, reporter_id, reason, status, created_at`,
		wallpaperID, reporterID, reason,
	).Scan(&rpt.ID, &rpt.WallpaperID, &rpt.ReporterID, &rpt.Reason, &rpt.Status, &rpt.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}
	return rpt, nil
}

func (r *ReportRepo) ListPending(ctx context.Context, limit, offset int) ([]*Report, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reports WHERE status = 'pending'`,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count pending reports: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, wallpaper_id, reporter_id, reason, status, created_at
		 FROM reports WHERE status = 'pending'
		 ORDER BY created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list pending reports: %w", err)
	}
	defer rows.Close()

	var reports []*Report
	for rows.Next() {
		rpt := &Report{}
		if err := rows.Scan(&rpt.ID, &rpt.WallpaperID, &rpt.ReporterID, &rpt.Reason, &rpt.Status, &rpt.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, rpt)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return reports, total, nil
}

func (r *ReportRepo) Dismiss(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE reports SET status = 'dismissed' WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("dismiss report: %w", err)
	}
	return nil
}
