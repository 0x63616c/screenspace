package handler

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
)

// testDB opens a test database connection, cleans test data, and returns
// the db along with common repositories and an auth service.
// Tests that need a database should call this instead of duplicating setup.
type testDB struct {
	DB         *sql.DB
	Users      *repository.UserRepo
	Wallpapers *repository.WallpaperRepo
	Favorites  *repository.FavoriteRepo
	Reports    *repository.ReportRepo
	Auth       *service.AuthService
}

func newTestDB(t *testing.T) *testDB {
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

	return &testDB{
		DB:         db,
		Users:      repository.NewUserRepo(db),
		Wallpapers: repository.NewWallpaperRepo(db),
		Favorites:  repository.NewFavoriteRepo(db),
		Reports:    repository.NewReportRepo(db),
		Auth:       service.NewAuthService("test-secret"),
	}
}

// createUser is a shared helper to create a test user.
func (tdb *testDB) createUser(t *testing.T, email, role string) *repository.User {
	t.Helper()
	u, err := tdb.Users.Create(context.Background(), email, "hashedpw", role)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

// createApprovedWallpaper creates a wallpaper and sets its status to approved.
func (tdb *testDB) createApprovedWallpaper(t *testing.T, title, category, uploaderID string) *repository.Wallpaper {
	t.Helper()
	ctx := context.Background()
	wp, err := tdb.Wallpapers.Create(ctx, repository.CreateParams{
		Title:      title,
		UploaderID: uploaderID,
		StorageKey: fmt.Sprintf("wallpapers/test/%s.mp4", title),
	})
	if err != nil {
		t.Fatalf("create wallpaper: %v", err)
	}
	if category != "" {
		tdb.Wallpapers.UpdateMetadata(ctx, wp.ID, title, category, []string{})
	}
	if err := tdb.Wallpapers.UpdateStatus(ctx, wp.ID, "approved"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	wp, err = tdb.Wallpapers.GetByID(ctx, wp.ID)
	if err != nil {
		t.Fatalf("get wallpaper: %v", err)
	}
	return wp
}

// createWallpaperWithStatus creates a wallpaper and sets the given status.
func (tdb *testDB) createWallpaperWithStatus(t *testing.T, title, uploaderID, status string) *repository.Wallpaper {
	t.Helper()
	ctx := context.Background()
	wp, err := tdb.Wallpapers.Create(ctx, repository.CreateParams{
		Title:      title,
		UploaderID: uploaderID,
		StorageKey: fmt.Sprintf("wallpapers/test/%s.mp4", title),
	})
	if err != nil {
		t.Fatalf("create wallpaper: %v", err)
	}
	if err := tdb.Wallpapers.UpdateStatus(ctx, wp.ID, status); err != nil {
		t.Fatalf("update status: %v", err)
	}
	wp, _ = tdb.Wallpapers.GetByID(ctx, wp.ID)
	return wp
}

// authRequest creates an HTTP request with valid auth claims injected via middleware.
func (tdb *testDB) authRequest(t *testing.T, method, url, body, userID, role string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, url, bodyReader)

	token, err := tdb.Auth.GenerateToken(userID, role)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var captured *http.Request
	authMiddleware := middleware.Auth(tdb.Auth)
	handler := authMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if captured == nil {
		t.Fatal("auth middleware did not pass request through")
	}

	return httptest.NewRecorder(), captured
}

func newTestAuthHandler(t *testing.T) *AuthHandler {
	t.Helper()
	tdb := newTestDB(t)
	return NewAuthHandler(tdb.Users, tdb.Auth, "admin@test.com")
}
