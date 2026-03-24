package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/handler"
)

var _ = Describe("Wrap", func() {
	It("runs the handler normally when no error is returned", func() {
		h := handler.Wrap(func(w http.ResponseWriter, _ *http.Request) error {
			w.WriteHeader(http.StatusOK)
			return nil
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("maps an AppError to the correct HTTP status", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.NotFound("wallpaper not found")
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("maps ErrForbidden sentinel to 403", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.ErrForbidden
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusForbidden))
	})

	It("maps ErrNotFound sentinel to 404", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.ErrNotFound
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("maps ErrConflict sentinel to 409", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.ErrConflict
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusConflict))
	})

	It("maps ErrBadRequest sentinel to 400", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return handler.ErrBadRequest
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})

	It("maps an unknown error to 500", func() {
		h := handler.Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
			return errors.New("something exploded")
		})
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest(http.MethodGet, "/", nil))
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})
})
