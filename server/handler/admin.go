package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	db "github.com/0x63616c/screenspace/server/db/generated"
)

type AdminHandler struct {
	q db.Querier
}

// NewAdminHandler creates a handler for admin operations.
func NewAdminHandler(q db.Querier) *AdminHandler {
	return &AdminHandler{q: q}
}

func requireAdmin(r *http.Request) bool {
	claims := claimsFromRequest(r)
	return claims != nil && claims.Role == "admin"
}

func (h *AdminHandler) Queue(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	q := r.URL.Query()
	limit, offset := parseLimitOffset(q)

	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status: "pending_review",
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	wallpapers, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
		Status: "pending_review",
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      int(total),
	}
	for i := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, recentRowToResponse(&wallpapers[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	claims := claimsFromRequest(r)
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	if err := h.q.UpdateWallpaperStatus(r.Context(), db.UpdateWallpaperStatusParams{
		Status: "approved",
		ID:     id,
	}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("wallpaper approved", "admin_id", claims.UserID, "wallpaper_id", id.String(), "action", "approve") //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	var req rejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	reason := &req.Reason
	if err := h.q.UpdateWallpaperStatusWithReason(r.Context(), db.UpdateWallpaperStatusWithReasonParams{
		Status:          "rejected",
		RejectionReason: reason,
		ID:              id,
	}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	claims := claimsFromRequest(r)
	slog.Info("wallpaper rejected", "admin_id", claims.UserID, "wallpaper_id", id.String(), "action", "reject") //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
}

func (h *AdminHandler) ListWallpapers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	q := r.URL.Query()
	limit, offset := parseLimitOffset(q)

	status := q.Get("status")
	if status == "" {
		status = "approved"
	}

	total, err := h.q.CountWallpapers(r.Context(), db.CountWallpapersParams{
		Status: status,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	wallpapers, err := h.q.ListWallpapersRecent(r.Context(), db.ListWallpapersRecentParams{
		Status: status,
		Lim:    int32(limit),
		Off:    int32(offset),
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      int(total),
	}
	for i := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, recentRowToResponse(&wallpapers[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

type editWallpaperRequest struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

func (h *AdminHandler) EditWallpaper(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid wallpaper id"}`, http.StatusBadRequest)
		return
	}

	var req editWallpaperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	var catPtr *string
	if req.Category != "" {
		catPtr = &req.Category
	}

	if err := h.q.UpdateWallpaperMetadata(r.Context(), db.UpdateWallpaperMetadataParams{
		Title:    req.Title,
		Category: catPtr,
		Tags:     tags,
		ID:       id,
	}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

type userResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Banned    bool   `json:"banned"`
	CreatedAt string `json:"created_at"`
}

type listUsersResponse struct {
	Users []userResponse `json:"users"`
	Total int            `json:"total"`
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	q := r.URL.Query()
	limit, offset := parseLimitOffset(q)
	search := q.Get("q")

	var users []db.User
	var total int64

	if search != "" {
		searchPattern := "%" + search + "%"
		var err error
		total, err = h.q.CountUsersWithSearch(r.Context(), searchPattern)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		users, err = h.q.ListUsersWithSearch(r.Context(), db.ListUsersWithSearchParams{
			Query: searchPattern,
			Lim:   int32(limit),
			Off:   int32(offset),
		})
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
	} else {
		var err error
		total, err = h.q.CountUsers(r.Context())
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		users, err = h.q.ListUsers(r.Context(), db.ListUsersParams{
			Lim: int32(limit),
			Off: int32(offset),
		})
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
	}

	resp := listUsersResponse{
		Users: make([]userResponse, 0, len(users)),
		Total: int(total),
	}
	for _, u := range users {
		resp.Users = append(resp.Users, userResponse{
			ID:        u.ID.String(),
			Email:     u.Email,
			Role:      u.Role,
			Banned:    u.Banned,
			CreatedAt: timestamptzToString(u.CreatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{Banned: true, ID: id}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	claims := claimsFromRequest(r)
	slog.Info("user banned", "admin_id", claims.UserID, "target_user_id", id.String(), "action", "ban") //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "banned"})
}

func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.q.SetBanned(r.Context(), db.SetBannedParams{Banned: false, ID: id}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	claims := claimsFromRequest(r)
	slog.Info("user unbanned", "admin_id", claims.UserID, "target_user_id", id.String(), "action", "unban") //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unbanned"})
}

func (h *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.q.GetUserByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.q.SetRole(r.Context(), db.SetRoleParams{Role: "admin", ID: id}); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	claims := claimsFromRequest(r)
	slog.Info("user promoted", "admin_id", claims.UserID, "target_user_id", id.String()) //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "promoted"})
}

type listReportsResponse struct {
	Reports []reportResponse `json:"reports"`
	Total   int              `json:"total"`
}

func (h *AdminHandler) ListReports(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	q := r.URL.Query()
	limit, offset := parseLimitOffset(q)

	total, err := h.q.CountPendingReports(r.Context())
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	reports, err := h.q.ListPendingReports(r.Context(), db.ListPendingReportsParams{
		Lim: int32(limit),
		Off: int32(offset),
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listReportsResponse{
		Reports: make([]reportResponse, 0, len(reports)),
		Total:   int(total),
	}
	for i := range reports {
		resp.Reports = append(resp.Reports, reportToResponse(&reports[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	claims := claimsFromRequest(r)
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, `{"error":"invalid report id"}`, http.StatusBadRequest)
		return
	}

	if err := h.q.DismissReport(r.Context(), id); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	slog.Info("report dismissed", "admin_id", claims.UserID, "report_id", id.String()) //nolint:gosec // claims from JWT

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "dismissed"})
}

func parseLimitOffset(q interface{ Get(string) string }) (int, int) {
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

	return limit, offset
}
