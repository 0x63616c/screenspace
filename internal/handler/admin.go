package handler

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/types"
)

// AdminHandler handles admin-only HTTP endpoints.
type AdminHandler struct {
	q            db.Querier
	store        storage.Store
	wallpaperSvc *service.WallpaperService
	bannedCache  *middleware.BannedCache
	cfg          *config.Config
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(q db.Querier, s storage.Store, svc *service.WallpaperService, cache *middleware.BannedCache, cfg *config.Config) *AdminHandler {
	return &AdminHandler{q: q, store: s, wallpaperSvc: svc, bannedCache: cache, cfg: cfg}
}

// Queue returns wallpapers in pending_review status.
func (h *AdminHandler) Queue(w http.ResponseWriter, r *http.Request) error {
	pg := respond.ParsePagination(r.URL.Query(), h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	// Use recent listing with pending_review status.
	wallpapers, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
		Status: string(types.StatusPendingReview),
		Off:    int32(pg.Offset),
		Lim:    int32(pg.Limit),
	})
	if err != nil {
		return Internal(fmt.Errorf("list queue: %w", err))
	}
	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status: string(types.StatusPendingReview),
	})
	if err != nil {
		return Internal(fmt.Errorf("count queue: %w", err))
	}
	items := make([]map[string]any, len(wallpapers))
	for i := range wallpapers {
		items[i] = recentRowToMap(&wallpapers[i], nil, r)
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

// Approve approves a wallpaper in the review queue.
func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	adminID, err := parseUUID(claims.UserID)
	if err != nil {
		return err
	}
	if err := h.wallpaperSvc.Approve(r.Context(), id, adminID); err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// Reject rejects a wallpaper in the review queue.
func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	adminID, err := parseUUID(claims.UserID)
	if err != nil {
		return err
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if err := h.wallpaperSvc.Reject(r.Context(), id, adminID, req.Reason); err != nil {
		return err
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// ListWallpapers lists wallpapers with optional status filter.
func (h *AdminHandler) ListWallpapers(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	status := q.Get("status")
	if status == "" {
		status = string(types.StatusApproved)
	}
	wallpapers, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
		Status: status,
		Off:    int32(pg.Offset),
		Lim:    int32(pg.Limit),
	})
	if err != nil {
		return Internal(fmt.Errorf("list wallpapers: %w", err))
	}
	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status: status,
	})
	if err != nil {
		return Internal(fmt.Errorf("count wallpapers: %w", err))
	}
	items := make([]map[string]any, len(wallpapers))
	for i := range wallpapers {
		items[i] = recentRowToMap(&wallpapers[i], nil, r)
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

// EditWallpaper updates wallpaper metadata.
func (h *AdminHandler) EditWallpaper(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid wallpaper id")
	}
	var req struct {
		Title    string   `json:"title"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}
	cat := ptrOrNil(req.Category)
	if err := h.q.UpdateWallpaperMetadata(r.Context(), db.UpdateWallpaperMetadataParams{
		ID:       id,
		Title:    req.Title,
		Category: cat,
		Tags:     tags,
	}); err != nil {
		return Internal(fmt.Errorf("update metadata: %w", err))
	}
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ListUsers returns paginated user list with optional search.
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	pg := respond.ParsePagination(q, h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	search := q.Get("q")

	var users []db.User
	var total int64
	var err error

	if search != "" {
		searchPattern := "%" + search + "%"
		users, err = h.q.ListUsersWithSearch(r.Context(), db.ListUsersWithSearchParams{
			Query: searchPattern,
			Off:   int32(pg.Offset),
			Lim:   int32(pg.Limit),
		})
		if err != nil {
			return Internal(fmt.Errorf("list users: %w", err))
		}
		total, err = h.q.CountUsersWithSearch(r.Context(), searchPattern)
	} else {
		users, err = h.q.ListUsers(r.Context(), db.ListUsersParams{
			Off: int32(pg.Offset),
			Lim: int32(pg.Limit),
		})
		if err != nil {
			return Internal(fmt.Errorf("list users: %w", err))
		}
		total, err = h.q.CountUsers(r.Context())
	}
	if err != nil {
		return Internal(fmt.Errorf("count users: %w", err))
	}

	items := make([]map[string]any, len(users))
	for i, u := range users {
		items[i] = map[string]any{
			"id":         u.ID.String(),
			"email":      u.Email,
			"role":       u.Role,
			"banned":     u.Banned,
			"created_at": formatTimestamp(u.CreatedAt),
		}
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

// BanUser bans a user.
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid user id")
	}
	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		return NotFound("user not found")
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{ID: id, Banned: true}); err != nil {
		return Internal(fmt.Errorf("ban user: %w", err))
	}
	h.bannedCache.Evict(id.String())
	slog.Info("user banned", "admin_id", claims.UserID, "target_user_id", id) //nolint:gosec // structured log
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "banned"})
}

// UnbanUser unbans a user.
func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid user id")
	}
	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		return NotFound("user not found")
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{ID: id, Banned: false}); err != nil {
		return Internal(fmt.Errorf("unban user: %w", err))
	}
	h.bannedCache.Evict(id.String())
	slog.Info("user unbanned", "admin_id", claims.UserID, "target_user_id", id) //nolint:gosec // structured log
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "unbanned"})
}

// PromoteUser promotes a user to admin.
func (h *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid user id")
	}
	user, err := h.q.GetUserByID(r.Context(), id)
	if err != nil {
		return NotFound("user not found")
	}
	if user.Banned {
		return BadRequest("cannot promote a banned user")
	}
	if types.UserRole(user.Role) == types.RoleAdmin {
		return respond.JSON(w, http.StatusOK, map[string]string{"status": "promoted"})
	}
	if err := h.q.SetRole(r.Context(), db.SetRoleParams{ID: id, Role: string(types.RoleAdmin)}); err != nil {
		return Internal(fmt.Errorf("promote user: %w", err))
	}
	slog.Info("user promoted", "admin_id", claims.UserID, "target_user_id", id) //nolint:gosec // structured log
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "promoted"})
}

// ListReports returns pending reports.
func (h *AdminHandler) ListReports(w http.ResponseWriter, r *http.Request) error {
	pg := respond.ParsePagination(r.URL.Query(), h.cfg.DefaultPageSize, h.cfg.MaxPageSize)
	reports, err := h.q.ListPendingReports(r.Context(), db.ListPendingReportsParams{
		Off: int32(pg.Offset),
		Lim: int32(pg.Limit),
	})
	if err != nil {
		return Internal(fmt.Errorf("list reports: %w", err))
	}
	total, err := h.q.CountPendingReports(r.Context())
	if err != nil {
		return Internal(fmt.Errorf("count reports: %w", err))
	}
	items := make([]map[string]any, len(reports))
	for i, rpt := range reports {
		items[i] = map[string]any{
			"id":           rpt.ID.String(),
			"wallpaper_id": rpt.WallpaperID.String(),
			"reporter_id":  rpt.ReporterID.String(),
			"reason":       rpt.Reason,
			"status":       rpt.Status,
			"created_at":   formatTimestamp(rpt.CreatedAt),
		}
	}
	return respond.Paginated(w, items, int(total), pg.Limit, pg.Offset)
}

// DismissReport dismisses a report.
func (h *AdminHandler) DismissReport(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return BadRequest("invalid report id")
	}
	if err := h.q.DismissReport(r.Context(), id); err != nil {
		return Internal(fmt.Errorf("dismiss report: %w", err))
	}
	slog.Info("report dismissed", "admin_id", claims.UserID, "report_id", id) //nolint:gosec // structured log
	return respond.JSON(w, http.StatusOK, map[string]string{"status": "dismissed"})
}
