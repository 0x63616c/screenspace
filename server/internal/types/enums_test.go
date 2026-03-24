package types_test

import (
	"testing"

	"github.com/0x63616c/screenspace/server/internal/types"
)

func TestWallpaperStatus_Valid(t *testing.T) {
	t.Parallel()
	valid := []types.WallpaperStatus{
		types.StatusPending,
		types.StatusPendingReview,
		types.StatusApproved,
		types.StatusRejected,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if types.WallpaperStatus("garbage").Valid() {
		t.Error("expected garbage to be invalid")
	}
}

func TestCategory_Valid(t *testing.T) {
	t.Parallel()
	for _, c := range types.AllCategories() {
		if !c.Valid() {
			t.Errorf("expected %q to be valid", c)
		}
	}
	if types.Category("galaxy").Valid() {
		t.Error("expected galaxy to be invalid")
	}
}

func TestSortOrder_Valid(t *testing.T) {
	t.Parallel()
	if !types.SortRecent.Valid() || !types.SortPopular.Valid() {
		t.Error("expected both sort orders to be valid")
	}
	if types.SortOrder("trending").Valid() {
		t.Error("expected trending to be invalid")
	}
}
