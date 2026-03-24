package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

var _ = Describe("RateLimiter", func() {
	Describe("Allow", func() {
		It("allows requests under the limit", func() {
			rl := middleware.NewRateLimiter(3, time.Minute)
			for i := range 3 {
				Expect(rl.Allow("key")).To(BeTrue(), "request %d should be allowed", i)
			}
		})

		It("blocks requests over the limit", func() {
			rl := middleware.NewRateLimiter(2, time.Minute)
			rl.Allow("key")
			rl.Allow("key")
			Expect(rl.Allow("key")).To(BeFalse())
		})

		It("resets after the window expires", func() {
			rl := middleware.NewRateLimiter(1, 10*time.Millisecond)
			rl.Allow("key")
			Expect(rl.Allow("key")).To(BeFalse())
			time.Sleep(15 * time.Millisecond)
			Expect(rl.Allow("key")).To(BeTrue())
		})
	})

	Describe("PerIP", func() {
		It("blocks requests over the limit", func() {
			rl := middleware.NewRateLimiter(1, time.Minute)
			handler := rl.PerIP()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// First request: allowed
			w1 := httptest.NewRecorder()
			r1 := httptest.NewRequest(http.MethodGet, "/", nil)
			r1.RemoteAddr = "1.2.3.4:9999"
			handler.ServeHTTP(w1, r1)
			Expect(w1.Code).To(Equal(http.StatusOK))

			// Second request: rate limited
			w2 := httptest.NewRecorder()
			r2 := httptest.NewRequest(http.MethodGet, "/", nil)
			r2.RemoteAddr = "1.2.3.4:9999"
			handler.ServeHTTP(w2, r2)
			Expect(w2.Code).To(Equal(http.StatusTooManyRequests))
			Expect(w2.Header().Get("Retry-After")).NotTo(BeEmpty())
		})
	})
})
