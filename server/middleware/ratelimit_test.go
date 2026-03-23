package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestRateLimiter_CleansUpStaleEntries(t *testing.T) {
	rl := NewRateLimiter(5)

	// Add some entries
	for i := 0; i < 10; i++ {
		rl.Allow("stale-user-" + string(rune('a'+i)))
	}

	// Verify entries exist
	rl.mu.Lock()
	if len(rl.limits) != 10 {
		t.Fatalf("expected 10 entries, got %d", len(rl.limits))
	}

	// Make all entries stale (25 hours old)
	for _, l := range rl.limits {
		l.lastReset = time.Now().Add(-25 * time.Hour)
	}
	rl.mu.Unlock()

	// Trigger cleanup by making 100 calls with a new key
	for i := 0; i < 100; i++ {
		rl.Allow("trigger-cleanup")
	}

	// Verify stale entries were removed
	rl.mu.Lock()
	defer rl.mu.Unlock()
	// Only "trigger-cleanup" should remain
	if len(rl.limits) != 1 {
		t.Fatalf("expected 1 entry after cleanup, got %d", len(rl.limits))
	}
	if _, exists := rl.limits["trigger-cleanup"]; !exists {
		t.Fatal("expected trigger-cleanup entry to exist")
	}
}

func TestRateLimiter_CleansUpByTime(t *testing.T) {
	rl := NewRateLimiter(5)

	rl.Allow("old-user")

	// Make entry stale and force time-based cleanup
	rl.mu.Lock()
	rl.limits["old-user"].lastReset = time.Now().Add(-25 * time.Hour)
	rl.lastCleanup = time.Now().Add(-6 * time.Minute)
	rl.mu.Unlock()

	// Single call should trigger time-based cleanup
	rl.Allow("new-user")

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, exists := rl.limits["old-user"]; exists {
		t.Fatal("expected old-user to be cleaned up")
	}
	if _, exists := rl.limits["new-user"]; !exists {
		t.Fatal("expected new-user to exist")
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
