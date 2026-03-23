package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
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

type testReportEnv struct {
	handler    *ReportHandler
	users      *repository.UserRepo
	wallpapers *repository.WallpaperRepo
	reports    *repository.ReportRepo
	auth       *service.AuthService
	db         *sql.DB
}

func newTestReportHandler(t *testing.T) *testReportEnv {
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

	users := repository.NewUserRepo(db)
	wallpapers := repository.NewWallpaperRepo(db)
	reports := repository.NewReportRepo(db)
	auth := service.NewAuthService("test-secret")
	h := NewReportHandler(reports)

	return &testReportEnv{
		handler:    h,
		users:      users,
		wallpapers: wallpapers,
		reports:    reports,
		auth:       auth,
		db:         db,
	}
}

func (env *testReportEnv) createUser(t *testing.T, email, role string) *repository.User {
	t.Helper()
	u, err := env.users.Create(context.Background(), email, "hashedpw", role)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

func (env *testReportEnv) createApprovedWallpaper(t *testing.T, title, uploaderID string) *repository.Wallpaper {
	t.Helper()
	ctx := context.Background()
	wp, err := env.wallpapers.Create(ctx, repository.CreateParams{
		Title:      title,
		UploaderID: uploaderID,
		StorageKey: fmt.Sprintf("wallpapers/test/%s.mp4", title),
	})
	if err != nil {
		t.Fatalf("create wallpaper: %v", err)
	}
	if err := env.wallpapers.UpdateStatus(ctx, wp.ID, "approved"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	wp, _ = env.wallpapers.GetByID(ctx, wp.ID)
	return wp
}

func (env *testReportEnv) authRequest(t *testing.T, method, url, body, userID, role string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, url, bodyReader)

	token, err := env.auth.GenerateToken(userID, role)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var captured *http.Request
	authMiddleware := middleware.Auth(env.auth)
	handler := authMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if captured == nil {
		t.Fatal("auth middleware did not pass request through")
	}

	return httptest.NewRecorder(), captured
}

func TestReportWallpaper_Success(t *testing.T) {
	env := newTestReportHandler(t)
	u := env.createUser(t, "reporter@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Report WP", u.ID)

	body := `{"reason":"inappropriate content"}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/report", body, u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp reportResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID == "" {
		t.Fatal("expected non-empty report ID")
	}
	if resp.Reason != "inappropriate content" {
		t.Fatalf("expected reason 'inappropriate content', got %s", resp.Reason)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected status 'pending', got %s", resp.Status)
	}
}

func TestReportWallpaper_MissingReason(t *testing.T) {
	env := newTestReportHandler(t)
	u := env.createUser(t, "reporter-noreason@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Report NoReason WP", u.ID)

	body := `{}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/report", body, u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportWallpaper_Unauthorized(t *testing.T) {
	env := newTestReportHandler(t)

	body := `{"reason":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/wallpapers/some-id/report", bytes.NewBufferString(body))
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()
	env.handler.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
