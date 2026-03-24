package db

import (
	"context"

	"github.com/google/uuid"
)

// MockQuerier is a test double for the Querier interface.
// Set Fn callback fields for flexible per-test behavior, or use the simple
// fields (e.g. UserRow, WallpaperRow) for straightforward single-return tests.
type MockQuerier struct {
	// Simple return fields (backward compat).
	WallpaperRow      GetWallpaperByIDRow
	WallpaperRowErr   error
	UserRow           User
	UserRowErr        error
	FavoriteExists    bool
	FavoriteExistsErr error
	ReportRow         Report
	ReportRowErr      error

	// Fn callback fields: when set, these are called instead of returning the simple fields.
	CheckFavoriteFn                func(ctx context.Context, arg CheckFavoriteParams) (bool, error)
	CountFavoritesByUserFn         func(ctx context.Context, userID uuid.UUID) (int64, error)
	CountPendingReportsFn          func(ctx context.Context) (int64, error)
	CountUsersFn                   func(ctx context.Context) (int64, error)
	CountUsersWithSearchFn         func(ctx context.Context, query string) (int64, error)
	CountWallpapersFn              func(ctx context.Context, arg CountWallpapersParams) (int64, error)
	CreateReportFn                 func(ctx context.Context, arg CreateReportParams) (Report, error)
	CreateUserFn                   func(ctx context.Context, arg CreateUserParams) (User, error)
	CreateWallpaperFn              func(ctx context.Context, arg CreateWallpaperParams) (CreateWallpaperRow, error)
	DeleteFavoriteFn               func(ctx context.Context, arg DeleteFavoriteParams) error
	DeleteWallpaperFn              func(ctx context.Context, id uuid.UUID) error
	DismissReportFn                func(ctx context.Context, id uuid.UUID) error
	GetUserByEmailFn               func(ctx context.Context, email string) (User, error)
	GetUserByIDFn                  func(ctx context.Context, id uuid.UUID) (User, error)
	GetWallpaperByIDFn             func(ctx context.Context, id uuid.UUID) (GetWallpaperByIDRow, error)
	IncrementDownloadCountFn       func(ctx context.Context, id uuid.UUID) error
	InsertFavoriteFn               func(ctx context.Context, arg InsertFavoriteParams) error
	ListFavoritesByUserFn          func(ctx context.Context, arg ListFavoritesByUserParams) ([]ListFavoritesByUserRow, error)
	ListPendingReportsFn           func(ctx context.Context, arg ListPendingReportsParams) ([]Report, error)
	ListUsersFn                    func(ctx context.Context, arg ListUsersParams) ([]User, error)
	ListUsersWithSearchFn          func(ctx context.Context, arg ListUsersWithSearchParams) ([]User, error)
	ListWallpapersPopularFn        func(ctx context.Context, arg ListWallpapersPopularParams) ([]ListWallpapersPopularRow, error)
	ListWallpapersRecentFn         func(ctx context.Context, arg ListWallpapersRecentParams) ([]ListWallpapersRecentRow, error)
	SetBannedFn                    func(ctx context.Context, arg SetBannedParams) error
	SetRoleFn                      func(ctx context.Context, arg SetRoleParams) error
	UpdateWallpaperAfterFinalizeFn func(ctx context.Context, arg UpdateWallpaperAfterFinalizeParams) (UpdateWallpaperAfterFinalizeRow, error)
	UpdateWallpaperMetadataFn      func(ctx context.Context, arg UpdateWallpaperMetadataParams) error
	UpdateWallpaperStatusFn        func(ctx context.Context, arg UpdateWallpaperStatusParams) error
	UpdateWallpaperStatusReasonFn  func(ctx context.Context, arg UpdateWallpaperStatusWithReasonParams) error
	UpdateWallpaperStorageKeyFn    func(ctx context.Context, arg UpdateWallpaperStorageKeyParams) error
}

func (m *MockQuerier) CheckFavorite(ctx context.Context, arg CheckFavoriteParams) (bool, error) {
	if m.CheckFavoriteFn != nil {
		return m.CheckFavoriteFn(ctx, arg)
	}
	return m.FavoriteExists, m.FavoriteExistsErr
}

func (m *MockQuerier) CountFavoritesByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.CountFavoritesByUserFn != nil {
		return m.CountFavoritesByUserFn(ctx, userID)
	}
	return 0, nil
}

func (m *MockQuerier) CountPendingReports(ctx context.Context) (int64, error) {
	if m.CountPendingReportsFn != nil {
		return m.CountPendingReportsFn(ctx)
	}
	return 0, nil
}

func (m *MockQuerier) CountUsers(ctx context.Context) (int64, error) {
	if m.CountUsersFn != nil {
		return m.CountUsersFn(ctx)
	}
	return 0, nil
}

func (m *MockQuerier) CountUsersWithSearch(ctx context.Context, query string) (int64, error) {
	if m.CountUsersWithSearchFn != nil {
		return m.CountUsersWithSearchFn(ctx, query)
	}
	return 0, nil
}

func (m *MockQuerier) CountWallpapers(ctx context.Context, arg CountWallpapersParams) (int64, error) {
	if m.CountWallpapersFn != nil {
		return m.CountWallpapersFn(ctx, arg)
	}
	return 0, nil
}

