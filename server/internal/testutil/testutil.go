package testutil

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// RequestWithClaims injects TokenClaims into the request context so that
// middleware.ClaimsFromContext returns them in handler tests.
func RequestWithClaims(r *http.Request, userID string, role types.UserRole) *http.Request {
	claims := &service.TokenClaims{UserID: userID, Role: role}
	ctx := middleware.ContextWithClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

// RequestWithChiParam injects a chi URL parameter into the request context.
func RequestWithChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// RequestWithChiParams injects multiple chi URL parameters into the request context.
func RequestWithChiParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}
