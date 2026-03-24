package handler

import (
	"errors"
	"net/http"
)

// Sentinel errors used across the service and handler layers.
var (
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
	ErrBadRequest = errors.New("bad request")
)

// AppError is a structured error with an HTTP status, machine-readable code,
// and a human-readable message. The internal Err is never exposed in responses.
type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

// NotFound returns a 404 AppError.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

// Forbidden returns a 403 AppError.
func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

// Conflict returns a 409 AppError.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: "conflict", Message: msg}
}

// BadRequest returns a 400 AppError.
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: "validation_failed", Message: msg}
}

// Internal returns a 500 AppError wrapping an internal error.
func Internal(err error) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
		Err:     err,
	}
}
