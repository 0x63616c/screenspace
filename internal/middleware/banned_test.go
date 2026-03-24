package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("BannedCache", func() {
	var cache *middleware.BannedCache

	BeforeEach(func() {
		cache = middleware.NewBannedCache()
	})

	It("stores and retrieves a banned status", func() {
		cache.Set("u1", true)
		banned, ok := cache.IsBanned("u1")
		Expect(ok).To(BeTrue())
		Expect(banned).To(BeTrue())
	})

	It("removes an entry on evict", func() {
		cache.Set("u1", true)
		cache.Evict("u1")
		_, ok := cache.IsBanned("u1")
		Expect(ok).To(BeFalse())
	})

	It("returns a miss for unknown user", func() {
		_, ok := cache.IsBanned("nobody")
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("BannedCheck Middleware", func() {
	var (
		auth       *service.AuthService
		cache      *middleware.BannedCache
		mock       *db.MockQuerier
		recorder   *httptest.ResponseRecorder
		nextCalled bool
	)

	BeforeEach(func() {
		cfg := config.DefaultConfig()
		auth = service.NewAuthService(cfg)
		cache = middleware.NewBannedCache()
		mock = &db.MockQuerier{}
		nextCalled = false
		recorder = httptest.NewRecorder()
	})

	// buildHandler chains Auth -> BannedCheck -> next so claims are properly set
	// in context using the middleware's own context key.
	buildHandler := func() http.Handler {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		return middleware.Auth(auth)(middleware.BannedCheck(mock, cache)(next))
	}

	// authedRequest creates a request with a valid Bearer token for the given userID.
	authedRequest := func(userID string, role types.UserRole) *http.Request {
		token, err := auth.GenerateToken(userID, role)
		Expect(err).NotTo(HaveOccurred())
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	It("returns 403 when cache says banned", func() {
		userID := uuid.New().String()
		cache.Set(userID, true)

		handler := buildHandler()
		handler.ServeHTTP(recorder, authedRequest(userID, types.RoleUser))

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())
	})

	It("passes through when cache says not banned", func() {
		userID := uuid.New().String()
		cache.Set(userID, false)

		handler := buildHandler()
		handler.ServeHTTP(recorder, authedRequest(userID, types.RoleUser))

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(nextCalled).To(BeTrue())
	})

	It("returns 403 and caches true on cache miss when DB says banned", func() {
		userID := uuid.New()
		mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
			Expect(id).To(Equal(userID))
			return db.User{ID: userID, Banned: true}, nil
		}

		handler := buildHandler()
		handler.ServeHTTP(recorder, authedRequest(userID.String(), types.RoleUser))

		Expect(recorder.Code).To(Equal(http.StatusForbidden))
		Expect(nextCalled).To(BeFalse())

		banned, ok := cache.IsBanned(userID.String())
		Expect(ok).To(BeTrue())
		Expect(banned).To(BeTrue())
	})

	It("passes through and caches false on cache miss when DB says not banned", func() {
		userID := uuid.New()
		mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
			Expect(id).To(Equal(userID))
			return db.User{ID: userID, Banned: false}, nil
		}

		handler := buildHandler()
		handler.ServeHTTP(recorder, authedRequest(userID.String(), types.RoleUser))

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(nextCalled).To(BeTrue())

		banned, ok := cache.IsBanned(userID.String())
		Expect(ok).To(BeTrue())
		Expect(banned).To(BeFalse())
	})

	It("passes through when no claims in context (unauthenticated)", func() {
		// No Auth middleware, just BannedCheck -> next
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		handler := middleware.BannedCheck(mock, cache)(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(nextCalled).To(BeTrue())
	})

	It("returns 401 when claims contain invalid UUID", func() {
		// Use a token with a non-UUID userID. The Auth middleware will accept it
		// (it only validates the JWT signature), but BannedCheck will fail on uuid.Parse.
		token, err := auth.GenerateToken("not-a-uuid", types.RoleUser)
		Expect(err).NotTo(HaveOccurred())

		handler := buildHandler()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})
})
