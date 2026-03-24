package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/handler"
	"github.com/0x63616c/screenspace/server/internal/middleware"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/storage"
	"github.com/0x63616c/screenspace/server/internal/testutil"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
)

var _ = Describe("AdminHandler", func() {
	var (
		mock    *db.MockQuerier
		store   *storage.MockStore
		wpSvc   *service.WallpaperService
		cache   *middleware.BannedCache
		cfg     *config.Config
		h       *handler.AdminHandler
		adminID uuid.UUID
	)

	BeforeEach(func() {
		cfg = config.DefaultConfig()
		mock = &db.MockQuerier{}
		store = &storage.MockStore{}
		prober := &video.MockProber{}
		wpSvc = service.NewWallpaperService(mock, store, prober, cfg)
		cache = middleware.NewBannedCache()
		h = handler.NewAdminHandler(mock, store, wpSvc, cache, cfg)
		adminID = uuid.New()
	})

	adminReq := func(method, path string, body string) *http.Request {
		var req *http.Request
		if body != "" {
			req = httptest.NewRequest(method, path, strings.NewReader(body))
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		return testutil.RequestWithClaims(req, adminID.String(), types.RoleAdmin)
	}

	Describe("Queue", func() {
		It("returns pending_review wallpapers", func() {
			mock.ListWallpapersRecentFn = func(_ context.Context, arg db.ListWallpapersRecentParams) ([]db.ListWallpapersRecentRow, error) {
				Expect(arg.Status).To(Equal(string(types.StatusPendingReview)))
				return []db.ListWallpapersRecentRow{
					{ID: uuid.New(), Title: "Pending", UploaderID: uuid.New(), Status: string(types.StatusPendingReview)},
				}, nil
			}
			mock.CountWallpapersFn = func(_ context.Context, _ db.CountWallpapersParams) (int64, error) {
				return 1, nil
			}

			req := adminReq(http.MethodGet, "/admin/queue", "")
			w := httptest.NewRecorder()

			err := h.Queue(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["total"]).To(BeNumerically("==", 1))
		})
	})

	Describe("Approve", func() {
		It("approves a wallpaper", func() {
			wpID := uuid.New()
			mock.GetWallpaperByIDFn = func(_ context.Context, _ uuid.UUID) (db.GetWallpaperByIDRow, error) {
				return db.GetWallpaperByIDRow{ID: wpID, Status: string(types.StatusPendingReview)}, nil
			}
			mock.UpdateWallpaperStatusFn = func(_ context.Context, _ db.UpdateWallpaperStatusParams) error {
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/wallpapers/"+wpID.String()+"/approve", "")
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Approve(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("approved"))
		})

		It("returns error for bad UUID", func() {
			req := adminReq(http.MethodPost, "/admin/wallpapers/bad/approve", "")
			req = testutil.RequestWithChiParam(req, "id", "bad-uuid")
			w := httptest.NewRecorder()

			err := h.Approve(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Reject", func() {
		It("rejects a wallpaper", func() {
			wpID := uuid.New()
			mock.GetWallpaperByIDFn = func(_ context.Context, _ uuid.UUID) (db.GetWallpaperByIDRow, error) {
				return db.GetWallpaperByIDRow{ID: wpID, Status: string(types.StatusPendingReview)}, nil
			}
			mock.UpdateWallpaperStatusReasonFn = func(_ context.Context, _ db.UpdateWallpaperStatusWithReasonParams) error {
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/wallpapers/"+wpID.String()+"/reject", `{"reason":"low quality"}`)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Reject(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("rejected"))
		})
	})

	Describe("ListWallpapers", func() {
		It("returns paginated wallpapers", func() {
			mock.ListWallpapersRecentFn = func(_ context.Context, _ db.ListWallpapersRecentParams) ([]db.ListWallpapersRecentRow, error) {
				return []db.ListWallpapersRecentRow{
					{ID: uuid.New(), Title: "WP1", UploaderID: uuid.New(), Status: string(types.StatusApproved)},
				}, nil
			}
			mock.CountWallpapersFn = func(_ context.Context, _ db.CountWallpapersParams) (int64, error) {
				return 1, nil
			}

			req := adminReq(http.MethodGet, "/admin/wallpapers", "")
			w := httptest.NewRecorder()

			err := h.ListWallpapers(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("EditWallpaper", func() {
		It("updates wallpaper metadata", func() {
			wpID := uuid.New()
			mock.UpdateWallpaperMetadataFn = func(_ context.Context, arg db.UpdateWallpaperMetadataParams) error {
				Expect(arg.Title).To(Equal("New Title"))
				return nil
			}

			body := `{"title":"New Title","category":"nature","tags":["scenic"]}`
			req := adminReq(http.MethodPut, "/admin/wallpapers/"+wpID.String(), body)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.EditWallpaper(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("updated"))
		})

		It("returns error for bad UUID", func() {
			body := `{"title":"T","category":"nature","tags":[]}`
			req := adminReq(http.MethodPut, "/admin/wallpapers/bad", body)
			req = testutil.RequestWithChiParam(req, "id", "bad-uuid")
			w := httptest.NewRecorder()

			err := h.EditWallpaper(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ListUsers", func() {
		It("returns paginated users", func() {
			userID := uuid.New()
			mock.ListUsersFn = func(_ context.Context, _ db.ListUsersParams) ([]db.User, error) {
				return []db.User{
					{ID: userID, Email: "user@example.com", Role: "user"},
				}, nil
			}
			mock.CountUsersFn = func(_ context.Context) (int64, error) {
				return 1, nil
			}

			req := adminReq(http.MethodGet, "/admin/users", "")
			w := httptest.NewRecorder()

			err := h.ListUsers(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["total"]).To(BeNumerically("==", 1))
		})

		It("uses search when q param is present", func() {
			mock.ListUsersWithSearchFn = func(_ context.Context, arg db.ListUsersWithSearchParams) ([]db.User, error) {
				Expect(arg.Query).To(Equal("%test%"))
				return []db.User{}, nil
			}
			mock.CountUsersWithSearchFn = func(_ context.Context, query string) (int64, error) {
				Expect(query).To(Equal("%test%"))
				return 0, nil
			}

			req := adminReq(http.MethodGet, "/admin/users?q=test", "")
			w := httptest.NewRecorder()

			err := h.ListUsers(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("BanUser", func() {
		It("bans a user and evicts cache", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Email: "target@example.com", Role: "user"}, nil
			}
			mock.SetBannedFn = func(_ context.Context, _ db.SetBannedParams) error {
				return nil
			}

			// Pre-populate cache to verify eviction.
			cache.Set(targetID.String(), false)

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/ban", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.BanUser(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			// Cache should be evicted.
			_, ok := cache.IsBanned(targetID.String())
			Expect(ok).To(BeFalse())
		})

		It("returns 404 when user not found", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, _ uuid.UUID) (db.User, error) {
				return db.User{}, fmt.Errorf("not found")
			}

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/ban", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.BanUser(w, req)
			Expect(err).To(HaveOccurred())
			//nolint:errorlint // test assertion
			Expect(err.(*handler.AppError).Status).To(Equal(http.StatusNotFound))
		})
	})

	Describe("UnbanUser", func() {
		It("unbans a user", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Banned: true}, nil
			}
			mock.SetBannedFn = func(_ context.Context, _ db.SetBannedParams) error {
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/unban", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.UnbanUser(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("unbanned"))
		})
	})

	Describe("PromoteUser", func() {
		It("promotes a user to admin", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Role: "user"}, nil
			}
			mock.SetRoleFn = func(_ context.Context, arg db.SetRoleParams) error {
				Expect(arg.Role).To(Equal(string(types.RoleAdmin)))
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/promote", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.PromoteUser(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("promoted"))
		})

		It("returns ok without calling SetRole when user is already admin", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Role: string(types.RoleAdmin)}, nil
			}
			setRoleCalled := false
			mock.SetRoleFn = func(_ context.Context, _ db.SetRoleParams) error {
				setRoleCalled = true
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/promote", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.PromoteUser(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(setRoleCalled).To(BeFalse())
		})

		It("returns 400 when user is banned", func() {
			targetID := uuid.New()
			mock.GetUserByIDFn = func(_ context.Context, id uuid.UUID) (db.User, error) {
				return db.User{ID: id, Role: "user", Banned: true}, nil
			}

			req := adminReq(http.MethodPost, "/admin/users/"+targetID.String()+"/promote", "")
			req = testutil.RequestWithChiParam(req, "id", targetID.String())
			w := httptest.NewRecorder()

			err := h.PromoteUser(w, req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ListReports", func() {
		It("returns pending reports", func() {
			reportID := uuid.New()
			mock.ListPendingReportsFn = func(_ context.Context, _ db.ListPendingReportsParams) ([]db.Report, error) {
				return []db.Report{
					{ID: reportID, WallpaperID: uuid.New(), ReporterID: uuid.New(), Reason: "spam", Status: "pending"},
				}, nil
			}
			mock.CountPendingReportsFn = func(_ context.Context) (int64, error) {
				return 1, nil
			}

			req := adminReq(http.MethodGet, "/admin/reports", "")
			w := httptest.NewRecorder()

			err := h.ListReports(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["total"]).To(BeNumerically("==", 1))
		})
	})

	Describe("DismissReport", func() {
		It("dismisses a report", func() {
			reportID := uuid.New()
			mock.DismissReportFn = func(_ context.Context, id uuid.UUID) error {
				Expect(id).To(Equal(reportID))
				return nil
			}

			req := adminReq(http.MethodPost, "/admin/reports/"+reportID.String()+"/dismiss", "")
			req = testutil.RequestWithChiParam(req, "id", reportID.String())
			w := httptest.NewRecorder()

			err := h.DismissReport(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp map[string]string
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["status"]).To(Equal("dismissed"))
		})

		It("returns error for bad UUID", func() {
			req := adminReq(http.MethodPost, "/admin/reports/bad/dismiss", "")
			req = testutil.RequestWithChiParam(req, "id", "bad-uuid")
			w := httptest.NewRecorder()

			err := h.DismissReport(w, req)
			Expect(err).To(HaveOccurred())
		})
	})
})
