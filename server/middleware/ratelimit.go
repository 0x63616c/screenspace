package middleware

import (
	"net/http"
	"sync"
	"time"
)

type limiter struct {
	tokens    int
	lastReset time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	limits      map[string]*limiter
	maxPerDay   int
	callCount   int
	lastCleanup time.Time
}

func NewRateLimiter(maxPerDay int) *RateLimiter {
	return &RateLimiter{
		limits:      make(map[string]*limiter),
		maxPerDay:   maxPerDay,
		lastCleanup: time.Now(),
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Periodic cleanup: every 100 calls or every 5 minutes
	rl.callCount++
	if rl.callCount >= 100 || now.Sub(rl.lastCleanup) > 5*time.Minute {
		rl.cleanup(now)
		rl.callCount = 0
		rl.lastCleanup = now
	}

	l, exists := rl.limits[key]
	if !exists || now.Sub(l.lastReset) > 24*time.Hour {
		rl.limits[key] = &limiter{tokens: 1, lastReset: now}
		return true
	}

	if l.tokens >= rl.maxPerDay {
		return false
	}
	l.tokens++
	return true
}

// cleanup removes entries older than 24 hours. Must be called with mu held.
func (rl *RateLimiter) cleanup(now time.Time) {
	for key, l := range rl.limits {
		if now.Sub(l.lastReset) > 24*time.Hour {
			delete(rl.limits, key)
		}
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		key := r.RemoteAddr
		if claims != nil {
			key = claims.UserID
		}

		if !rl.Allow(key) {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
