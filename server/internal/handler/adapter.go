package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/0x63616c/screenspace/server/internal/apperr"
	"github.com/0x63616c/screenspace/server/internal/respond"
)

// HandlerFunc is an http.HandlerFunc that returns an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Wrap converts a HandlerFunc into an http.HandlerFunc. If the handler
// returns an error, it is mapped to an appropriate HTTP response.
func Wrap(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			handleError(w, r, err)
		}
	}
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	if appErr, ok := errors.AsType[*apperr.Error](err); ok {
		if appErr.Status >= 500 {
			slog.Error("request error",
				"method", r.Method,
				"path", r.URL.Path,
				"status", appErr.Status,
				"error", appErr.Err,
			)
		}
		respond.Error(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}

	// Sentinel error mapping for errors returned directly from services.
	switch {
	case errors.Is(err, apperr.ErrNotFound):
		respond.Error(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, apperr.ErrForbidden):
		respond.Error(w, http.StatusForbidden, "forbidden", "forbidden")
	case errors.Is(err, apperr.ErrConflict):
		respond.Error(w, http.StatusConflict, "conflict", "conflict")
	case errors.Is(err, apperr.ErrBadRequest):
		respond.Error(w, http.StatusBadRequest, "bad_request", "bad request")
	default:
		slog.Error("unhandled request error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		respond.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
