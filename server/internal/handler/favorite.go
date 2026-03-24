package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

// FavoriteHandler handles favorite toggle and listing.
type FavoriteHandler struct {
	svc *service.FavoriteService
}

// NewFavoriteHandler creates a new FavoriteHandler.
func NewFavoriteHandler(svc *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{svc: svc}
}

// Toggle adds or removes a favorite.
func (h *FavoriteHandler) Toggle(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return BadRequest("invalid user id")
	}
	wallpaperID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	favorited, err := h.svc.Toggle(r.Context(), userID, wallpaperID)
	if err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]bool{"favorited": favorited})
}
