package handler

import "github.com/0x63616c/screenspace/server/internal/apperr"

// AppError is an alias for apperr.Error so handler callers
// don't need to know about the internal package split.
type AppError = apperr.Error

// Sentinel errors re-exported from apperr.
var (
	ErrNotFound   = apperr.ErrNotFound   //nolint:revive // re-export
	ErrForbidden  = apperr.ErrForbidden  //nolint:revive // re-export
	ErrConflict   = apperr.ErrConflict   //nolint:revive // re-export
	ErrBadRequest = apperr.ErrBadRequest //nolint:revive // re-export
)

// Error constructors re-exported from apperr.
var (
	NotFound   = apperr.NotFound   //nolint:revive // re-export
	Forbidden  = apperr.Forbidden  //nolint:revive // re-export
	Conflict   = apperr.Conflict   //nolint:revive // re-export
	BadRequest = apperr.BadRequest //nolint:revive // re-export
	Internal   = apperr.Internal   //nolint:revive // re-export
)
