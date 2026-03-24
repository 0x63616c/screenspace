package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
)

// FavoriteHandler handles favorite toggle and listing.
type FavoriteHandler struct {
	svc *service.FavoriteService
	cfg *config.Config
}

// NewFavoriteHandler creates a new FavoriteHandler.
func NewFavoriteHandler(svc *service.FavoriteService, cfg *config.Config) *FavoriteHandler {
	return &FavoriteHandler{svc: svc, cfg: cfg}
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

// List returns paginated favorites for the authenticated user.
func (h *FavoriteHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return BadRequest("invalid user id")
	}

	pg := respond.ParsePagination(r.URL.Query(), h.cfg.DefaultPageSize, h.cfg.MaxPageSize)

	items, total, err := h.svc.ListByUser(r.Context(), userID, pg.Limit, pg.Offset)
	if err != nil {
		return Internal(fmt.Errorf("list favorites: %w", err))
	}

	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}
