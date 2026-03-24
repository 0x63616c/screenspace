package service_test

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/apperr"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/service"
)

var _ = Describe("ReportService", func() {
	var (
		mock *db.MockQuerier
		cfg  *config.Config
		svc  *service.ReportService
		ctx  context.Context
	)

	BeforeEach(func() {
		mock = &db.MockQuerier{}
		cfg = config.DefaultConfig()
		svc = service.NewReportService(mock, cfg)
		ctx = context.Background()
	})

	Describe("Create", func() {
		It("returns 400 for empty reason", func() {
			_, err := svc.Create(ctx, uuid.New(), uuid.New(), "")
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
		})

		It("returns 400 for too-long reason", func() {
			longReason := strings.Repeat("x", cfg.MaxReportLength+1)
			_, err := svc.Create(ctx, uuid.New(), uuid.New(), longReason)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
		})

		It("succeeds with valid reason", func() {
			wpID := uuid.New()
			reporterID := uuid.New()

			report, err := svc.Create(ctx, wpID, reporterID, "inappropriate content")
			Expect(err).NotTo(HaveOccurred())
			Expect(report).NotTo(BeNil())
			Expect(report.WallpaperID).To(Equal(wpID))
			Expect(report.ReporterID).To(Equal(reporterID))
			Expect(report.Reason).To(Equal("inappropriate content"))
		})

		It("wraps DB error as 500", func() {
			mock.CreateReportFn = func(_ context.Context, _ db.CreateReportParams) (db.Report, error) {
				return db.Report{}, fmt.Errorf("connection refused")
			}

			_, err := svc.Create(ctx, uuid.New(), uuid.New(), "valid reason")
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})
	})

	Describe("Dismiss", func() {
		It("succeeds when DB returns no error", func() {
			err := svc.Dismiss(ctx, uuid.New())
			Expect(err).NotTo(HaveOccurred())
		})

		It("wraps DB error as 500", func() {
			mock.DismissReportFn = func(_ context.Context, _ uuid.UUID) error {
				return fmt.Errorf("connection refused")
			}

			err := svc.Dismiss(ctx, uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})
	})
})
