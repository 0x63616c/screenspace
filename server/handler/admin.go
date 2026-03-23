package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/0x63616c/screenspace/server/repository"
)

type AdminHandler struct {
	wallpapers *repository.WallpaperRepo
	users      *repository.UserRepo
	reports    *repository.ReportRepo
}

func NewAdminHandler(
	wallpapers *repository.WallpaperRepo,
	users *repository.UserRepo,
	reports *repository.ReportRepo,
) *AdminHandler {
	return &AdminHandler{
		wallpapers: wallpapers,
		users:      users,
		reports:    reports,
	}
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

	wallpapers, total, err := h.wallpapers.List(r.Context(), repository.ListParams{
		Status: "pending_review",
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      total,
	}
	for _, wp := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, wallpaperToResponse(wp))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) Approve(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := h.wallpapers.UpdateStatus(r.Context(), id, "approved"); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := r.PathValue("id")

	var req rejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.wallpapers.UpdateStatus(r.Context(), id, "rejected", req.Reason); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected"})
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

	wallpapers, total, err := h.wallpapers.List(r.Context(), repository.ListParams{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listWallpapersResponse{
		Wallpapers: make([]wallpaperResponse, 0, len(wallpapers)),
		Total:      total,
	}
	for _, wp := range wallpapers {
		resp.Wallpapers = append(resp.Wallpapers, wallpaperToResponse(wp))
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

	id := r.PathValue("id")

	var req editWallpaperRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	if err := h.wallpapers.UpdateMetadata(r.Context(), id, req.Title, req.Category, tags); err != nil {
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

	users, total, err := h.users.ListWithSearch(r.Context(), search, limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listUsersResponse{
		Users: make([]userResponse, 0, len(users)),
		Total: total,
	}
	for _, u := range users {
		resp.Users = append(resp.Users, userResponse{
			ID:        u.ID,
			Email:     u.Email,
			Role:      u.Role,
			Banned:    u.Banned,
			CreatedAt: u.CreatedAt.Format(time.RFC3339),
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

	id := r.PathValue("id")
	if _, err := h.users.GetByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.users.SetBanned(r.Context(), id, true); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "banned"})
}

func (h *AdminHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if _, err := h.users.GetByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.users.SetBanned(r.Context(), id, false); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "unbanned"})
}

func (h *AdminHandler) PromoteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if _, err := h.users.GetByID(r.Context(), id); err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if err := h.users.SetRole(r.Context(), id, "admin"); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

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

	reports, total, err := h.reports.ListPending(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp := listReportsResponse{
		Reports: make([]reportResponse, 0, len(reports)),
		Total:   total,
	}
	for _, rpt := range reports {
		resp.Reports = append(resp.Reports, reportToResponse(rpt))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminHandler) DismissReport(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(r) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	id := r.PathValue("id")
	if err := h.reports.Dismiss(r.Context(), id); err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

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
