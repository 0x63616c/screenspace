package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testFavoriteEnv struct {
	*testDB
	handler *FavoriteHandler
}

func newTestFavoriteHandler(t *testing.T) *testFavoriteEnv {
	t.Helper()
	tdb := newTestDB(t)
	h := NewFavoriteHandler(tdb.Favorites)
	return &testFavoriteEnv{testDB: tdb, handler: h}
}

func TestToggleFavorite_Add(t *testing.T) {
	env := newTestFavoriteHandler(t)
	u := env.createUser(t, "fav-add@example.com", "user")
	wp := env.createApprovedWallpaper(t, "Fav WP", "", u.ID)

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
	wp := env.createApprovedWallpaper(t, "Fav Remove WP", "", u.ID)

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
	wp1 := env.createApprovedWallpaper(t, "Fav List 1", "", u.ID)
	wp2 := env.createApprovedWallpaper(t, "Fav List 2", "", u.ID)

	// Favorite both
	env.Favorites.Toggle(context.Background(), u.ID, wp1.ID)
	env.Favorites.Toggle(context.Background(), u.ID, wp2.ID)

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
