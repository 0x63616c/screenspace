package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgconn"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/respond"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// AuthHandler handles registration, login, and token validation endpoints.
type AuthHandler struct {
	q           db.Querier
	auth        *service.AuthService
	bannedCache *middleware.BannedCache
	cfg         *config.Config
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(q db.Querier, auth *service.AuthService, cache *middleware.BannedCache, cfg *config.Config) *AuthHandler {
	return &AuthHandler{q: q, auth: auth, bannedCache: cache, cfg: cfg}
}

// Register creates a new user and returns a JWT.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest("email and password are required")
	}
	if !emailRe.MatchString(req.Email) {
		return BadRequest("invalid email format")
	}
	if len(req.Password) < h.cfg.MinPasswordLen {
		return BadRequest(fmt.Sprintf("password must be at least %d characters", h.cfg.MinPasswordLen))
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		return Internal(fmt.Errorf("hash password: %w", err))
	}

	role := types.RoleUser
	if req.Email == h.cfg.AdminEmail {
		role = types.RoleAdmin
	}

	user, err := h.q.CreateUser(r.Context(), db.CreateUserParams{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         string(role),
	})
	if err != nil {
		// Use pgx error code 23505 (unique_violation) instead of string matching.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &AppError{Status: http.StatusConflict, Code: "conflict", Message: "registration failed"}
		}
		return Internal(fmt.Errorf("create user: %w", err))
	}

	token, err := h.auth.GenerateToken(user.ID.String(), types.UserRole(user.Role))
	if err != nil {
		return Internal(fmt.Errorf("generate token: %w", err))
	}

	return respond.JSON(w, http.StatusCreated, map[string]string{
		"token": token,
		"role":  user.Role,
	})
}

// Login authenticates a user and returns a JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		return BadRequest("invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return BadRequest("email and password are required")
	}

	user, err := h.q.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid credentials"}
	}

	if user.Banned {
		return &AppError{Status: http.StatusForbidden, Code: "banned", Message: "account banned"}
	}

	if !h.auth.VerifyPassword(user.PasswordHash, req.Password) {
		return &AppError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid credentials"}
	}

	token, err := h.auth.GenerateToken(user.ID.String(), types.UserRole(user.Role))
	if err != nil {
		return Internal(fmt.Errorf("generate token: %w", err))
	}

	return respond.JSON(w, http.StatusOK, map[string]string{
		"token": token,
		"role":  user.Role,
	})
}

// Me returns the current user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) error {
	claims := middleware.ClaimsFromContext(r.Context())
	userID, err := parseUUID(claims.UserID)
	if err != nil {
		return err
	}
	user, err := h.q.GetUserByID(r.Context(), userID)
	if err != nil {
		return NotFound("user not found")
	}
	return respond.JSON(w, http.StatusOK, map[string]string{
		"id":    user.ID.String(),
		"email": user.Email,
		"role":  user.Role,
	})
}
