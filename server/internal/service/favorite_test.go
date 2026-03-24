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
	"github.com/0x63616c/screenspace/server/internal/service"
)

var _ = Describe("FavoriteService", func() {
	var (
		mock *db.MockQuerier
		svc  *service.FavoriteService
		ctx  context.Context
	)

	BeforeEach(func() {
		mock = &db.MockQuerier{}
		svc = service.NewFavoriteService(mock)
		ctx = context.Background()
	})

	Describe("Toggle", func() {
		It("adds favorite when it does not exist", func() {
			userID := uuid.New()
			wpID := uuid.New()
			mock.CheckFavoriteFn = func(_ context.Context, arg db.CheckFavoriteParams) (bool, error) {
				return false, nil
			}
			var insertCalled bool
			mock.InsertFavoriteFn = func(_ context.Context, arg db.InsertFavoriteParams) error {
				insertCalled = true
				Expect(arg.UserID).To(Equal(userID))
				Expect(arg.WallpaperID).To(Equal(wpID))
				return nil
			}

			added, err := svc.Toggle(ctx, userID, wpID)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(BeTrue())
			Expect(insertCalled).To(BeTrue())
		})

		It("removes favorite when it already exists", func() {
			userID := uuid.New()
			wpID := uuid.New()
			mock.CheckFavoriteFn = func(_ context.Context, _ db.CheckFavoriteParams) (bool, error) {
				return true, nil
			}
			var deleteCalled bool
			mock.DeleteFavoriteFn = func(_ context.Context, arg db.DeleteFavoriteParams) error {
				deleteCalled = true
				Expect(arg.UserID).To(Equal(userID))
				Expect(arg.WallpaperID).To(Equal(wpID))
				return nil
			}

			added, err := svc.Toggle(ctx, userID, wpID)
			Expect(err).NotTo(HaveOccurred())
			Expect(added).To(BeFalse())
			Expect(deleteCalled).To(BeTrue())
		})

		It("propagates check error", func() {
			mock.CheckFavoriteFn = func(_ context.Context, _ db.CheckFavoriteParams) (bool, error) {
				return false, fmt.Errorf("db down")
			}

			_, err := svc.Toggle(ctx, uuid.New(), uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})

		It("propagates insert error", func() {
			mock.CheckFavoriteFn = func(_ context.Context, _ db.CheckFavoriteParams) (bool, error) {
				return false, nil
			}
			mock.InsertFavoriteFn = func(_ context.Context, _ db.InsertFavoriteParams) error {
				return fmt.Errorf("insert failed")
			}

			_, err := svc.Toggle(ctx, uuid.New(), uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})

		It("propagates delete error", func() {
			mock.CheckFavoriteFn = func(_ context.Context, _ db.CheckFavoriteParams) (bool, error) {
				return true, nil
			}
			mock.DeleteFavoriteFn = func(_ context.Context, _ db.DeleteFavoriteParams) error {
				return fmt.Errorf("delete failed")
			}

			_, err := svc.Toggle(ctx, uuid.New(), uuid.New())
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})
	})

	Describe("ListByUser", func() {
		It("returns favorites and total count", func() {
			userID := uuid.New()
			expectedRows := []db.ListFavoritesByUserRow{
				{ID: uuid.New(), Title: "wallpaper 1"},
				{ID: uuid.New(), Title: "wallpaper 2"},
			}
			mock.CountFavoritesByUserFn = func(_ context.Context, _ uuid.UUID) (int64, error) {
				return 2, nil
			}
			mock.ListFavoritesByUserFn = func(_ context.Context, arg db.ListFavoritesByUserParams) ([]db.ListFavoritesByUserRow, error) {
				Expect(arg.UserID).To(Equal(userID))
				Expect(arg.Lim).To(Equal(int32(20)))
				Expect(arg.Off).To(Equal(int32(0)))
				return expectedRows, nil
			}

			rows, total, err := svc.ListByUser(ctx, userID, 20, 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(rows).To(HaveLen(2))
		})

		It("propagates count error", func() {
			mock.CountFavoritesByUserFn = func(_ context.Context, _ uuid.UUID) (int64, error) {
				return 0, fmt.Errorf("count failed")
			}

			_, _, err := svc.ListByUser(ctx, uuid.New(), 20, 0)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})

		It("propagates list error", func() {
			mock.CountFavoritesByUserFn = func(_ context.Context, _ uuid.UUID) (int64, error) {
				return 5, nil
			}
			mock.ListFavoritesByUserFn = func(_ context.Context, _ db.ListFavoritesByUserParams) ([]db.ListFavoritesByUserRow, error) {
				return nil, fmt.Errorf("list failed")
			}

			_, _, err := svc.ListByUser(ctx, uuid.New(), 20, 0)
			appErr, ok := errors.AsType[*apperr.Error](err)
			Expect(ok).To(BeTrue())
			Expect(appErr.Status).To(Equal(500))
		})
	})
})
