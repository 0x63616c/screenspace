package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

type contextKey string

const claimsKey contextKey = "claims"

// Auth validates the Bearer token and stores claims in context.
// Returns 401 if missing or invalid.
func Auth(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := auth.ValidateToken(token)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext retrieves TokenClaims from the request context.
func ClaimsFromContext(ctx context.Context) *service.TokenClaims {
	claims, _ := ctx.Value(claimsKey).(*service.TokenClaims)
	return claims
}

// ContextWithClaims stores TokenClaims in the context. Exported for use in tests.
func ContextWithClaims(ctx context.Context, claims *service.TokenClaims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// Admin returns 403 if the authenticated user is not an admin.
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil || claims.Role != types.RoleAdmin {
			respond.Error(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
