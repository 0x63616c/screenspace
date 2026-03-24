package db

import (
	"context"

	"github.com/google/uuid"
)

// MockQuerier is a test double for the Querier interface.
// Set the fields you need for each test scenario.
type MockQuerier struct {
	WallpaperRow      GetWallpaperByIDRow
	WallpaperRowErr   error
	UserRow           User
	UserRowErr        error
	FavoriteExists    bool
	FavoriteExistsErr error
	ReportRow         Report
	ReportRowErr      error
	CreateWallpaperFn func(ctx context.Context, arg CreateWallpaperParams) (CreateWallpaperRow, error)
	UpdateStatusFn    func(ctx context.Context, arg UpdateWallpaperStatusParams) error
	FinalizeFn        func(ctx context.Context, arg UpdateWallpaperAfterFinalizeParams) (UpdateWallpaperAfterFinalizeRow, error)
}

func (m *MockQuerier) CheckFavorite(_ context.Context, _ CheckFavoriteParams) (bool, error) {
	return m.FavoriteExists, m.FavoriteExistsErr
}

func (m *MockQuerier) CountFavoritesByUser(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) CountPendingReports(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) CountUsers(_ context.Context) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) CountUsersWithSearch(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) CountWallpapers(_ context.Context, _ CountWallpapersParams) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) CreateReport(_ context.Context, arg CreateReportParams) (Report, error) {
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

func (m *MockQuerier) CreateUser(_ context.Context, _ CreateUserParams) (User, error) {
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) CreateWallpaper(ctx context.Context, arg CreateWallpaperParams) (CreateWallpaperRow, error) {
	if m.CreateWallpaperFn != nil {
		return m.CreateWallpaperFn(ctx, arg)
	}
	return CreateWallpaperRow{}, nil
}

func (m *MockQuerier) DeleteFavorite(_ context.Context, _ DeleteFavoriteParams) error {
	return nil
}

func (m *MockQuerier) DeleteWallpaper(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *MockQuerier) DismissReport(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *MockQuerier) GetUserByEmail(_ context.Context, _ string) (User, error) {
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) GetUserByID(_ context.Context, _ uuid.UUID) (User, error) {
	return m.UserRow, m.UserRowErr
}

func (m *MockQuerier) GetWallpaperByID(_ context.Context, _ uuid.UUID) (GetWallpaperByIDRow, error) {
	return m.WallpaperRow, m.WallpaperRowErr
}

func (m *MockQuerier) IncrementDownloadCount(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *MockQuerier) InsertFavorite(_ context.Context, _ InsertFavoriteParams) error {
	return nil
}

func (m *MockQuerier) ListFavoritesByUser(_ context.Context, _ ListFavoritesByUserParams) ([]ListFavoritesByUserRow, error) {
	return nil, nil
}

func (m *MockQuerier) ListPendingReports(_ context.Context, _ ListPendingReportsParams) ([]Report, error) {
	return nil, nil
}

func (m *MockQuerier) ListUsers(_ context.Context, _ ListUsersParams) ([]User, error) {
	return nil, nil
}

func (m *MockQuerier) ListUsersWithSearch(_ context.Context, _ ListUsersWithSearchParams) ([]User, error) {
	return nil, nil
}

func (m *MockQuerier) ListWallpapersPopular(_ context.Context, _ ListWallpapersPopularParams) ([]ListWallpapersPopularRow, error) {
	return nil, nil
}

func (m *MockQuerier) ListWallpapersRecent(_ context.Context, _ ListWallpapersRecentParams) ([]ListWallpapersRecentRow, error) {
	return nil, nil
}

func (m *MockQuerier) SetBanned(_ context.Context, _ SetBannedParams) error {
	return nil
}

func (m *MockQuerier) SetRole(_ context.Context, _ SetRoleParams) error {
	return nil
}

func (m *MockQuerier) UpdateWallpaperAfterFinalize(ctx context.Context, arg UpdateWallpaperAfterFinalizeParams) (UpdateWallpaperAfterFinalizeRow, error) {
	if m.FinalizeFn != nil {
		return m.FinalizeFn(ctx, arg)
	}
	return UpdateWallpaperAfterFinalizeRow{}, nil
}

func (m *MockQuerier) UpdateWallpaperStorageKey(_ context.Context, _ UpdateWallpaperStorageKeyParams) error {
	return nil
}

func (m *MockQuerier) UpdateWallpaperMetadata(_ context.Context, _ UpdateWallpaperMetadataParams) error {
	return nil
}

func (m *MockQuerier) UpdateWallpaperStatus(ctx context.Context, arg UpdateWallpaperStatusParams) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, arg)
	}
	return nil
}

func (m *MockQuerier) UpdateWallpaperStatusWithReason(_ context.Context, _ UpdateWallpaperStatusWithReasonParams) error {
	return nil
}

var _ Querier = (*MockQuerier)(nil)
