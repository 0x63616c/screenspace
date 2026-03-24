package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/0x63616c/screenspace/server/db/generated"
)

type FavoriteHandler struct {
	q    db.Querier
	pool *pgxpool.Pool
}

// NewFavoriteHandler creates a handler for favorite operations.
func NewFavoriteHandler(q db.Querier, pool *pgxpool.Pool) *FavoriteHandler {
	return &FavoriteHandler{q: q, pool: pool}
}

type toggleFavoriteResponse struct {
	Favorited bool `json:"favorited"`
}

func (h *FavoriteHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	wallpaperID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"wallpaper id is required"}`, http.StatusBadRequest)
		return
	}

	userID, err := parseUUID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	// Use a transaction for the check+insert/delete toggle
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	txQ := db.New(tx)

	exists, err := txQ.CheckFavorite(r.Context(), db.CheckFavoriteParams{
		UserID:      userID,
		WallpaperID: wallpaperID,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	var favorited bool
	if exists {
		if err := txQ.DeleteFavorite(r.Context(), db.DeleteFavoriteParams{
			UserID:      userID,
			WallpaperID: wallpaperID,
		}); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		favorited = false
	} else {
		if err := txQ.InsertFavorite(r.Context(), db.InsertFavoriteParams{
			UserID:      userID,
			WallpaperID: wallpaperID,
		}); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		favorited = true
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toggleFavoriteResponse{Favorited: favorited})
}

type listFavoritesResponse struct {
	Wallpapers []wallpaperResponse `json:"wallpapers"`
	Total      int                 `json:"total"`
}

func (h *FavoriteHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userID, err := parseUUID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	limit, offset := parseLimitOffset(q)

	total, err := h.q.CountFavoritesByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	wallpapers, err := h.q.ListFavoritesByUser(r.Context(), db.ListFavoritesByUserParams{
		UserID: userID,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listFavoritesResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      int(total),
	}
	for i := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, favoriteRowToResponse(&wallpapers[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
