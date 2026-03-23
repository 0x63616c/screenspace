package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(3)

	for i := 0; i < 3; i++ {
		if !rl.Allow("user-1") {
			t.Fatalf("expected allow on request %d", i+1)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(2)

	rl.Allow("user-1")
	rl.Allow("user-1")

	if rl.Allow("user-1") {
		t.Fatal("expected block after exceeding limit")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	rl := NewRateLimiter(1)

	rl.Allow("user-1")
	if rl.Allow("user-1") {
		t.Fatal("expected block")
	}

	// Simulate expiry by manipulating the internal state directly
	rl.mu.Lock()
	rl.limits["user-1"].lastReset = rl.limits["user-1"].lastReset.Add(-25 * 60 * 60 * 1e9) // 25 hours ago
	rl.mu.Unlock()

	if !rl.Allow("user-1") {
		t.Fatal("expected allow after window reset")
	}
}

func TestRateLimiter_SeparateKeys(t *testing.T) {
	rl := NewRateLimiter(1)

	if !rl.Allow("user-a") {
		t.Fatal("expected allow for user-a")
	}
	if !rl.Allow("user-b") {
		t.Fatal("expected allow for user-b")
	}
	if rl.Allow("user-a") {
		t.Fatal("expected block for user-a")
	}
}

func TestRateLimiterMiddleware_Returns429(t *testing.T) {
	rl := NewRateLimiter(1)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rl.Middleware(inner)

	// First request should pass
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second request should be rate limited
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}
