package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/repository"
	"github.com/0x63616c/screenspace/server/service"
)

type AuthHandler struct {
	users *repository.UserRepo
	auth  *service.AuthService
	admin string
}

func NewAuthHandler(users *repository.UserRepo, auth *service.AuthService, adminEmail string) *AuthHandler {
	return &AuthHandler{users: users, auth: auth, admin: adminEmail}
}

func claimsFromRequest(r *http.Request) *service.TokenClaims {
	return middleware.ClaimsFromContext(r.Context())
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

type meResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password are required"}`, http.StatusBadRequest)
		return
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	role := "user"
	if req.Email == h.admin {
		role = "admin"
	}

	user, err := h.users.Create(r.Context(), req.Email, hash, role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	token, err := h.auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(authResponse{Token: token, Role: user.Role})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password are required"}`, http.StatusBadRequest)
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if user.Banned {
		http.Error(w, `{"error":"account banned"}`, http.StatusForbidden)
		return
	}

	if !h.auth.VerifyPassword(user.PasswordHash, req.Password) {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := h.auth.GenerateToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{Token: token, Role: user.Role})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromRequest(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	user, err := h.users.GetByID(r.Context(), claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(meResponse{ID: user.ID, Email: user.Email, Role: user.Role})
}
