package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/0x63616c/screenspace/server/repository"
)

type FavoriteHandler struct {
	favorites *repository.FavoriteRepo
}

func NewFavoriteHandler(favorites *repository.FavoriteRepo) *FavoriteHandler {
	return &FavoriteHandler{favorites: favorites}
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

	wallpaperID := r.PathValue("id")
	if wallpaperID == "" {
		http.Error(w, `{"error":"wallpaper id is required"}`, http.StatusBadRequest)
		return
	}

	favorited, err := h.favorites.Toggle(r.Context(), claims.UserID, wallpaperID)
	if err != nil {
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

	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	wallpapers, total, err := h.favorites.ListByUser(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listFavoritesResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      total,
	}
	for _, wp := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, wallpaperToResponse(wp))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
