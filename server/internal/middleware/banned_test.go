package middleware_test

import (
	"log/slog"
	"testing"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

func init() {
	slog.SetDefault(slog.New(slog.DiscardHandler))
}

func TestBannedCache_SetAndGet(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	c.Set("u1", true)

	banned, ok := c.IsBanned("u1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !banned {
		t.Error("expected banned=true")
	}
}

func TestBannedCache_Evict(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	c.Set("u1", true)
	c.Evict("u1")

	_, ok := c.IsBanned("u1")
	if ok {
		t.Error("expected cache miss after evict")
	}
}

func TestBannedCache_Miss(t *testing.T) {
	t.Parallel()
	c := middleware.NewBannedCache()
	_, ok := c.IsBanned("nobody")
	if ok {
		t.Error("expected cache miss for unknown user")
	}
}
