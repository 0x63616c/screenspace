package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/apperr"
)

// FavoriteService handles toggling and listing favorites.
type FavoriteService struct {
	db db.Querier
}

// NewFavoriteService creates a new FavoriteService.
func NewFavoriteService(q db.Querier) *FavoriteService {
	return &FavoriteService{db: q}
}

// Toggle adds a favorite if absent, removes it if present.
// Returns true if the wallpaper is now favorited.
func (s *FavoriteService) Toggle(ctx context.Context, userID, wallpaperID uuid.UUID) (bool, error) {
	exists, err := s.db.CheckFavorite(ctx, db.CheckFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	})
	if err != nil {
		return false, apperr.Internal(fmt.Errorf("check favorite: %w", err))
	}

	if exists {
		if err := s.db.DeleteFavorite(ctx, db.DeleteFavoriteParams{
			UserID:      userID,
			WallpaperID: wallpaperID,
		}); err != nil {
			return false, apperr.Internal(fmt.Errorf("delete favorite: %w", err))
		}
		return false, nil
	}

	if err := s.db.InsertFavorite(ctx, db.InsertFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	}); err != nil {
		return false, apperr.Internal(fmt.Errorf("insert favorite: %w", err))
	}
	return true, nil
}

// ListByUser returns paginated favorites for a user and the total count.
func (s *FavoriteService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]db.ListFavoritesByUserRow, int64, error) {
	total, err := s.db.CountFavoritesByUser(ctx, userID)
	if err != nil {
		return nil, 0, apperr.Internal(fmt.Errorf("count favorites: %w", err))
	}

	rows, err := s.db.ListFavoritesByUser(ctx, db.ListFavoritesByUserParams{
		UserID: userID,
		Off:    int32(offset),
		Lim:    int32(limit),
	})
	if err != nil {
		return nil, 0, apperr.Internal(fmt.Errorf("list favorites: %w", err))
	}

	return rows, total, nil
}
