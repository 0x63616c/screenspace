package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/testutil"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("FavoriteHandler", func() {
	var (
		mock *db.MockQuerier
		svc  *service.FavoriteService
		h    *handler.FavoriteHandler
		cfg  *config.Config
	)

	BeforeEach(func() {
		cfg = config.DefaultConfig()
		mock = &db.MockQuerier{}
		svc = service.NewFavoriteService(mock)
		h = handler.NewFavoriteHandler(svc, cfg)
	})

	Describe("Toggle", func() {
		It("toggles a favorite successfully", func() {
			userID := uuid.New()
			wpID := uuid.New()

			mock.CheckFavoriteFn = func(_ context.Context, _ db.CheckFavoriteParams) (bool, error) {
				return false, nil
			}
			mock.InsertFavoriteFn = func(_ context.Context, _ db.InsertFavoriteParams) error {
				return nil
			}

			req := httptest.NewRequest(http.MethodPost, "/wallpapers/"+wpID.String()+"/favorite", nil)
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Toggle(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]bool
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["favorited"]).To(BeTrue())
		})

		It("returns error for bad wallpaper UUID", func() {
			userID := uuid.New()

			req := httptest.NewRequest(http.MethodPost, "/wallpapers/bad/favorite", nil)
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			req = testutil.RequestWithChiParam(req, "id", "bad-uuid")
			w := httptest.NewRecorder()

			err := h.Toggle(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("List", func() {
		It("returns paginated favorites", func() {
			userID := uuid.New()

			mock.CountFavoritesByUserFn = func(_ context.Context, _ uuid.UUID) (int64, error) {
				return 1, nil
			}
			mock.ListFavoritesByUserFn = func(_ context.Context, _ db.ListFavoritesByUserParams) ([]db.ListFavoritesByUserRow, error) {
				return []db.ListFavoritesByUserRow{
					{
						ID:         uuid.New(),
						Title:      "Fav Wallpaper",
						UploaderID: uuid.New(),
						Status:     string(types.StatusApproved),
					},
				}, nil
			}

			req := httptest.NewRequest(http.MethodGet, "/favorites", nil)
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			w := httptest.NewRecorder()

			err := h.List(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp).To(HaveKey("items"))
			Expect(resp["total"]).To(BeNumerically("==", 1))
		})
	})
})
