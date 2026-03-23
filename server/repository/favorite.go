package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type FavoriteRepo struct {
	db *sql.DB
}

func NewFavoriteRepo(db *sql.DB) *FavoriteRepo {
	return &FavoriteRepo{db: db}
}

// Toggle adds a favorite if it doesn't exist, or removes it if it does.
// Returns true if the favorite was added, false if removed.
func (r *FavoriteRepo) Toggle(ctx context.Context, userID, wallpaperID string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND wallpaper_id = $2)`,
		userID, wallpaperID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check favorite: %w", err)
	}

	if exists {
		_, err = tx.ExecContext(ctx,
			`DELETE FROM favorites WHERE user_id = $1 AND wallpaper_id = $2`,
			userID, wallpaperID,
		)
		if err != nil {
			return false, fmt.Errorf("delete favorite: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit: %w", err)
		}
		return false, nil
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO favorites (user_id, wallpaper_id) VALUES ($1, $2)`,
		userID, wallpaperID,
	)
	if err != nil {
		return false, fmt.Errorf("insert favorite: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

// ListByUser returns favorited wallpapers for a user with pagination.
func (r *FavoriteRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Wallpaper, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM favorites WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count favorites: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT w.id, w.title, w.uploader_id, w.status, COALESCE(w.category, ''), w.tags,
		        w.resolution, w.width, w.height, w.duration, w.file_size, w.format,
		        w.download_count, w.storage_key, w.thumbnail_key, w.preview_key,
		        w.created_at, w.updated_at
		 FROM favorites f
		 JOIN wallpapers w ON w.id = f.wallpaper_id
		 WHERE f.user_id = $1
		 ORDER BY f.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list favorites: %w", err)
	}
	defer rows.Close()

	var wallpapers []*Wallpaper
	for rows.Next() {
		w := &Wallpaper{}
		err := rows.Scan(
			&w.ID, &w.Title, &w.UploaderID, &w.Status, &w.Category,
			pq.Array(&w.Tags), &w.Resolution, &w.Width, &w.Height,
			&w.Duration, &w.FileSize, &w.Format, &w.DownloadCount,
			&w.StorageKey, &w.ThumbnailKey, &w.PreviewKey,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan favorite wallpaper: %w", err)
		}
		if w.Tags == nil {
			w.Tags = []string{}
		}
		wallpapers = append(wallpapers, w)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}
	return wallpapers, total, nil
}
