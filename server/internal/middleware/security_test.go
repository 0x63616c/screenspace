package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/middleware"
)

var _ = Describe("SecurityHeaders", func() {
	It("sets X-Content-Type-Options, X-Frame-Options, and CSP", func() {
		handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
		Expect(recorder.Header().Get("X-Content-Type-Options")).To(Equal("nosniff"))
		Expect(recorder.Header().Get("X-Frame-Options")).To(Equal("DENY"))
		Expect(recorder.Header().Get("Content-Security-Policy")).To(Equal("default-src 'none'"))
	})
})

var _ = Describe("MaxBodySize", func() {
	It("passes through when body is under 1MB", func() {
		handler := middleware.MaxBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(HaveLen(100))
			w.WriteHeader(http.StatusOK)
		}))

		smallBody := strings.NewReader(strings.Repeat("a", 100))
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", smallBody)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusOK))
	})

	It("returns an error when reading body over 1MB", func() {
		handler := middleware.MaxBodySize(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			Expect(err).To(HaveOccurred())
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}))

		// 1MB + 1 byte
		bigBody := strings.NewReader(strings.Repeat("a", (1<<20)+1))
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", bigBody)
		handler.ServeHTTP(recorder, req)

		Expect(recorder.Code).To(Equal(http.StatusRequestEntityTooLarge))
	})
})
