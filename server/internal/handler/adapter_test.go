package handler_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/handler"
)

func init() {
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func TestWrap_NoError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWrap_AppError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return handler.NotFound("wallpaper not found")
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestWrap_SentinelError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return handler.ErrForbidden
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestWrap_UnknownError(t *testing.T) {
	t.Parallel()
	h := handler.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("something exploded")
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
