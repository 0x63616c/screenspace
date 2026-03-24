package service_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/service"
)

func TestReportService_Create_EmptyReason(t *testing.T) {
	t.Parallel()
	svc := service.NewReportService(&db.MockQuerier{}, config.DefaultConfig())
	_, err := svc.Create(t.Context(), uuid.New(), uuid.New(), "")
	if appErr, ok := errors.AsType[*handler.AppError](err); !ok || appErr.Status != 400 {
		t.Errorf("expected 400 for empty reason, got %v", err)
	}
}

func TestReportService_Create_TooLong(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	svc := service.NewReportService(&db.MockQuerier{}, cfg)
	_, err := svc.Create(t.Context(), uuid.New(), uuid.New(), strings.Repeat("x", cfg.MaxReportLength+1))
	if appErr, ok := errors.AsType[*handler.AppError](err); !ok || appErr.Status != 400 {
		t.Errorf("expected 400 for too-long reason, got %v", err)
	}
}
