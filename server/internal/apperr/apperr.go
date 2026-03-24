package apperr

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

// Error is a structured error with an HTTP status, machine-readable code,
// and a human-readable message. The internal Err is never exposed in responses.
type Error struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// NotFound returns a 404 Error.
func NotFound(msg string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

// Forbidden returns a 403 Error.
func Forbidden(msg string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

// Conflict returns a 409 Error.
func Conflict(msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: "conflict", Message: msg}
}

// BadRequest returns a 400 Error.
func BadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "validation_failed", Message: msg}
}

// Internal returns a 500 Error wrapping an internal error.
func Internal(err error) *Error {
	return &Error{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
		Err:     err,
	}
}
