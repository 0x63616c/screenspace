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

type testAdminEnv struct {
	handler    *AdminHandler
	users      *repository.UserRepo
	wallpapers *repository.WallpaperRepo
	reports    *repository.ReportRepo
	favorites  *repository.FavoriteRepo
	auth       *service.AuthService
	db         *sql.DB
}

func newTestAdminHandler(t *testing.T) *testAdminEnv {
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
	favorites := repository.NewFavoriteRepo(db)
	auth := service.NewAuthService("test-secret")
	h := NewAdminHandler(wallpapers, users, reports)

	return &testAdminEnv{
		handler:    h,
		users:      users,
		wallpapers: wallpapers,
		reports:    reports,
		favorites:  favorites,
		auth:       auth,
		db:         db,
	}
}

func (env *testAdminEnv) createUser(t *testing.T, email, role string) *repository.User {
	t.Helper()
	u, err := env.users.Create(context.Background(), email, "hashedpw", role)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

func (env *testAdminEnv) createWallpaperWithStatus(t *testing.T, title, uploaderID, status string) *repository.Wallpaper {
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
	if err := env.wallpapers.UpdateStatus(ctx, wp.ID, status); err != nil {
		t.Fatalf("update status: %v", err)
	}
	wp, _ = env.wallpapers.GetByID(ctx, wp.ID)
	return wp
}

func (env *testAdminEnv) authRequest(t *testing.T, method, url, body, userID, role string) (*httptest.ResponseRecorder, *http.Request) {
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

func TestAdminQueue_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-queue@example.com", "admin")
	user := env.createUser(t, "uploader-queue@example.com", "user")

	env.createWallpaperWithStatus(t, "Pending WP", user.ID, "pending_review")

	w, r := env.authRequest(t, http.MethodGet, "/admin/queue", "", admin.ID, "admin")
	env.handler.Queue(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1, got %d", resp.Total)
	}
	if resp.Wallpapers[0].Status != "pending_review" {
		t.Fatalf("expected pending_review, got %s", resp.Wallpapers[0].Status)
	}
}

func TestAdminQueue_NonAdmin(t *testing.T) {
	env := newTestAdminHandler(t)
	user := env.createUser(t, "nonadmin-queue@example.com", "user")

	w, r := env.authRequest(t, http.MethodGet, "/admin/queue", "", user.ID, "user")
	env.handler.Queue(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAdminApprove_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-approve@example.com", "admin")
	user := env.createUser(t, "uploader-approve@example.com", "user")

	wp := env.createWallpaperWithStatus(t, "Approve WP", user.ID, "pending_review")

	w, r := env.authRequest(t, http.MethodPost, "/admin/wallpapers/"+wp.ID+"/approve", "", admin.ID, "admin")
	r.SetPathValue("id", wp.ID)
	env.handler.Approve(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify status changed
	updated, _ := env.wallpapers.GetByID(context.Background(), wp.ID)
	if updated.Status != "approved" {
		t.Fatalf("expected approved, got %s", updated.Status)
	}
}

func TestAdminReject_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-reject@example.com", "admin")
	user := env.createUser(t, "uploader-reject@example.com", "user")

	wp := env.createWallpaperWithStatus(t, "Reject WP", user.ID, "pending_review")

	body := `{"reason":"low quality"}`
	w, r := env.authRequest(t, http.MethodPost, "/admin/wallpapers/"+wp.ID+"/reject", body, admin.ID, "admin")
	r.SetPathValue("id", wp.ID)
	env.handler.Reject(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := env.wallpapers.GetByID(context.Background(), wp.ID)
	if updated.Status != "rejected" {
		t.Fatalf("expected rejected, got %s", updated.Status)
	}
}

func TestAdminListWallpapers_IncludesPending(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-listwp@example.com", "admin")
	user := env.createUser(t, "uploader-listwp@example.com", "user")

	env.createWallpaperWithStatus(t, "Pending WP", user.ID, "pending_review")
	env.createWallpaperWithStatus(t, "Approved WP", user.ID, "approved")

	w, r := env.authRequest(t, http.MethodGet, "/admin/wallpapers?status=pending_review", "", admin.ID, "admin")
	env.handler.ListWallpapers(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1 pending, got %d", resp.Total)
	}
}

func TestAdminEditWallpaper_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-edit@example.com", "admin")
	user := env.createUser(t, "uploader-edit@example.com", "user")

	wp := env.createWallpaperWithStatus(t, "Edit WP", user.ID, "approved")

	body := `{"title":"Updated Title","category":"abstract","tags":["cool","modern"]}`
	w, r := env.authRequest(t, http.MethodPut, "/admin/wallpapers/"+wp.ID, body, admin.ID, "admin")
	r.SetPathValue("id", wp.ID)
	env.handler.EditWallpaper(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := env.wallpapers.GetByID(context.Background(), wp.ID)
	if updated.Title != "Updated Title" {
		t.Fatalf("expected 'Updated Title', got '%s'", updated.Title)
	}
	if updated.Category != "abstract" {
		t.Fatalf("expected 'abstract', got '%s'", updated.Category)
	}
}

func TestAdminListUsers_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-listusers@example.com", "admin")
	env.createUser(t, "user1-listusers@example.com", "user")
	env.createUser(t, "user2-listusers@example.com", "user")

	w, r := env.authRequest(t, http.MethodGet, "/admin/users", "", admin.ID, "admin")
	env.handler.ListUsers(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listUsersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total < 3 {
		t.Fatalf("expected at least 3, got %d", resp.Total)
	}
}

func TestAdminBanUser_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-ban@example.com", "admin")
	target := env.createUser(t, "target-ban@example.com", "user")

	w, r := env.authRequest(t, http.MethodPost, "/admin/users/"+target.ID+"/ban", "", admin.ID, "admin")
	r.SetPathValue("id", target.ID)
	env.handler.BanUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := env.users.GetByID(context.Background(), target.ID)
	if !updated.Banned {
		t.Fatal("expected banned=true")
	}
}

func TestAdminUnbanUser_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-unban@example.com", "admin")
	target := env.createUser(t, "target-unban@example.com", "user")

	// Ban first
	env.users.SetBanned(context.Background(), target.ID, true)

	w, r := env.authRequest(t, http.MethodPost, "/admin/users/"+target.ID+"/unban", "", admin.ID, "admin")
	r.SetPathValue("id", target.ID)
	env.handler.UnbanUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := env.users.GetByID(context.Background(), target.ID)
	if updated.Banned {
		t.Fatal("expected banned=false")
	}
}

func TestAdminPromoteUser_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-promote@example.com", "admin")
	target := env.createUser(t, "target-promote@example.com", "user")

	w, r := env.authRequest(t, http.MethodPost, "/admin/users/"+target.ID+"/promote", "", admin.ID, "admin")
	r.SetPathValue("id", target.ID)
	env.handler.PromoteUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := env.users.GetByID(context.Background(), target.ID)
	if updated.Role != "admin" {
		t.Fatalf("expected role 'admin', got '%s'", updated.Role)
	}
}

