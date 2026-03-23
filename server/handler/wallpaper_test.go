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
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
	"github.com/0x63616c/screenspace/server/storage"
)

// mockStore implements storage.Store for testing without real S3.
type mockStore struct {
	objects map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{objects: make(map[string][]byte)}
}

func (m *mockStore) Put(_ context.Context, key string, reader io.Reader, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.objects[key] = data
	return nil
}

func (m *mockStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *mockStore) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func (m *mockStore) Stat(_ context.Context, key string) (*storage.ObjectInfo, error) {
	data, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("not found: %s", key)
	}
	return &storage.ObjectInfo{Key: key, Size: int64(len(data))}, nil
}

func (m *mockStore) List(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (m *mockStore) PreSignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://mock-s3.local/" + key + "?signed=true", nil
}

func (m *mockStore) PreSignedUploadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://mock-s3.local/upload/" + key + "?signed=true", nil
}

type testWallpaperEnv struct {
	handler    *WallpaperHandler
	users      *repository.UserRepo
	wallpapers *repository.WallpaperRepo
	auth       *service.AuthService
	store      *mockStore
	db         *sql.DB
}

func newTestWallpaperHandler(t *testing.T) *testWallpaperEnv {
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
	auth := service.NewAuthService("test-secret")
	store := newMockStore()
	video := service.NewVideoService()

	h := NewWallpaperHandler(wallpapers, store, video, auth)

	return &testWallpaperEnv{
		handler:    h,
		users:      users,
		wallpapers: wallpapers,
		auth:       auth,
		store:      store,
		db:         db,
	}
}

func (env *testWallpaperEnv) createUser(t *testing.T, email, role string) *repository.User {
	t.Helper()
	u, err := env.users.Create(context.Background(), email, "hashedpw", role)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

// authRequest creates a request with valid auth claims injected via middleware.
// It runs the request through middleware.Auth to properly set the context.
func (env *testWallpaperEnv) authRequest(t *testing.T, method, url, body, userID, role string) (*httptest.ResponseRecorder, *http.Request) {
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

	// Run through middleware.Auth to inject claims into context properly
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

func TestCreateWallpaper_Success(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "create-wp@example.com", "user")

	body := `{"title":"My Wallpaper","category":"nature","tags":["sunset"]}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers", body, u.ID, "user")
	env.handler.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp createWallpaperResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if resp.UploadURL == "" {
		t.Fatal("expected non-empty upload URL")
	}
}

func TestCreateWallpaper_MissingTitle(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "create-notitle@example.com", "user")

	body := `{"category":"nature"}`
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers", body, u.ID, "user")
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWallpaper_TitleTooLong(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "create-longtitle@example.com", "user")

	longTitle := strings.Repeat("a", 256)
	body := fmt.Sprintf(`{"title":"%s"}`, longTitle)
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers", body, u.ID, "user")
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWallpaper_TooManyTags(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "create-manytags@example.com", "user")

	tags := make([]string, 11)
	for i := range tags {
		tags[i] = fmt.Sprintf("tag%d", i)
	}
	tagsJSON, _ := json.Marshal(tags)
	body := fmt.Sprintf(`{"title":"Test","tags":%s}`, tagsJSON)
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers", body, u.ID, "user")
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWallpaper_TagTooLong(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "create-longtag@example.com", "user")

	longTag := strings.Repeat("b", 51)
	body := fmt.Sprintf(`{"title":"Test","tags":["%s"]}`, longTag)
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers", body, u.ID, "user")
	env.handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateWallpaper_Unauthorized(t *testing.T) {
	env := newTestWallpaperHandler(t)

	body := `{"title":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/wallpapers", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	env.handler.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListWallpapers_Empty(t *testing.T) {
	env := newTestWallpaperHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/wallpapers", nil)
	w := httptest.NewRecorder()
	env.handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listWallpapersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Wallpapers) != 0 {
		t.Fatalf("expected 0, got %d", len(resp.Wallpapers))
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
}

func (env *testWallpaperEnv) createApprovedWallpaper(t *testing.T, title, category string, uploaderID string) *repository.Wallpaper {
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
	if category != "" {
		env.wallpapers.UpdateMetadata(ctx, wp.ID, title, category, []string{})
	}
	if err := env.wallpapers.UpdateStatus(ctx, wp.ID, "approved"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	// Re-fetch to get updated fields
	wp, err = env.wallpapers.GetByID(ctx, wp.ID)
	if err != nil {
		t.Fatalf("get wallpaper: %v", err)
	}
	return wp
}

func TestListWallpapers_Paginated(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "list-page@example.com", "user")

	for i := range 5 {
		env.createApprovedWallpaper(t, fmt.Sprintf("WP %d", i), "", u.ID)
	}

	req := httptest.NewRequest(http.MethodGet, "/wallpapers?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	env.handler.List(w, req)

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 5 {
		t.Fatalf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Wallpapers) != 2 {
		t.Fatalf("expected 2 in page, got %d", len(resp.Wallpapers))
	}
}

func TestListWallpapers_FilterByCategory(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "list-cat@example.com", "user")

	env.createApprovedWallpaper(t, "Nature WP", "nature", u.ID)
	env.createApprovedWallpaper(t, "Abstract WP", "abstract", u.ID)

	req := httptest.NewRequest(http.MethodGet, "/wallpapers?category=nature", nil)
	w := httptest.NewRecorder()
	env.handler.List(w, req)

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1, got %d", resp.Total)
	}
}

func TestListWallpapers_SortByPopular(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "list-pop@example.com", "user")

	env.createApprovedWallpaper(t, "Unpopular", "", u.ID)
	pop := env.createApprovedWallpaper(t, "Popular", "", u.ID)

	for range 10 {
		env.wallpapers.IncrementDownloadCount(context.Background(), pop.ID)
	}

	req := httptest.NewRequest(http.MethodGet, "/wallpapers?sort=popular", nil)
	w := httptest.NewRecorder()
	env.handler.List(w, req)

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Wallpapers) < 2 {
		t.Fatalf("expected at least 2, got %d", len(resp.Wallpapers))
	}
	if resp.Wallpapers[0].Title != "Popular" {
		t.Fatalf("expected Popular first, got %s", resp.Wallpapers[0].Title)
	}
}

