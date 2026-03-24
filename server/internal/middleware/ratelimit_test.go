package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(3, time.Minute)
	for i := range 3 {
		if !rl.Allow("key") {
			t.Errorf("request %d should be allowed", i)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(2, time.Minute)
	rl.Allow("key")
	rl.Allow("key")
	if rl.Allow("key") {
		t.Error("third request should be blocked")
	}
}

func TestRateLimiter_ResetsAfterWindow(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(1, 10*time.Millisecond)
	rl.Allow("key")
	if rl.Allow("key") {
		t.Error("should be blocked before window expires")
	}
	time.Sleep(15 * time.Millisecond)
	if !rl.Allow("key") {
		t.Error("should be allowed after window expires")
	}
}

func TestRateLimiter_PerIP_Blocks(t *testing.T) {
	t.Parallel()
	rl := middleware.NewRateLimiter(1, time.Minute)
	h := rl.PerIP()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i, wantCode := range []int{http.StatusOK, http.StatusTooManyRequests} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "1.2.3.4:9999"
		h.ServeHTTP(w, r)
		if w.Code != wantCode {
			t.Errorf("request %d: expected %d, got %d", i, wantCode, w.Code)
		}
	}
}
