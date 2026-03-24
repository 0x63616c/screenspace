package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/testutil"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
)

var _ = Describe("WallpaperHandler", func() {
	var (
		mock    *db.MockQuerier
		store   *storage.MockStore
		svc     *service.WallpaperService
		auth    *service.AuthService
		cfg     *config.Config
		h       *handler.WallpaperHandler
		prober  *video.MockProber
	)

	BeforeEach(func() {
		cfg = config.DefaultConfig()
		cfg.BcryptCost = 4
		mock = &db.MockQuerier{}
		store = &storage.MockStore{}
		prober = &video.MockProber{}
		svc = service.NewWallpaperService(mock, store, prober, cfg)
		auth = service.NewAuthService(cfg)
		h = handler.NewWallpaperHandler(mock, store, svc, auth, cfg)
	})

	Describe("Health", func() {
		It("returns ok status", func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			err := h.Health(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("ok"))
			Expect(resp["db"]).To(Equal("ok"))
		})
	})

	Describe("ListCategories", func() {
		It("returns all categories", func() {
			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			w := httptest.NewRecorder()

			err := handler.ListCategories(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var cats []string
			Expect(json.NewDecoder(w.Body).Decode(&cats)).To(Succeed())
			Expect(cats).To(HaveLen(len(types.AllCategories())))
			Expect(cats).To(ContainElement("nature"))
			Expect(cats).To(ContainElement("abstract"))
		})
	})

	Describe("Get", func() {
		It("returns an approved wallpaper", func() {
			wpID := uuid.New()
			uploaderID := uuid.New()
			mock.GetWallpaperByIDFn = func(_ context.Context, id uuid.UUID) (db.GetWallpaperByIDRow, error) {
				return db.GetWallpaperByIDRow{
					ID:         id,
					Title:      "Test Wallpaper",
					UploaderID: uploaderID,
					Status:     string(types.StatusApproved),
					StorageKey: "wallpapers/test/original.mp4",
				}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/wallpapers/"+wpID.String(), nil)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Get(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["title"]).To(Equal("Test Wallpaper"))
		})

		It("returns error for bad UUID", func() {
			req := httptest.NewRequest(http.MethodGet, "/wallpapers/not-a-uuid", nil)
			req = testutil.RequestWithChiParam(req, "id", "not-a-uuid")
			w := httptest.NewRecorder()

			err := h.Get(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("List", func() {
		It("returns paginated wallpapers", func() {
			mock.CountWallpapersFn = func(_ context.Context, _ db.CountWallpapersParams) (int64, error) {
				return 1, nil
			}
			mock.ListWallpapersRecentFn = func(_ context.Context, _ db.ListWallpapersRecentParams) ([]db.ListWallpapersRecentRow, error) {
				return []db.ListWallpapersRecentRow{
					{
						ID:         uuid.New(),
						Title:      "Test",
						UploaderID: uuid.New(),
						Status:     string(types.StatusApproved),
					},
				}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/wallpapers", nil)
			w := httptest.NewRecorder()

			err := h.List(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("items"))
			Expect(resp).To(HaveKey("total"))
		})
	})

	Describe("Create", func() {
		It("creates wallpaper and returns upload URL with 201", func() {
			userID := uuid.New()
			wpID := uuid.New()
			mock.CreateWallpaperFn = func(_ context.Context, _ db.CreateWallpaperParams) (db.CreateWallpaperRow, error) {
				return db.CreateWallpaperRow{ID: wpID, Title: "New Wallpaper"}, nil
			}
			mock.UpdateWallpaperStorageKeyFn = func(_ context.Context, _ db.UpdateWallpaperStorageKeyParams) error {
				return nil
			}

			body := `{"title":"New Wallpaper","category":"nature"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusCreated))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("id"))
			Expect(resp).To(HaveKey("upload_url"))
		})

		It("returns error when title is missing", func() {
			userID := uuid.New()
			body := `{"title":"","category":"nature"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when title is too long", func() {
			userID := uuid.New()
			longTitle := strings.Repeat("a", cfg.MaxTitleLength+1)
			body := `{"title":"` + longTitle + `","category":"nature"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid category", func() {
			userID := uuid.New()
			body := `{"title":"Valid Title","category":"invalid_cat"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download", func() {
		It("returns a presigned download URL", func() {
			wpID := uuid.New()
			mock.GetWallpaperByIDFn = func(_ context.Context, id uuid.UUID) (db.GetWallpaperByIDRow, error) {
				return db.GetWallpaperByIDRow{
					ID:         id,
					Status:     string(types.StatusApproved),
					StorageKey: "wallpapers/test/original.mp4",
				}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/wallpapers/"+wpID.String()+"/download", nil)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Download(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("download_url"))
		})

		It("returns error for bad UUID", func() {
			req := httptest.NewRequest(http.MethodGet, "/wallpapers/bad/download", nil)
			req = testutil.RequestWithChiParam(req, "id", "bad")
			w := httptest.NewRecorder()

			err := h.Download(w, req)
			Expect(err).To(HaveOccurred())
		})
	})
})
