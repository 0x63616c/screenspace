import Foundation
import OpenAPIRuntime
import OpenAPIURLSession
import ScreenSpaceAPI

final class APIClient: Sendable {
    let baseURL: String
    private let keychain: KeychainProviding
    private let client: Client

    init(
        baseURL: String? = nil,
        keychain: KeychainProviding = LiveKeychain()
    ) {
        let resolvedBaseURL = baseURL ?? AppConfig.defaultServerURL
        self.baseURL = resolvedBaseURL
        self.keychain = keychain
        client = Client(
            serverURL: URL(string: resolvedBaseURL + "/api/v1")!,
            transport: URLSessionTransport(),
            middlewares: [AuthMiddleware(keychain: keychain)]
        )
    }

    // MARK: - Auth

    func register(email: String, password: String) async throws -> Components.Schemas.AuthResponse {
        let response = try await client.register(
            body: .json(.init(email: email, password: password))
        )
        let result = try response.created.body.json
        try keychain.save(key: "auth_token", data: Data(result.token.utf8))
        return result
    }

    func login(email: String, password: String) async throws -> Components.Schemas.AuthResponse {
        let response = try await client.login(
            body: .json(.init(email: email, password: password))
        )
        let result = try response.ok.body.json
        try keychain.save(key: "auth_token", data: Data(result.token.utf8))
        return result
    }

    func me() async throws -> Components.Schemas.MeResponse {
        let response = try await client.getMe()
        return try response.ok.body.json
    }

    func logout() {
        keychain.delete(key: "auth_token")
    }

    // MARK: - Wallpapers

    func listWallpapers(
        sort: SortOrder = .recent,
        category: Category? = nil,
        query: String? = nil,
        limit: Int = 20,
        offset: Int = 0
    ) async throws -> Components.Schemas.WallpaperListResponse {
        let sortPayload = Operations.ListWallpapers.Input.Query.SortPayload(rawValue: sort.rawValue)
        let categoryPayload = category.flatMap { Components.Schemas.WallpaperCategory(rawValue: $0.rawValue) }
        let response = try await client.listWallpapers(
            query: .init(
                sort: sortPayload,
                category: categoryPayload,
                q: query,
                limit: limit,
                offset: offset
            )
        )
        return try response.ok.body.json
    }

    func getWallpaper(id: String) async throws -> Components.Schemas.Wallpaper {
        let response = try await client.getWallpaper(path: .init(id: id))
        return try response.ok.body.json
    }

    func popularWallpapers(limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas.WallpaperListResponse {
        let response = try await client.listWallpapersPopular(
            query: .init(limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    func recentWallpapers(limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas.WallpaperListResponse {
        let response = try await client.listWallpapersRecent(
            query: .init(limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    // MARK: - Categories

    func listCategories() async throws -> [Components.Schemas.WallpaperCategory] {
        let response = try await client.listCategories()
        return try response.ok.body.json.categories
    }

    // MARK: - Upload

    func initiateUpload(
        title: String,
        category: Category?,
        tags: [String]
    ) async throws -> Components.Schemas.CreateWallpaperResponse {
        let categoryPayload = category.flatMap { Components.Schemas.WallpaperCategory(rawValue: $0.rawValue) }
        let response = try await client.createWallpaper(
            body: .json(.init(title: title, category: categoryPayload, tags: tags))
        )
        return try response.created.body.json
    }

    func finalizeUpload(id: String) async throws {
        let response = try await client.finalizeWallpaper(path: .init(id: id))
        _ = try response.ok
    }

    // MARK: - Favorites

    func toggleFavorite(id: String) async throws -> Bool {
        let response = try await client.toggleFavorite(path: .init(id: id))
        return try response.ok.body.json.favorited
    }

    func listFavorites(limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas.WallpaperListResponse {
        let response = try await client.listFavorites(
            query: .init(limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    // MARK: - Reports

    func reportWallpaper(id: String, reason: String) async throws {
        let response = try await client.reportWallpaper(
            path: .init(id: id),
            body: .json(.init(reason: reason))
        )
        _ = try response.created
    }

    // MARK: - Admin

    func listQueue(limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas.WallpaperListResponse {
        let response = try await client.getAdminQueue(
            query: .init(limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    func approveWallpaper(id: String) async throws {
        let response = try await client.approveWallpaper(path: .init(id: id))
        _ = try response.ok
    }

    func rejectWallpaper(id: String, reason: String) async throws {
        let response = try await client.rejectWallpaper(
            path: .init(id: id),
            body: .json(.init(reason: reason))
        )
        _ = try response.ok
    }

    func listAllWallpapers(
        status: WallpaperStatus? = nil,
        limit: Int = 20,
        offset: Int = 0
    ) async throws -> Components.Schemas.WallpaperListResponse {
        let statusPayload = status.flatMap { Components.Schemas.WallpaperStatus(rawValue: $0.rawValue) }
        let response = try await client.adminListWallpapers(
            query: .init(status: statusPayload, limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    func editWallpaper(id: String, title: String?, category: String?, tags: [String]?) async throws {
        let categoryPayload = category.flatMap { Components.Schemas.WallpaperCategory(rawValue: $0) }
        let response = try await client.adminEditWallpaper(
            path: .init(id: id),
            body: .json(.init(title: title, category: categoryPayload, tags: tags))
        )
        _ = try response.ok
    }

    func listUsers(query: String? = nil, limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas
        .UserListResponse
    {
        let response = try await client.adminListUsers(
            query: .init(q: query, limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    func banUser(id: String) async throws {
        let response = try await client.banUser(path: .init(id: id))
        _ = try response.ok
    }

    func unbanUser(id: String) async throws {
        let response = try await client.unbanUser(path: .init(id: id))
        _ = try response.ok
    }

    func promoteUser(id: String) async throws {
        let response = try await client.promoteUser(path: .init(id: id))
        _ = try response.ok
    }

    func listReports(limit: Int = 20, offset: Int = 0) async throws -> Components.Schemas.ReportListResponse {
        let response = try await client.adminListReports(
            query: .init(limit: limit, offset: offset)
        )
        return try response.ok.body.json
    }

    func dismissReport(id: String) async throws {
        let response = try await client.dismissReport(path: .init(id: id))
        _ = try response.ok
    }
}
