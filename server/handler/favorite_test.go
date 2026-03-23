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

type testFavoriteEnv struct {
	handler    *FavoriteHandler
	users      *repository.UserRepo
	wallpapers *repository.WallpaperRepo
	favorites  *repository.FavoriteRepo
	auth       *service.AuthService
	db         *sql.DB
}

func newTestFavoriteHandler(t *testing.T) *testFavoriteEnv {
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
	favorites := repository.NewFavoriteRepo(db)
	auth := service.NewAuthService("test-secret")
	h := NewFavoriteHandler(favorites)

	return &testFavoriteEnv{
		handler:    h,
		users:      users,
		wallpapers: wallpapers,
		favorites:  favorites,
		auth:       auth,
		db:         db,
	}
}

func (env *testFavoriteEnv) createUser(t *testing.T, email, role string) *repository.User {
	t.Helper()
	u, err := env.users.Create(context.Background(), email, "hashedpw", role)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

func (env *testFavoriteEnv) createApprovedWallpaper(t *testing.T, title, uploaderID string) *repository.Wallpaper {
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

func (env *testFavoriteEnv) authRequest(t *testing.T, method, url, body, userID, role string) (*httptest.ResponseRecorder, *http.Request) {
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

func TestToggleFavorite_Add(t *testing.T) {
	env := newTestFavoriteHandler(t)
	u := env.createUser(t, "fav-add@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Fav WP", u.ID)

	w, r := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/favorite", "", u.ID, "user")
	r.SetPathValue("id", wp.ID)
	env.handler.Toggle(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp toggleFavoriteResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Favorited {
		t.Fatal("expected favorited=true")
	}
}

func TestToggleFavorite_Remove(t *testing.T) {
	env := newTestFavoriteHandler(t)
	u := env.createUser(t, "fav-remove@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Fav Remove WP", u.ID)

	// First toggle: add
	w1, r1 := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/favorite", "", u.ID, "user")
	r1.SetPathValue("id", wp.ID)
	env.handler.Toggle(w1, r1)

	// Second toggle: remove
	w2, r2 := env.authRequest(t, http.MethodPost, "/wallpapers/"+wp.ID+"/favorite", "", u.ID, "user")
	r2.SetPathValue("id", wp.ID)
	env.handler.Toggle(w2, r2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var resp toggleFavoriteResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Favorited {
		t.Fatal("expected favorited=false")
	}
}

func TestToggleFavorite_Unauthorized(t *testing.T) {
	env := newTestFavoriteHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/wallpapers/some-id/favorite", nil)
	req.SetPathValue("id", "some-id")
	w := httptest.NewRecorder()
	env.handler.Toggle(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListFavorites_Empty(t *testing.T) {
	env := newTestFavoriteHandler(t)
	u := env.createUser(t, "fav-empty@example.com", "user")

	w, r := env.authRequest(t, http.MethodGet, "/favorites", "", u.ID, "user")
	env.handler.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listFavoritesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Wallpapers) != 0 {
		t.Fatalf("expected 0, got %d", len(resp.Wallpapers))
	}
	if resp.Total != 0 {
		t.Fatalf("expected total 0, got %d", resp.Total)
	}
}

func TestListFavorites_WithItems(t *testing.T) {
	env := newTestFavoriteHandler(t)
	u := env.createUser(t, "fav-list@example.com", "user")
	wp1 := env.createApprovedWallpaper(t, "Fav List 1", u.ID)
	wp2 := env.createApprovedWallpaper(t, "Fav List 2", u.ID)

	// Favorite both
	env.favorites.Toggle(context.Background(), u.ID, wp1.ID)
	env.favorites.Toggle(context.Background(), u.ID, wp2.ID)

	w, r := env.authRequest(t, http.MethodGet, "/favorites", "", u.ID, "user")
	env.handler.List(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp listFavoritesResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Wallpapers) != 2 {
		t.Fatalf("expected 2, got %d", len(resp.Wallpapers))
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
}