func TestListWallpapers_OnlyApproved(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "list-approved@example.com", "user")

	// Create one approved, one pending
	env.createApprovedWallpaper(t, "Approved WP", "", u.ID)
	env.wallpapers.Create(context.Background(), repository.CreateParams{
		Title:      "Pending WP",
		UploaderID: u.ID,
		StorageKey: "k-pending",
	})

	req := httptest.NewRequest(http.MethodGet, "/wallpapers", nil)
	w := httptest.NewRecorder()
	env.handler.List(w, req)

	var resp listWallpapersResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Total != 1 {
		t.Fatalf("expected 1 approved, got %d", resp.Total)
	}
	if resp.Wallpapers[0].Title != "Approved WP" {
		t.Fatalf("expected Approved WP, got %s", resp.Wallpapers[0].Title)
	}
}

func TestGetWallpaper_Success(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "get-wp@example.com", "user")

	wp := env.createApprovedWallpaper(t, "Get Test", "", u.ID)

	req := httptest.NewRequest(http.MethodGet, "/wallpapers/"+wp.ID, nil)
	req.SetPathValue("id", wp.ID)
	w := httptest.NewRecorder()
	env.handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp wallpaperResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != wp.ID {
		t.Fatalf("expected id %s, got %s", wp.ID, resp.ID)
	}
}

func TestGetWallpaper_DoesNotIncrementDownloadCount(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "get-noinc@example.com", "user")

	wp := env.createApprovedWallpaper(t, "No Inc Test", "", u.ID)

	// Get wallpaper multiple times
	for range 3 {
		req := httptest.NewRequest(http.MethodGet, "/wallpapers/"+wp.ID, nil)
		req.SetPathValue("id", wp.ID)
		w := httptest.NewRecorder()
		env.handler.Get(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	}

	// Verify download count is still 0
	updated, err := env.wallpapers.GetByID(context.Background(), wp.ID)
	if err != nil {
		t.Fatalf("get wallpaper: %v", err)
	}
	if updated.DownloadCount != 0 {
		t.Fatalf("expected download_count 0, got %d", updated.DownloadCount)
	}
}

