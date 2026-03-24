package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("Auth Middleware", func() {
	var (
		auth     *service.AuthService
		recorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		cfg := config.DefaultConfig()
		auth = service.NewAuthService(cfg)
		recorder = httptest.NewRecorder()
	})

	Describe("Auth", func() {
		var handler http.Handler

		BeforeEach(func() {
			handler = middleware.Auth(auth)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims := middleware.ClaimsFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"user_id": claims.UserID,
					"role":    string(claims.Role),
				})
			}))
		})

		It("returns 401 when Authorization header is missing", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns 401 when Authorization header has bad prefix", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Token xyz")
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns 401 when Bearer token is invalid", func() {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer garbage")
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})

		It("stores claims in context and calls next handler for valid token", func() {
			userID := uuid.New().String()
			token, err := auth.GenerateToken(userID, types.RoleUser)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			handler.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusOK))
			var body map[string]string
			Expect(json.NewDecoder(recorder.Body).Decode(&body)).To(Succeed())
			Expect(body["user_id"]).To(Equal(userID))
			Expect(body["role"]).To(Equal(string(types.RoleUser)))
		})
	})

	Describe("Admin", func() {
		It("returns 403 when no claims in context", func() {
			handler := middleware.Admin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusForbidden))
		})

		It("returns 403 when user is not admin", func() {
			// Chain Auth -> Admin -> next to get proper claims in context
			token, err := auth.GenerateToken(uuid.New().String(), types.RoleUser)
			Expect(err).NotTo(HaveOccurred())

			handler := middleware.Auth(auth)(middleware.Admin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusForbidden))
		})

		It("calls next handler when user is admin", func() {
			token, err := auth.GenerateToken(uuid.New().String(), types.RoleAdmin)
			Expect(err).NotTo(HaveOccurred())

			handler := middleware.Auth(auth)(middleware.Admin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			handler.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("ClaimsFromContext", func() {
		It("returns nil for empty context", func() {
			claims := middleware.ClaimsFromContext(context.Background())
			Expect(claims).To(BeNil())
		})
	})
})
