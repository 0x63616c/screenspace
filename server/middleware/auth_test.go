package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0x63616c/screenspace/server/service"
)

func TestAuthMiddleware_ValidToken(t *testing.T) {
	auth := service.NewAuthService("test-secret")
	token, _ := auth.GenerateToken("user-123", "user")

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims.UserID != "user-123" {
			t.Errorf("expected user-123, got %s", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	auth := service.NewAuthService("test-secret")

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	auth := service.NewAuthService("test-secret")

	handler := Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAdminMiddleware_AdminRole(t *testing.T) {
	auth := service.NewAuthService("test-secret")
	token, _ := auth.GenerateToken("admin-1", "admin")

	handler := Auth(auth)(Admin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAdminMiddleware_NonAdmin(t *testing.T) {
	auth := service.NewAuthService("test-secret")
	token, _ := auth.GenerateToken("user-123", "user")

	handler := Auth(auth)(Admin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestClaimsFromContext_NilWhenMissing(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	claims := ClaimsFromContext(req.Context())
	if claims != nil {
		t.Error("expected nil claims")
	}
}