func TestDownloadWallpaper_IncrementsCount(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "dl-inc@example.com", "user")

	wp := env.createApprovedWallpaper(t, "DL Inc Test", "", u.ID)

	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/download", "", u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Download(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp downloadResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.DownloadURL == "" {
		t.Fatal("expected non-empty download URL")
	}

	// Verify download count incremented
	updated, err := env.wallpapers.GetByID(context.Background(), wp.ID)
	if err != nil {
		t.Fatalf("get wallpaper: %v", err)
	}
	if updated.DownloadCount != 1 {
		t.Fatalf("expected download_count 1, got %d", updated.DownloadCount)
	}
}

func TestDownloadWallpaper_Unauthorized(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "dl-unauth@example.com", "user")

	wp := env.createApprovedWallpaper(t, "DL Unauth Test", "", u.ID)

	req := httptest.NewRequest(http.MethodPost, "/wallpapers/"+wp.ID+"/download", nil)
	req.SetPathValue("id", wp.ID)
	w := httptest.NewRecorder()
	env.handler.Download(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetWallpaper_NotFound(t *testing.T) {
	env := newTestWallpaperHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/wallpapers/00000000-0000-0000-0000-000000000000", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000000")
	w := httptest.NewRecorder()
	env.handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDeleteWallpaper_AsOwner(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "del-owner@example.com", "user")

	wp := env.createApprovedWallpaper(t, "Delete Owner", "", u.ID)

	w, r := env.authRequest(t, http.MethodDelete, "/wallpapers/"+wp.ID, "", u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	_, err := env.wallpapers.GetByID(context.Background(), wp.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteWallpaper_AsAdmin(t *testing.T) {
	env := newTestWallpaperHandler(t)
	owner := env.createUser(t, "del-admin-owner@example.com", "user")
	admin := env.createUser(t, "del-admin@example.com", "admin")

	wp := env.createApprovedWallpaper(t, "Delete Admin", "", owner.ID)

	w, r := env.authRequest(t, http.MethodDelete, "/wallpapers/"+wp.ID, "", admin.ID, "admin")
	r.SetPathValue("id", wp.ID)
	env.handler.Delete(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteWallpaper_AsOtherUser(t *testing.T) {
	env := newTestWallpaperHandler(t)
	owner := env.createUser(t, "del-other-owner@example.com", "user")
	other := env.createUser(t, "del-other@example.com", "user")

	wp := env.createApprovedWallpaper(t, "Delete Other", "", owner.ID)

	w, r := env.authRequest(t, http.MethodDelete, "/wallpapers/"+wp.ID, "", other.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Delete(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteWallpaper_NotFound(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "del-notfound@example.com", "user")

	fakeID := "00000000-0000-0000-0000-000000000000"
	w, r := env.authRequest(t, http.MethodDelete, "/wallpapers/"+fakeID, "", u.ID, "user")
	r.SetPathValue("id", fakeID)
	env.handler.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFinalize_NotFound(t *testing.T) {
	env := newTestWallpaperHandler(t)
	u := env.createUser(t, "finalize-notfound@example.com", "user")

	fakeID := "00000000-0000-0000-0000-000000000000"
	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+fakeID+"/finalize", "", u.ID, "user")
	r.SetPathValue("id", fakeID)
	env.handler.Finalize(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFinalize_NotOwner(t *testing.T) {
	env := newTestWallpaperHandler(t)
	owner := env.createUser(t, "finalize-owner@example.com", "user")
	other := env.createUser(t, "finalize-other@example.com", "user")

	wp, err := env.wallpapers.Create(context.Background(), repository.CreateParams{
		Title:      "Finalize Test",
		UploaderID: owner.ID,
		StorageKey: "wallpapers/test/original.mp4",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/finalize", "", other.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Finalize(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