func (m *MockQuerier) CreateReport(ctx context.Context, arg CreateReportParams) (Report, error) {
	if m.CreateReportFn != nil {
		return m.CreateReportFn(ctx, arg)
	}
	if m.ReportRowErr != nil {
		return Report{}, m.ReportRowErr
	}
	return Report{
		ID:          uuid.New(),
		WallpaperID: arg.WallpaperID,
		ReporterID:  arg.ReporterID,
		Reason:      arg.Reason,
		Status:      "pending",
	}, nil
}

func (m *MockQuerier) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, arg)
	}
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) CreateWallpaper(ctx context.Context, arg CreateWallpaperParams) (CreateWallpaperRow, error) {
	if m.CreateWallpaperFn != nil {
		return m.CreateWallpaperFn(ctx, arg)
	}
	return CreateWallpaperRow{}, nil
}

func (m *MockQuerier) DeleteFavorite(ctx context.Context, arg DeleteFavoriteParams) error {
	if m.DeleteFavoriteFn != nil {
		return m.DeleteFavoriteFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) DeleteWallpaper(ctx context.Context, id uuid.UUID) error {
	if m.DeleteWallpaperFn != nil {
		return m.DeleteWallpaperFn(ctx, id)
	}
	return nil
}

func (m *MockQuerier) DismissReport(ctx context.Context, id uuid.UUID) error {
	if m.DismissReportFn != nil {
		return m.DismissReportFn(ctx, id)
	}
	return nil
}

func (m *MockQuerier) GetUserByEmail(ctx context.Context, email string) (User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(ctx, id)
	}
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) GetWallpaperByID(ctx context.Context, id uuid.UUID) (GetWallpaperByIDRow, error) {
	if m.GetWallpaperByIDFn != nil {
		return m.GetWallpaperByIDFn(ctx, id)
	}
	return m.WallpaperRow, m.WallpaperRowErr
}

func (m *MockQuerier) IncrementDownloadCount(ctx context.Context, id uuid.UUID) error {
	if m.IncrementDownloadCountFn != nil {
		return m.IncrementDownloadCountFn(ctx, id)
	}
	return nil
}

func (m *MockQuerier) InsertFavorite(ctx context.Context, arg InsertFavoriteParams) error {
	if m.InsertFavoriteFn != nil {
		return m.InsertFavoriteFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) ListFavoritesByUser(ctx context.Context, arg ListFavoritesByUserParams) ([]ListFavoritesByUserRow, error) {
	if m.ListFavoritesByUserFn != nil {
		return m.ListFavoritesByUserFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) ListPendingReports(ctx context.Context, arg ListPendingReportsParams) ([]Report, error) {
	if m.ListPendingReportsFn != nil {
		return m.ListPendingReportsFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error) {
	if m.ListUsersFn != nil {
		return m.ListUsersFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) ListUsersWithSearch(ctx context.Context, arg ListUsersWithSearchParams) ([]User, error) {
	if m.ListUsersWithSearchFn != nil {
		return m.ListUsersWithSearchFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) ListWallpapersPopular(ctx context.Context, arg ListWallpapersPopularParams) ([]ListWallpapersPopularRow, error) {
	if m.ListWallpapersPopularFn != nil {
		return m.ListWallpapersPopularFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) ListWallpapersRecent(ctx context.Context, arg ListWallpapersRecentParams) ([]ListWallpapersRecentRow, error) {
	if m.ListWallpapersRecentFn != nil {
		return m.ListWallpapersRecentFn(ctx, arg)
	}
	return nil, nil
}

func (m *MockQuerier) SetBanned(ctx context.Context, arg SetBannedParams) error {
	if m.SetBannedFn != nil {
		return m.SetBannedFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) SetRole(ctx context.Context, arg SetRoleParams) error {
	if m.SetRoleFn != nil {
		return m.SetRoleFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) UpdateWallpaperAfterFinalize(ctx context.Context, arg UpdateWallpaperAfterFinalizeParams) (UpdateWallpaperAfterFinalizeRow, error) {
	if m.UpdateWallpaperAfterFinalizeFn != nil {
		return m.UpdateWallpaperAfterFinalizeFn(ctx, arg)
	}
	return UpdateWallpaperAfterFinalizeRow{}, nil
}

func (m *MockQuerier) UpdateWallpaperStorageKey(ctx context.Context, arg UpdateWallpaperStorageKeyParams) error {
	if m.UpdateWallpaperStorageKeyFn != nil {
		return m.UpdateWallpaperStorageKeyFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) UpdateWallpaperMetadata(ctx context.Context, arg UpdateWallpaperMetadataParams) error {
	if m.UpdateWallpaperMetadataFn != nil {
		return m.UpdateWallpaperMetadataFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) UpdateWallpaperStatus(ctx context.Context, arg UpdateWallpaperStatusParams) error {
	if m.UpdateWallpaperStatusFn != nil {
		return m.UpdateWallpaperStatusFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) UpdateWallpaperStatusWithReason(ctx context.Context, arg UpdateWallpaperStatusWithReasonParams) error {
	if m.UpdateWallpaperStatusReasonFn != nil {
		return m.UpdateWallpaperStatusReasonFn(ctx, arg)
	}
	return nil
}

var _ Querier = (*MockQuerier)(nil)
