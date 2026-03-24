package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/0x63616c/screenspace/server/internal/types"
)

var _ = Describe("Types", func() {
	Describe("WallpaperStatus", func() {
		It("validates known statuses", func() {
			for _, s := range []types.WallpaperStatus{
				types.StatusPending,
				types.StatusPendingReview,
				types.StatusApproved,
				types.StatusRejected,
			} {
				Expect(s.Valid()).To(BeTrue(), "expected %q to be valid", s)
			}
		})

		It("rejects unknown statuses", func() {
			Expect(types.WallpaperStatus("garbage").Valid()).To(BeFalse())
		})
	})

	Describe("Category", func() {
		It("validates all known categories", func() {
			for _, c := range types.AllCategories() {
				Expect(c.Valid()).To(BeTrue(), "expected %q to be valid", c)
			}
		})

		It("rejects unknown categories", func() {
			Expect(types.Category("galaxy").Valid()).To(BeFalse())
		})
	})

	Describe("SortOrder", func() {
		It("validates known sort orders", func() {
			Expect(types.SortRecent.Valid()).To(BeTrue())
			Expect(types.SortPopular.Valid()).To(BeTrue())
		})

		It("rejects unknown sort orders", func() {
			Expect(types.SortOrder("trending").Valid()).To(BeFalse())
		})
	})
})
