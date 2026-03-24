import Foundation

@MainActor
final class MockAPI: APIProviding {
    // Auth
    var loginResponse: Result<AuthToken, Error> = .failure(APIError.httpError(status: 500))
    var registerResponse: Result<AuthToken, Error> = .failure(APIError.httpError(status: 500))
    var meResponse: Result<UserInfo, Error> = .failure(APIError.httpError(status: 500))
    var logoutCalled = false

    // Wallpapers
    var popularResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 20, offset: 0))
    var recentResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 20, offset: 0))
    var listWallpapersResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 20, offset: 0))
    var listWallpapersCalled = false
    var wallpaperDetailResponse: Result<WallpaperDetail, Error> = .failure(APIError.notFound)
    var categoriesResponse: Result<[Category], Error> = .success(Category.allCases)

    // Favorites
    var toggleFavoriteResponse: Result<Bool, Error> = .success(true)
    var favoritesResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 20, offset: 0))

    // Reports
    var reportResponse: Result<Void, Error> = .success(())
    var reportCalled = false

    // Upload
    var uploadResponse: Result<UploadTicket, Error> = .failure(APIError.httpError(status: 500))
    var finalizeResponse: Result<Void, Error> = .success(())

    // Admin
    var queueResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 50, offset: 0))
    var approveResponse: Result<Void, Error> = .success(())
    var rejectResponse: Result<Void, Error> = .success(())
    var adminWallpapersResponse: Result<PagedWallpapers, Error> = .success(PagedWallpapers(items: [], total: 0, limit: 50, offset: 0))
    var editResponse: Result<Void, Error> = .success(())
    var usersResponse: Result<PagedUsers, Error> = .success(PagedUsers(items: [], total: 0, limit: 50, offset: 0))
    var banResponse: Result<Void, Error> = .success(())
    var banCalled = false
    var unbanResponse: Result<Void, Error> = .success(())
    var promoteResponse: Result<Void, Error> = .success(())
    var reportsResponse: Result<PagedReports, Error> = .success(PagedReports(items: [], total: 0, limit: 50, offset: 0))
    var dismissReportResponse: Result<Void, Error> = .success(())
    var loadUsersCalled = false

    // MARK: - APIProviding

    nonisolated func login(email: String, password: String) async throws -> AuthToken {
        try await MainActor.run { try loginResponse.get() }
    }

    nonisolated func register(email: String, password: String) async throws -> AuthToken {
        try await MainActor.run { try registerResponse.get() }
    }

    nonisolated func me() async throws -> UserInfo {
        try await MainActor.run { try meResponse.get() }
    }

    nonisolated func logout() {
        MainActor.assumeIsolated { logoutCalled = true }
    }

    nonisolated func popularWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run { try popularResponse.get() }
    }

    nonisolated func recentWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run { try recentResponse.get() }
    }

    nonisolated func listWallpapers(category: Category?, query: String?, sort: SortOrder, limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run {
            listWallpapersCalled = true
            return try listWallpapersResponse.get()
        }
    }

    nonisolated func getWallpaper(id: String) async throws -> WallpaperDetail {
        try await MainActor.run { try wallpaperDetailResponse.get() }
    }

    nonisolated func listCategories() async throws -> [Category] {
        try await MainActor.run { try categoriesResponse.get() }
    }

    nonisolated func toggleFavorite(id: String) async throws -> Bool {
        try await MainActor.run { try toggleFavoriteResponse.get() }
    }

    nonisolated func listFavorites(limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run { try favoritesResponse.get() }
    }

    nonisolated func reportWallpaper(id: String, reason: String) async throws {
        try await MainActor.run {
            reportCalled = true
            try reportResponse.get()
        }
    }

    nonisolated func initiateUpload(title: String, category: Category?, tags: [String]) async throws -> UploadTicket {
        try await MainActor.run { try uploadResponse.get() }
    }

    nonisolated func finalizeUpload(id: String) async throws {
        try await MainActor.run { try finalizeResponse.get() }
    }

    nonisolated func listQueue(limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run { try queueResponse.get() }
    }

    nonisolated func approveWallpaper(id: String) async throws {
        try await MainActor.run { try approveResponse.get() }
    }

    nonisolated func rejectWallpaper(id: String, reason: String) async throws {
        try await MainActor.run { try rejectResponse.get() }
    }

    nonisolated func listAdminWallpapers(status: WallpaperStatus?, limit: Int, offset: Int) async throws -> PagedWallpapers {
        try await MainActor.run { try adminWallpapersResponse.get() }
    }

    nonisolated func editWallpaper(id: String, title: String?, category: Category?, tags: [String]?) async throws {
        try await MainActor.run { try editResponse.get() }
    }

    nonisolated func listUsers(query: String?, limit: Int, offset: Int) async throws -> PagedUsers {
        try await MainActor.run {
            loadUsersCalled = true
            return try usersResponse.get()
        }
    }

    nonisolated func banUser(id: String) async throws {
        try await MainActor.run {
            banCalled = true
            try banResponse.get()
        }
    }

    nonisolated func unbanUser(id: String) async throws {
        try await MainActor.run { try unbanResponse.get() }
    }

    nonisolated func promoteUser(id: String) async throws {
        try await MainActor.run { try promoteResponse.get() }
    }

    nonisolated func listReports(limit: Int, offset: Int) async throws -> PagedReports {
        try await MainActor.run { try reportsResponse.get() }
    }

    nonisolated func dismissReport(id: String) async throws {
        try await MainActor.run { try dismissReportResponse.get() }
    }
}
