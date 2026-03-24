package handler

import "github.com/0x63616c/screenspace/server/internal/apperr"

// Re-export error types and constructors from apperr so handler callers
// don't need to know about the internal package split.
type AppError = apperr.Error

var (
	ErrNotFound   = apperr.ErrNotFound
	ErrForbidden  = apperr.ErrForbidden
	ErrConflict   = apperr.ErrConflict
	ErrBadRequest = apperr.ErrBadRequest
)

var (
	NotFound   = apperr.NotFound
	Forbidden  = apperr.Forbidden
	Conflict   = apperr.Conflict
	BadRequest = apperr.BadRequest
	Internal   = apperr.Internal
)
