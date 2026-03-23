package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func newTestFavoriteRepo(t *testing.T) (*FavoriteRepo, *UserRepo, *WallpaperRepo, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://screenspace:devpassword@localhost:5432/screenspace?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skipping, no database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("skipping, database unreachable: %v", err)
	}

	db.Exec("DELETE FROM favorites")
	db.Exec("DELETE FROM reports")
	db.Exec("DELETE FROM wallpapers")
	db.Exec("DELETE FROM users WHERE email LIKE '%repo.test'")

	return NewFavoriteRepo(db), NewUserRepo(db), NewWallpaperRepo(db), db
}

func TestFavoriteToggle_AddAndRemove(t *testing.T) {
	favs, users, wps, _ := newTestFavoriteRepo(t)
	ctx := context.Background()

	u, err := users.Create(ctx, "fav-toggle@repo.test", "hash", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	wp, err := wps.Create(ctx, CreateParams{Title: "Fav Test", UploaderID: u.ID, StorageKey: "k1"})
	if err != nil {
		t.Fatalf("create wallpaper: %v", err)
	}

	// Add
	added, err := favs.Toggle(ctx, u.ID, wp.ID)
	if err != nil {
		t.Fatalf("toggle add: %v", err)
	}
	if !added {
		t.Fatal("expected added=true")
	}

	// Remove
	removed, err := favs.Toggle(ctx, u.ID, wp.ID)
	if err != nil {
		t.Fatalf("toggle remove: %v", err)
	}
	if removed {
		t.Fatal("expected added=false")
	}
}

func TestFavoriteListByUser(t *testing.T) {
	favs, users, wps, _ := newTestFavoriteRepo(t)
	ctx := context.Background()

	u, _ := users.Create(ctx, "fav-list@repo.test", "hash", "user")
	wp1, _ := wps.Create(ctx, CreateParams{Title: "Fav 1", UploaderID: u.ID, StorageKey: "k1"})
	wp2, _ := wps.Create(ctx, CreateParams{Title: "Fav 2", UploaderID: u.ID, StorageKey: "k2"})

	favs.Toggle(ctx, u.ID, wp1.ID)
	favs.Toggle(ctx, u.ID, wp2.ID)

	wallpapers, total, err := favs.ListByUser(ctx, u.ID, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(wallpapers) != 2 {
		t.Fatalf("expected 2, got %d", len(wallpapers))
	}
}

func TestFavoriteListByUser_Empty(t *testing.T) {
	favs, users, _, _ := newTestFavoriteRepo(t)
	ctx := context.Background()

	u, _ := users.Create(ctx, "fav-empty@repo.test", "hash", "user")

	wallpapers, total, err := favs.ListByUser(ctx, u.ID, 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
	if wallpapers != nil {
		t.Fatalf("expected nil, got %v", wallpapers)
	}
}
