package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/middleware"
	"github.com/0x63616c/screenspace/server/service"
)

type AuthHandler struct {
	q     db.Querier
	auth  *service.AuthService
	admin string
}

// NewAuthHandler creates a handler for authentication operations.
func NewAuthHandler(q db.Querier, auth *service.AuthService, adminEmail string) *AuthHandler {
	return &AuthHandler{q: q, auth: auth, admin: adminEmail}
}

func claimsFromRequest(r *http.Request) *service.TokenClaims {
	return middleware.ClaimsFromContext(r.Context())
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
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

	user, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	token, err := h.auth.GenerateToken(user.ID.String(), user.Role)
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

	user, err := h.q.GetUserByEmail(r.Context(), req.Email)
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

	token, err := h.auth.GenerateToken(user.ID.String(), user.Role)
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

	uid, err := parseUUID(claims.UserID)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	user, err := h.q.GetUserByID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(meResponse{ID: user.ID.String(), Email: user.Email, Role: user.Role})
}
