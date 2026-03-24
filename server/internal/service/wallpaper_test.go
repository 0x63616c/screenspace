package service_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/apperr"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

func init() {
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func TestWallpaperService_Finalize_WrongStatus(t *testing.T) {
	t.Parallel()
	userID := uuid.New()
	wpID := uuid.New()
	mock := &db.MockQuerier{
		WallpaperRow: db.GetWallpaperByIDRow{
			ID:         wpID,
			UploaderID: userID,
			Status:     string(types.StatusPendingReview),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	_, err := svc.Finalize(t.Context(), wpID, userID)
	if appErr, ok := errors.AsType[*apperr.Error](err); !ok || appErr.Status != 400 {
		t.Errorf("expected 400 AppError, got %v", err)
	}
}

func TestWallpaperService_Finalize_WrongOwner(t *testing.T) {
	t.Parallel()
	wpID := uuid.New()
	mock := &db.MockQuerier{
		WallpaperRow: db.GetWallpaperByIDRow{
			ID:         wpID,
			UploaderID: uuid.New(),
			Status:     string(types.StatusPending),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	_, err := svc.Finalize(t.Context(), wpID, uuid.New())
	if appErr, ok := errors.AsType[*apperr.Error](err); !ok || appErr.Status != 403 {
		t.Errorf("expected 403 AppError, got %v", err)
	}
}

func TestWallpaperService_Approve_WrongStatus(t *testing.T) {
	t.Parallel()
	wpID := uuid.New()
	mock := &db.MockQuerier{
		WallpaperRow: db.GetWallpaperByIDRow{
			ID:     wpID,
			Status: string(types.StatusApproved),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	err := svc.Approve(t.Context(), wpID, uuid.New())
	if appErr, ok := errors.AsType[*apperr.Error](err); !ok || appErr.Status != 400 {
		t.Errorf("expected 400 AppError, got %v", err)
	}
}

func TestWallpaperService_GetApproved_NotApproved(t *testing.T) {
	t.Parallel()
	wpID := uuid.New()
	mock := &db.MockQuerier{
		WallpaperRow: db.GetWallpaperByIDRow{
			ID:     wpID,
			Status: string(types.StatusPending),
		},
	}
	cfg := config.DefaultConfig()
	svc := service.NewWallpaperService(mock, nil, nil, cfg)

	_, err := svc.GetApproved(t.Context(), wpID)
	if appErr, ok := errors.AsType[*apperr.Error](err); !ok || appErr.Status != 404 {
		t.Errorf("expected 404 AppError, got %v", err)
	}
}
