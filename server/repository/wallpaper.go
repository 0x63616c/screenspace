package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type Wallpaper struct {
	ID            string
	Title         string
	UploaderID    string
	Status        string
	Category      string
	Tags          []string
	Resolution    string
	Width         int
	Height        int
	Duration      float64
	FileSize      int64
	Format        string
	DownloadCount int64
	StorageKey    string
	ThumbnailKey  string
	PreviewKey    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateParams struct {
	Title      string
	UploaderID string
	StorageKey string
}

type ListParams struct {
	Status   string
	Category string
	Query    string
	Sort     string
	Limit    int
	Offset   int
}

type FinalizeParams struct {
	Width        int
	Height       int
	Duration     float64
	FileSize     int64
	Format       string
	Resolution   string
	ThumbnailKey string
	PreviewKey   string
	Status       string
}

type WallpaperRepo struct {
	db *sql.DB
}

func NewWallpaperRepo(db *sql.DB) *WallpaperRepo {
	return &WallpaperRepo{db: db}
}

func scanWallpaper(row interface{ Scan(...any) error }) (*Wallpaper, error) {
	w := &Wallpaper{}
	err := row.Scan(
		&w.ID, &w.Title, &w.UploaderID, &w.Status, &w.Category,
		pq.Array(&w.Tags), &w.Resolution, &w.Width, &w.Height,
		&w.Duration, &w.FileSize, &w.Format, &w.DownloadCount,
		&w.StorageKey, &w.ThumbnailKey, &w.PreviewKey,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if w.Tags == nil {
		w.Tags = []string{}
	}
	return w, nil
}

const wallpaperColumns = `id, title, uploader_id, status, COALESCE(category, ''), tags, resolution, width, height, duration, file_size, format, download_count, storage_key, thumbnail_key, preview_key, created_at, updated_at`

func (r *WallpaperRepo) Create(ctx context.Context, p CreateParams) (*Wallpaper, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO wallpapers (title, uploader_id, storage_key, resolution, width, height, duration, file_size, format, thumbnail_key, preview_key)
		 VALUES ($1, $2, $3, '', 0, 0, 0, 0, '', '', '')
		 RETURNING `+wallpaperColumns,
		p.Title, p.UploaderID, p.StorageKey,
	)
	w, err := scanWallpaper(row)
	if err != nil {
		return nil, fmt.Errorf("create wallpaper: %w", err)
	}
	return w, nil
}

func (r *WallpaperRepo) GetByID(ctx context.Context, id string) (*Wallpaper, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+wallpaperColumns+` FROM wallpapers WHERE id = $1`, id,
	)
	w, err := scanWallpaper(row)
	if err != nil {
		return nil, fmt.Errorf("get wallpaper by id: %w", err)
	}
	return w, nil
}

func (r *WallpaperRepo) List(ctx context.Context, p ListParams) ([]*Wallpaper, int, error) {
	var conditions []string
	var args []any
	argN := 1

	status := p.Status
	if status == "" {
		status = "approved"
	}
	conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
	args = append(args, status)
	argN++

	if p.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argN))
		args = append(args, p.Category)
		argN++
	}

	if p.Query != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", argN))
		args = append(args, "%"+p.Query+"%")
		argN++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	// Count query
	var total int
	countQuery := "SELECT COUNT(*) FROM wallpapers " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count wallpapers: %w", err)
	}

	// Order
	orderBy := "ORDER BY created_at DESC"
	if p.Sort == "popular" {
		orderBy = "ORDER BY download_count DESC"
	}

	limit := p.Limit
	if limit <= 0 {
		limit = 20
	}

	query := fmt.Sprintf("SELECT %s FROM wallpapers %s %s LIMIT $%d OFFSET $%d",
		wallpaperColumns, where, orderBy, argN, argN+1)
	args = append(args, limit, p.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list wallpapers: %w", err)
	}
	defer rows.Close()

	var wallpapers []*Wallpaper
	for rows.Next() {
		w, err := scanWallpaper(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan wallpaper: %w", err)
		}
		wallpapers = append(wallpapers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return wallpapers, total, nil
}

func (r *WallpaperRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallpapers SET status = $1, updated_at = now() WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (r *WallpaperRepo) UpdateMetadata(ctx context.Context, id, title, category string, tags []string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallpapers SET title = $1, category = $2, tags = $3, updated_at = now() WHERE id = $4`,
		title, category, pq.Array(tags), id,
	)
	if err != nil {
		return fmt.Errorf("update metadata: %w", err)
	}
	return nil
}

func (r *WallpaperRepo) UpdateAfterFinalize(ctx context.Context, id string, p FinalizeParams) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallpapers SET width = $1, height = $2, duration = $3, file_size = $4, format = $5, resolution = $6, thumbnail_key = $7, preview_key = $8, status = $9, updated_at = now() WHERE id = $10`,
		p.Width, p.Height, p.Duration, p.FileSize, p.Format, p.Resolution, p.ThumbnailKey, p.PreviewKey, p.Status, id,
	)
	if err != nil {
		return fmt.Errorf("update after finalize: %w", err)
	}
	return nil
}

func (r *WallpaperRepo) IncrementDownloadCount(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE wallpapers SET download_count = download_count + 1 WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("increment download count: %w", err)
	}
	return nil
}

func (r *WallpaperRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM wallpapers WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("delete wallpaper: %w", err)
	}
	return nil
}
