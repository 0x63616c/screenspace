import Foundation

protocol APIProviding: Sendable {
    // Auth
    func login(email: String, password: String) async throws -> AuthToken
    func register(email: String, password: String) async throws -> AuthToken
    func me() async throws -> UserInfo
    func logout()

    // Wallpapers
    func popularWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers
    func recentWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers
    func listWallpapers(category: Category?, query: String?, sort: SortOrder, limit: Int, offset: Int) async throws
        -> PagedWallpapers
    func getWallpaper(id: String) async throws -> WallpaperDetail
    func listCategories() async throws -> [Category]

    // Favorites
    func toggleFavorite(id: String) async throws -> Bool
    func listFavorites(limit: Int, offset: Int) async throws -> PagedWallpapers

    /// Reports
    func reportWallpaper(id: String, reason: String) async throws

    // Upload
    func initiateUpload(title: String, category: Category?, tags: [String]) async throws -> UploadTicket
    func finalizeUpload(id: String) async throws

    // Admin
    func listQueue(limit: Int, offset: Int) async throws -> PagedWallpapers
    func approveWallpaper(id: String) async throws
    func rejectWallpaper(id: String, reason: String) async throws
    func listAdminWallpapers(status: WallpaperStatus?, limit: Int, offset: Int) async throws -> PagedWallpapers
    func editWallpaper(id: String, title: String?, category: Category?, tags: [String]?) async throws
    func listUsers(query: String?, limit: Int, offset: Int) async throws -> PagedUsers
    func banUser(id: String) async throws
    func unbanUser(id: String) async throws
    func promoteUser(id: String) async throws
    func listReports(limit: Int, offset: Int) async throws -> PagedReports
    func dismissReport(id: String) async throws
}

enum APIError: Error, LocalizedError {
    case notFound
    case forbidden
    case httpError(status: Int, message: String? = nil)
    case invalidURL
    case invalidResponse

    var errorDescription: String? {
        switch self {
        case .notFound: return "Not found"
        case .forbidden: return "Forbidden"
        case let .httpError(status, message):
            if let message { return "HTTP \(status): \(message)" }
            return "Server error (\(status))"
        case .invalidURL: return "Invalid URL"
        case .invalidResponse: return "Invalid response"
        }
    }
}
