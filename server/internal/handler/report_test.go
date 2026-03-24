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
	"github.com/0x63616c/screenspace/server/internal/testutil"
	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("ReportHandler", func() {
	var (
		mock *db.MockQuerier
		svc  *service.ReportService
		h    *handler.ReportHandler
	)

	BeforeEach(func() {
		cfg := config.DefaultConfig()
		mock = &db.MockQuerier{}
		svc = service.NewReportService(mock, cfg)
		h = handler.NewReportHandler(svc)
	})

	Describe("Create", func() {
		It("creates a report and returns 201", func() {
			userID := uuid.New()
			wpID := uuid.New()
			reportID := uuid.New()

			mock.CreateReportFn = func(_ context.Context, arg db.CreateReportParams) (db.Report, error) {
				return db.Report{
					ID:          reportID,
					WallpaperID: arg.WallpaperID,
					ReporterID:  arg.ReporterID,
					Reason:      arg.Reason,
					Status:      "pending",
				}, nil
			}

			body := `{"reason":"inappropriate content"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers/"+wpID.String()+"/report", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			req = testutil.RequestWithChiParam(req, "id", wpID.String())
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusCreated))

			var resp map[string]any
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp["id"]).To(Equal(reportID.String()))
			Expect(resp["reason"]).To(Equal("inappropriate content"))
		})

		It("returns error for bad wallpaper UUID", func() {
			userID := uuid.New()

			body := `{"reason":"bad content"}`
			req := httptest.NewRequest(http.MethodPost, "/wallpapers/bad/report", strings.NewReader(body))
			req = testutil.RequestWithClaims(req, userID.String(), types.RoleUser)
			req = testutil.RequestWithChiParam(req, "id", "bad-uuid")
			w := httptest.NewRecorder()

			err := h.Create(w, req)
			Expect(err).To(HaveOccurred())
		})
	})
})
