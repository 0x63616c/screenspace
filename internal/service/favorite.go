package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/handler"
)

// FavoriteService handles toggling and listing favorites.
type FavoriteService struct {
	db db.Querier
}

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
		return false, handler.Internal(fmt.Errorf("check favorite: %w", err))
	}

	if exists {
		if err := s.db.DeleteFavorite(ctx, db.DeleteFavoriteParams{
			UserID:      userID,
			WallpaperID: wallpaperID,
		}); err != nil {
			return false, handler.Internal(fmt.Errorf("delete favorite: %w", err))
		}
		return false, nil
	}

	if err := s.db.InsertFavorite(ctx, db.InsertFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	}); err != nil {
		return false, handler.Internal(fmt.Errorf("insert favorite: %w", err))
	}
	return true, nil
}
