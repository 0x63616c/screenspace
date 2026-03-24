package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/testutil"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("AuthHandler", func() {
	var (
		mock    *db.MockQuerier
		auth    *service.AuthService
		h       *handler.AuthHandler
		cfg     *config.Config
		cache   *middleware.BannedCache
	)

	BeforeEach(func() {
		cfg = config.DefaultConfig()
		cfg.BcryptCost = 4
		cfg.AdminEmail = "admin@example.com"
		mock = &db.MockQuerier{}
		auth = service.NewAuthService(cfg)
		cache = middleware.NewBannedCache()
		h = handler.NewAuthHandler(mock, auth, cache, cfg)
	})

	Describe("Register", func() {
		It("creates user and returns token with 201", func() {
			userID := uuid.New()
			mock.CreateUserFn = func(_ context.Context, arg db.CreateUserParams) (db.User, error) {
				return db.User{ID: userID, Email: arg.Email, Role: arg.Role}, nil
			}

			body := `{"email":"test@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusCreated))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("token"))
			Expect(resp["role"]).To(Equal("user"))
		})

		It("returns 400 when email or password is missing", func() {
			body := `{"email":"","password":""}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).To(HaveOccurred())
			var appErr *handler.AppError
			Expect(err).To(BeAssignableToTypeOf(appErr))
		})

		It("returns 400 for bad email format", func() {
			body := `{"email":"notanemail","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).To(HaveOccurred())
		})

		It("returns 400 for short password", func() {
			body := `{"email":"test@example.com","password":"short"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).To(HaveOccurred())
		})

		It("returns 409 for duplicate email (PG 23505)", func() {
			mock.CreateUserFn = func(_ context.Context, _ db.CreateUserParams) (db.User, error) {
				return db.User{}, &pgconn.PgError{Code: "23505"}
			}

			body := `{"email":"dup@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).To(HaveOccurred())
			var appErr *handler.AppError
			Expect(err).To(BeAssignableToTypeOf(appErr))
			//nolint:errorlint // test assertion, not production error handling
			Expect(err.(*handler.AppError).Status).To(Equal(http.StatusConflict))
		})

		It("assigns admin role when email matches admin email", func() {
			userID := uuid.New()
			var capturedRole string
			mock.CreateUserFn = func(_ context.Context, arg db.CreateUserParams) (db.User, error) {
				capturedRole = arg.Role
				return db.User{ID: userID, Email: arg.Email, Role: arg.Role}, nil
			}

			body := `{"email":"admin@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Register(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRole).To(Equal(string(types.RoleAdmin)))
		})
	})

	Describe("Login", func() {
		var hashedPassword string

		BeforeEach(func() {
			var err error
			hashedPassword, err = auth.HashPassword("secret123")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns token on successful login", func() {
			userID := uuid.New()
			mock.GetUserByEmailFn = func(_ context.Context, _ string) (db.User, error) {
				return db.User{ID: userID, Email: "test@example.com", PasswordHash: hashedPassword, Role: "user"}, nil
			}

			body := `{"email":"test@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Login(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("token"))
		})

		It("returns 400 when fields are missing", func() {
			body := `{"email":"","password":""}`
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Login(w, req)
			Expect(err).To(HaveOccurred())
		})

		It("returns 401 when user not found", func() {
			mock.GetUserByEmailFn = func(_ context.Context, _ string) (db.User, error) {
				return db.User{}, context.DeadlineExceeded // any non-nil error
			}

			body := `{"email":"nobody@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Login(w, req)
			Expect(err).To(HaveOccurred())
			//nolint:errorlint // test assertion
			Expect(err.(*handler.AppError).Status).To(Equal(http.StatusUnauthorized))
		})

		It("returns 403 when user is banned", func() {
			mock.GetUserByEmailFn = func(_ context.Context, _ string) (db.User, error) {
				return db.User{Banned: true, PasswordHash: hashedPassword}, nil
			}

			body := `{"email":"banned@example.com","password":"secret123"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Login(w, req)
			Expect(err).To(HaveOccurred())
			//nolint:errorlint // test assertion
			Expect(err.(*handler.AppError).Status).To(Equal(http.StatusForbidden))
		})

		It("returns 401 when password is wrong", func() {
			mock.GetUserByEmailFn = func(_ context.Context, _ string) (db.User, error) {
				return db.User{PasswordHash: hashedPassword}, nil
			}

			body := `{"email":"test@example.com","password":"wrongpassword"}`
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
			w := httptest.NewRecorder()

			err := h.Login(w, req)
			Expect(err).To(HaveOccurred())
			//nolint:errorlint // test assertion
			Expect(err.(*handler.AppError).Status).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("Me", func() {
		It("returns user data on success", func() {
			userID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Email: "me@example.com", Role: "user"}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.Me(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["email"]).To(Equal("me@example.com"))
			Expect(resp["id"]).To(Equal(userID.String()))
		})
	})
})
