package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/respond"
)

const bannedCacheTTL = 60 * time.Second

type bannedCacheEntry struct {
	banned    bool
	expiresAt time.Time
}

// BannedCache is a thread-safe in-memory TTL cache for banned user status.
type BannedCache struct {
	mu      sync.Mutex
	entries map[string]bannedCacheEntry
}

// NewBannedCache creates a new empty BannedCache.
func NewBannedCache() *BannedCache {
	return &BannedCache{entries: make(map[string]bannedCacheEntry)}
}

// IsBanned checks the cache. Returns (banned, ok) where ok=false means cache miss.
func (c *BannedCache) IsBanned(userID string) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[userID]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, userID)
		return false, false
	}
	return e.banned, true
}

// Set stores a banned status with TTL.
func (c *BannedCache) Set(userID string, banned bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[userID] = bannedCacheEntry{
		banned:    banned,
		expiresAt: time.Now().Add(bannedCacheTTL),
	}
}

// Evict removes a user from the cache immediately (call on ban action).
func (c *BannedCache) Evict(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, userID)
}

// BannedCheck returns middleware that checks if the authenticated user is banned.
// Uses the provided cache with 60s TTL; falls through to DB on cache miss.
func BannedCheck(q db.Querier, cache *BannedCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Cache hit.
			if banned, ok := cache.IsBanned(claims.UserID); ok {
				if banned {
					respond.Error(w, http.StatusForbidden, "banned", "account banned")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Cache miss: query DB.
			userUUID, err := uuid.Parse(claims.UserID)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "invalid user id")
				return
			}
			user, err := q.GetUserByID(r.Context(), userUUID)
			if err != nil || user.Banned {
				cache.Set(claims.UserID, true)
				respond.Error(w, http.StatusForbidden, "banned", "account banned")
				return
			}
			cache.Set(claims.UserID, false)
			next.ServeHTTP(w, r)
		})
	}
}