func TestAdminBanUser_NonAdmin(t *testing.T) {
	env := newTestAdminHandler(t)
	user := env.createUser(t, "nonadmin-ban@example.com", "user")
	target := env.createUser(t, "target-nonadmin-ban@example.com", "user")

	w, r := env.authRequest(t, http.MethodPost, "/admin/users/"+target.ID+"/ban", "", user.ID, "user")
	r.SetPathValue("id", target.ID)
	env.handler.BanUser(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAdminListReports_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-listreports@example.com", "admin")
	user := env.createUser(t, "reporter-listreports@example.com", "user")
	wp := env.createWallpaperWithStatus(t, "Reported WP", user.ID, "approved")

	env.reports.Create(context.Background(), wp.ID, user.ID, "spam")

	w, r := env.authRequest(t, http.MethodGet, "/admin/reports", "", admin.ID, "admin")
	env.handler.ListReports(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listReportsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1, got %d", resp.Total)
	}
	if resp.Reports[0].Reason != "spam" {
		t.Fatalf("expected 'spam', got '%s'", resp.Reports[0].Reason)
	}
}

func TestAdminDismissReport_Success(t *testing.T) {
	env := newTestAdminHandler(t)
	admin := env.createUser(t, "admin-dismiss@example.com", "admin")
	user := env.createUser(t, "reporter-dismiss@example.com", "user")
	wp := env.createWallpaperWithStatus(t, "Dismiss WP", user.ID, "approved")

	report, _ := env.reports.Create(context.Background(), wp.ID, user.ID, "spam")

	w, r := env.authRequest(t, http.MethodPost, "/admin/reports/"+report.ID+"/dismiss", "", admin.ID, "admin")
	r.SetPathValue("id", report.ID)
	env.handler.DismissReport(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify no longer in pending
	reports, total, _ := env.reports.ListPending(context.Background(), 100, 0)
	if total != 0 {
		t.Fatalf("expected 0 pending, got %d", total)
	}
	_ = reports
}
