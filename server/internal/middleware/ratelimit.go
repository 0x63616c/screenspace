package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/0x63616c/screenspace/server/internal/respond"
)

type windowEntry struct {
	count     int
	windowEnd time.Time
}

// RateLimiter is a fixed-window rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*windowEntry
	max     int
	window  time.Duration
}

// NewRateLimiter creates a rate limiter with max requests per window duration.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]*windowEntry),
		max:     max,
		window:  window,
	}
}

// Allow returns true if the key is within rate limits. Thread-safe.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	e, ok := rl.entries[key]
	if !ok || now.After(e.windowEnd) {
		rl.entries[key] = &windowEntry{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if e.count >= rl.max {
		return false
	}
	e.count++
	return true
}

// PerIP returns middleware that rate-limits by remote IP.
func (rl *RateLimiter) PerIP() func(http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(rl.window.Seconds()))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(r.RemoteAddr) {
				w.Header().Set("Retry-After", retryAfter)
				respond.Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerUser returns middleware that rate-limits by authenticated user ID.
// Falls back to IP if no claims in context.
func (rl *RateLimiter) PerUser() func(http.Handler) http.Handler {
	retryAfter := strconv.Itoa(int(rl.window.Seconds()))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if claims := ClaimsFromContext(r.Context()); claims != nil {
				key = claims.UserID
			}
			if !rl.Allow(key) {
				w.Header().Set("Retry-After", retryAfter)
				respond.Error(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
