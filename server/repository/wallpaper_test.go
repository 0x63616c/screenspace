package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func newTestWallpaperRepo(t *testing.T) (*WallpaperRepo, *UserRepo) {
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
	db.Exec("DELETE FROM users WHERE email LIKE '%example.com'")

	return NewWallpaperRepo(db), NewUserRepo(db)
}

func createTestUser(t *testing.T, users *UserRepo, suffix string) *User {
	t.Helper()
	u, err := users.Create(context.Background(), fmt.Sprintf("wp-test-%s@example.com", suffix), "hashedpw", "user")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

func TestWallpaperRepo_CreateAndGetByID(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "create")

	w, err := repo.Create(ctx, CreateParams{
		Title:      "Test Wallpaper",
		UploaderID: u.ID,
		StorageKey: "wallpapers/test/original.mp4",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if w.Title != "Test Wallpaper" {
		t.Fatalf("expected title Test Wallpaper, got %s", w.Title)
	}
	if w.Status != "pending" {
		t.Fatalf("expected status pending, got %s", w.Status)
	}
	if w.UploaderID != u.ID {
		t.Fatalf("expected uploader_id %s, got %s", u.ID, w.UploaderID)
	}

	got, err := repo.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Title != w.Title {
		t.Fatalf("expected title %s, got %s", w.Title, got.Title)
	}
}

func TestWallpaperRepo_List_Empty(t *testing.T) {
	repo, _ := newTestWallpaperRepo(t)
	ctx := context.Background()

	wallpapers, total, err := repo.List(ctx, ListParams{Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(wallpapers) != 0 {
		t.Fatalf("expected 0 wallpapers, got %d", len(wallpapers))
	}
	if total != 0 {
		t.Fatalf("expected total 0, got %d", total)
	}
}

func TestWallpaperRepo_List_FilterByStatus(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "status-filter")

	w, err := repo.Create(ctx, CreateParams{Title: "Pending One", UploaderID: u.ID, StorageKey: "k1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Default list returns approved only, so pending should not appear
	wallpapers, total, err := repo.List(ctx, ListParams{Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected 0 approved, got %d", total)
	}

	// Explicitly list pending
	wallpapers, total, err = repo.List(ctx, ListParams{Status: "pending", Limit: 20})
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 pending, got %d", total)
	}
	if wallpapers[0].ID != w.ID {
		t.Fatalf("expected id %s, got %s", w.ID, wallpapers[0].ID)
	}
}

func TestWallpaperRepo_List_FilterByCategory(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "cat-filter")

	w1, _ := repo.Create(ctx, CreateParams{Title: "Nature", UploaderID: u.ID, StorageKey: "k1"})
	w2, _ := repo.Create(ctx, CreateParams{Title: "Abstract", UploaderID: u.ID, StorageKey: "k2"})

	repo.UpdateMetadata(ctx, w1.ID, "Nature", "nature", []string{})
	repo.UpdateStatus(ctx, w1.ID, "approved")
	repo.UpdateMetadata(ctx, w2.ID, "Abstract", "abstract", []string{})
	repo.UpdateStatus(ctx, w2.ID, "approved")

	wallpapers, total, err := repo.List(ctx, ListParams{Category: "nature", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1, got %d", total)
	}
	if wallpapers[0].Category != "nature" {
		t.Fatalf("expected category nature, got %s", wallpapers[0].Category)
	}
}

func TestWallpaperRepo_List_SearchByQuery(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "query-search")

	w1, _ := repo.Create(ctx, CreateParams{Title: "Mountain Sunset", UploaderID: u.ID, StorageKey: "k1"})
	w2, _ := repo.Create(ctx, CreateParams{Title: "Ocean Waves", UploaderID: u.ID, StorageKey: "k2"})

	repo.UpdateStatus(ctx, w1.ID, "approved")
	repo.UpdateStatus(ctx, w2.ID, "approved")

	wallpapers, total, err := repo.List(ctx, ListParams{Query: "mountain", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1, got %d", total)
	}
	if wallpapers[0].Title != "Mountain Sunset" {
		t.Fatalf("expected Mountain Sunset, got %s", wallpapers[0].Title)
	}
}

func TestWallpaperRepo_List_SortByPopular(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "sort-popular")

	w1, _ := repo.Create(ctx, CreateParams{Title: "Less Popular", UploaderID: u.ID, StorageKey: "k1"})
	w2, _ := repo.Create(ctx, CreateParams{Title: "More Popular", UploaderID: u.ID, StorageKey: "k2"})

	repo.UpdateStatus(ctx, w1.ID, "approved")
	repo.UpdateStatus(ctx, w2.ID, "approved")

	// Increment w2 downloads
	repo.IncrementDownloadCount(ctx, w2.ID)
	repo.IncrementDownloadCount(ctx, w2.ID)
	repo.IncrementDownloadCount(ctx, w2.ID)

	wallpapers, _, err := repo.List(ctx, ListParams{Sort: "popular", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(wallpapers) < 2 {
		t.Fatalf("expected at least 2, got %d", len(wallpapers))
	}
	if wallpapers[0].Title != "More Popular" {
		t.Fatalf("expected More Popular first, got %s", wallpapers[0].Title)
	}
}

func TestWallpaperRepo_List_SortByRecent(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "sort-recent")

	repo.Create(ctx, CreateParams{Title: "First", UploaderID: u.ID, StorageKey: "k1"})
	w2, _ := repo.Create(ctx, CreateParams{Title: "Second", UploaderID: u.ID, StorageKey: "k2"})

	// Both need to be approved for the default list
	// Use raw SQL to set different created_at times would be complex,
	// but insertion order + DESC should put Second first
	repo.UpdateStatus(ctx, w2.ID, "approved")

	wallpapers, _, err := repo.List(ctx, ListParams{Sort: "recent", Status: "approved", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(wallpapers) != 1 {
		t.Fatalf("expected 1, got %d", len(wallpapers))
	}
	if wallpapers[0].Title != "Second" {
		t.Fatalf("expected Second, got %s", wallpapers[0].Title)
	}
}

func TestWallpaperRepo_UpdateStatus(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "update-status")

	w, _ := repo.Create(ctx, CreateParams{Title: "Status Test", UploaderID: u.ID, StorageKey: "k1"})

	if err := repo.UpdateStatus(ctx, w.ID, "approved"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err := repo.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "approved" {
		t.Fatalf("expected approved, got %s", got.Status)
	}
}

func TestWallpaperRepo_UpdateMetadata(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "update-meta")

	w, _ := repo.Create(ctx, CreateParams{Title: "Old Title", UploaderID: u.ID, StorageKey: "k1"})

	if err := repo.UpdateMetadata(ctx, w.ID, "New Title", "nature", []string{"sunset", "mountain"}); err != nil {
		t.Fatalf("update metadata: %v", err)
	}

	got, err := repo.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "New Title" {
		t.Fatalf("expected New Title, got %s", got.Title)
	}
	if got.Category != "nature" {
		t.Fatalf("expected nature, got %s", got.Category)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(got.Tags))
	}
}

func TestWallpaperRepo_IncrementDownloadCount(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "inc-dl")

	w, _ := repo.Create(ctx, CreateParams{Title: "Download Test", UploaderID: u.ID, StorageKey: "k1"})

	for range 5 {
		if err := repo.IncrementDownloadCount(ctx, w.ID); err != nil {
			t.Fatalf("increment: %v", err)
		}
	}

	got, err := repo.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.DownloadCount != 5 {
		t.Fatalf("expected 5 downloads, got %d", got.DownloadCount)
	}
}

func TestWallpaperRepo_Delete(t *testing.T) {
	repo, users := newTestWallpaperRepo(t)
	ctx := context.Background()
	u := createTestUser(t, users, "delete")

	w, _ := repo.Create(ctx, CreateParams{Title: "Delete Me", UploaderID: u.ID, StorageKey: "k1"})

	if err := repo.Delete(ctx, w.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := repo.GetByID(ctx, w.ID)
	if err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}
