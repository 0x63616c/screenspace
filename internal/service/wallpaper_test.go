package service_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	db "github.com/0x63616c/screenspace/server/db/generated"
	"github.com/0x63616c/screenspace/server/internal/apperr"
	"github.com/0x63616c/screenspace/server/internal/config"
	"github.com/0x63616c/screenspace/server/internal/service"
	"github.com/0x63616c/screenspace/server/internal/types"
	"github.com/0x63616c/screenspace/server/internal/video"
)

var _ = Describe("WallpaperService", func() {
	var (
		mock *db.MockQuerier
		cfg  *config.Config
		svc  *service.WallpaperService
		ctx  context.Context
	)

	BeforeEach(func() {
		mock = &db.MockQuerier{}
		cfg = config.DefaultConfig()
		ctx = context.Background()
	})

	Describe("Finalize", func() {
		It("returns 400 when wallpaper is not in pending status", func() {
			userID := uuid.New()
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:         wpID,
				UploaderID: userID,
				Status:     string(types.StatusPendingReview),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			_, err := svc.Finalize(ctx, wpID, userID)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
		})

		It("returns 403 when user is not the owner", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:         wpID,
				UploaderID: uuid.New(),
				Status:     string(types.StatusPending),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			_, err := svc.Finalize(ctx, wpID, uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(403))
		})
	})

	Describe("Approve", func() {
		It("returns 400 when wallpaper is not in pending_review status", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusApproved),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			err := svc.Approve(ctx, wpID, uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
		})

		It("succeeds for pending_review wallpaper", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusPendingReview),
			}
			var capturedParams db.UpdateWallpaperStatusParams
			mock.UpdateWallpaperStatusFn = func(_ context.Context, arg db.UpdateWallpaperStatusParams) error {
				capturedParams = arg
				return nil
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			err := svc.Approve(ctx, wpID, uuid.New())
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedParams.ID).To(Equal(wpID))
			Expect(capturedParams.Status).To(Equal(string(types.StatusApproved)))
		})

		It("returns 404 when wallpaper not found", func() {
			mock.WallpaperRowErr = fmt.Errorf("not found")
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			err := svc.Approve(ctx, uuid.New(), uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(404))
		})
	})

	Describe("Reject", func() {
		It("succeeds for pending_review wallpaper", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusPendingReview),
			}
			var capturedParams db.UpdateWallpaperStatusWithReasonParams
			mock.UpdateWallpaperStatusReasonFn = func(_ context.Context, arg db.UpdateWallpaperStatusWithReasonParams) error {
				capturedParams = arg
				return nil
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			err := svc.Reject(ctx, wpID, uuid.New(), "inappropriate content")
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedParams.ID).To(Equal(wpID))
			Expect(capturedParams.Status).To(Equal(string(types.StatusRejected)))
			Expect(*capturedParams.RejectionReason).To(Equal("inappropriate content"))
		})

		It("returns 400 when wallpaper is not in pending_review status", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusApproved),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			err := svc.Reject(ctx, wpID, uuid.New(), "reason")
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
		})
	})

	Describe("GetApproved", func() {
		It("returns approved wallpaper", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusApproved),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			wp, err := svc.GetApproved(ctx, wpID)
			Expect(err).NotTo(HaveOccurred())
			Expect(wp.ID).To(Equal(wpID))
		})

		It("returns 404 when not found", func() {
			mock.WallpaperRowErr = fmt.Errorf("not found")
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			_, err := svc.GetApproved(ctx, uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(404))
		})

		It("returns 404 when wallpaper is not approved", func() {
			wpID := uuid.New()
			mock.WallpaperRow = db.GetWallpaperByIDRow{
				ID:     wpID,
				Status: string(types.StatusPending),
			}
			svc = service.NewWallpaperService(mock, nil, nil, cfg)

			_, err := svc.GetApproved(ctx, wpID)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(404))
		})
	})

	Describe("ValidateVideo", func() {
		var prober *video.MockProber

		BeforeEach(func() {
			prober = &video.MockProber{}
			svc = service.NewWallpaperService(mock, nil, prober, cfg)
		})

		It("returns error when file is too large", func() {
			info := &video.ProbeResult{
				Width:    1920,
				Height:   1080,
				Duration: 30,
				Size:     cfg.MaxFileSize + 1,
				Format:   "h264",
			}
			err := svc.ValidateVideo(info)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
			Expect(appErr.Message).To(ContainSubstring("too large"))
		})

		It("returns error when video is too long", func() {
			info := &video.ProbeResult{
				Width:    1920,
				Height:   1080,
				Duration: cfg.MaxDuration + 1,
				Size:     1024,
				Format:   "h264",
			}
			err := svc.ValidateVideo(info)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
			Expect(appErr.Message).To(ContainSubstring("too long"))
		})

		It("returns error when height is too small", func() {
			info := &video.ProbeResult{
				Width:    1280,
				Height:   720,
				Duration: 30,
				Size:     1024,
				Format:   "h264",
			}
			err := svc.ValidateVideo(info)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
			Expect(appErr.Message).To(ContainSubstring("minimum resolution"))
		})

		It("returns error when codec is not supported", func() {
			info := &video.ProbeResult{
				Width:    1920,
				Height:   1080,
				Duration: 30,
				Size:     1024,
				Format:   "vp9",
			}
			err := svc.ValidateVideo(info)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(400))
			Expect(appErr.Message).To(ContainSubstring("h264 and h265"))
		})

		It("passes for valid video", func() {
			info := &video.ProbeResult{
				Width:    1920,
				Height:   1080,
				Duration: 30,
				Size:     1024,
				Format:   "h264",
			}
			err := svc.ValidateVideo(info)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
