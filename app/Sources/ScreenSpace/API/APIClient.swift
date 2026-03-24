import Foundation

final class APIClient: Sendable {
    let baseURL: String
    private let network: NetworkProviding
    private let keychain: KeychainProviding

    init(
        baseURL: String? = nil,
        network: NetworkProviding = LiveNetwork(),
        keychain: KeychainProviding = LiveKeychain()
    ) {
        self.baseURL = baseURL ?? AppConfig.defaultServerURL
        self.network = network
        self.keychain = keychain
    }

    // MARK: - Auth

    func register(email: String, password: String) async throws -> AuthResponse {
        let body = AuthRequest(email: email, password: password)
        let response: AuthResponse = try await post(path: "/api/v1/auth/register", body: body)
        try keychain.save(key: "auth_token", data: Data(response.token.utf8))
        return response
    }

    func login(email: String, password: String) async throws -> AuthResponse {
        let body = AuthRequest(email: email, password: password)
        let response: AuthResponse = try await post(path: "/api/v1/auth/login", body: body)
        try keychain.save(key: "auth_token", data: Data(response.token.utf8))
        return response
    }

    func me() async throws -> UserResponse {
        try await get(path: "/api/v1/auth/me", authenticated: true)
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
    ) async throws -> WallpaperListResponse {
        var params: [String: String] = ["sort": sort.rawValue, "limit": "\(limit)", "offset": "\(offset)"]
        if let category { params["category"] = category.rawValue }
        if let query { params["q"] = query }
        return try await get(path: "/api/v1/wallpapers", query: params)
    }

    func getWallpaper(id: String) async throws -> WallpaperResponse {
        try await get(path: "/api/v1/wallpapers/\(id)")
    }

    func popularWallpapers(limit: Int = 20, offset: Int = 0) async throws -> WallpaperListResponse {
        try await get(path: "/api/v1/wallpapers/popular", query: ["limit": "\(limit)", "offset": "\(offset)"])
    }

    func recentWallpapers(limit: Int = 20, offset: Int = 0) async throws -> WallpaperListResponse {
        try await get(path: "/api/v1/wallpapers/recent", query: ["limit": "\(limit)", "offset": "\(offset)"])
    }

    // MARK: - Categories

    func listCategories() async throws -> [String] {
        let response: CategoriesResponse = try await get(path: "/api/v1/categories")
        return response.categories
    }

    // MARK: - Upload

    func initiateUpload(title: String, category: Category?, tags: [String]) async throws -> UploadInitResponse {
        let body: [String: Any] = ["title": title, "category": category?.rawValue as Any, "tags": tags]
        let data = try JSONSerialization.data(withJSONObject: body)
        return try await postRaw(path: "/api/v1/wallpapers", body: data, authenticated: true)
    }

    func finalizeUpload(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/wallpapers/\(id)/finalize",
            body: String?.none,
            authenticated: true
        )
    }

    // MARK: - Favorites

    func toggleFavorite(id: String) async throws -> Bool {
        let response: [String: Bool] = try await post(
            path: "/api/v1/wallpapers/\(id)/favorite",
            body: String?.none,
            authenticated: true
        )
        return response["favorited"] ?? false
    }

    func listFavorites(limit: Int = 20, offset: Int = 0) async throws -> WallpaperListResponse {
        try await get(
            path: "/api/v1/me/favorites",
            query: ["limit": "\(limit)", "offset": "\(offset)"],
            authenticated: true
        )
    }

    // MARK: - Reports

    func reportWallpaper(id: String, reason: String) async throws {
        let body = ["reason": reason]
        let _: [String: String] = try await post(
            path: "/api/v1/wallpapers/\(id)/report",
            body: body,
            authenticated: true
        )
    }

    // MARK: - Admin

    func listQueue(limit: Int = 20, offset: Int = 0) async throws -> WallpaperListResponse {
        try await get(
            path: "/api/v1/admin/queue",
            query: ["limit": "\(limit)", "offset": "\(offset)"],
            authenticated: true
        )
    }

    func approveWallpaper(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/admin/queue/\(id)/approve",
            body: String?.none,
            authenticated: true
        )
    }

    func rejectWallpaper(id: String, reason: String) async throws {
        let body = ["reason": reason]
        let _: [String: String] = try await post(
            path: "/api/v1/admin/queue/\(id)/reject",
            body: body,
            authenticated: true
        )
    }

    func listAllWallpapers(
        status: WallpaperStatus? = nil,
        limit: Int = 20,
        offset: Int = 0
    ) async throws -> WallpaperListResponse {
        var params: [String: String] = ["limit": "\(limit)", "offset": "\(offset)"]
        if let status { params["status"] = status.rawValue }
        return try await get(path: "/api/v1/admin/wallpapers", query: params, authenticated: true)
    }

    func editWallpaper(id: String, title: String?, category: String?, tags: [String]?) async throws {
        var body: [String: Any] = [:]
        if let title { body["title"] = title }
        if let category { body["category"] = category }
        if let tags { body["tags"] = tags }
        let data = try JSONSerialization.data(withJSONObject: body)
        let _: [String: String] = try await patchRaw(
            path: "/api/v1/admin/wallpapers/\(id)",
            body: data,
            authenticated: true
        )
    }

    func listUsers(query: String? = nil, limit: Int = 20, offset: Int = 0) async throws -> UserListResponse {
        var params: [String: String] = ["limit": "\(limit)", "offset": "\(offset)"]
        if let query { params["q"] = query }
        return try await get(path: "/api/v1/admin/users", query: params, authenticated: true)
    }

    func banUser(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/admin/users/\(id)/ban",
            body: String?.none,
            authenticated: true
        )
    }

    func unbanUser(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/admin/users/\(id)/unban",
            body: String?.none,
            authenticated: true
        )
    }

    func promoteUser(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/admin/users/\(id)/promote",
            body: String?.none,
            authenticated: true
        )
    }

    func listReports(limit: Int = 20, offset: Int = 0) async throws -> ReportListResponse {
        try await get(
            path: "/api/v1/admin/reports",
            query: ["limit": "\(limit)", "offset": "\(offset)"],
            authenticated: true
        )
    }

    func dismissReport(id: String) async throws {
        let _: [String: String] = try await post(
            path: "/api/v1/admin/reports/\(id)/dismiss",
            body: String?.none,
            authenticated: true
        )
    }

    // MARK: - HTTP Helpers

    func buildURL(path: String, query: [String: String] = [:]) -> URL? {
        var components = URLComponents(string: baseURL + path)
        if !query.isEmpty {
            components?.queryItems = query.sorted(by: { $0.key < $1.key }).map { URLQueryItem(
                name: $0.key,
                value: $0.value
            ) }
        }
        return components?.url
    }

    private func get<T: Decodable>(
        path: String,
        query: [String: String] = [:],
        authenticated: Bool = false
    ) async throws -> T {
        guard let url = buildURL(path: path, query: query) else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        if authenticated { addAuth(&request) }
        let (data, response) = try await network.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func post<T: Decodable, B: Encodable>(
        path: String,
        body: B?,
        authenticated: Bool = false
    ) async throws -> T {
        guard let url = buildURL(path: path) else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let body { request.httpBody = try JSONEncoder().encode(body) }
        if authenticated { addAuth(&request) }
        let (data, response) = try await network.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func postRaw<T: Decodable>(path: String, body: Data, authenticated: Bool = false) async throws -> T {
        guard let url = buildURL(path: path) else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = body
        if authenticated { addAuth(&request) }
        let (data, response) = try await network.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func patchRaw<T: Decodable>(path: String, body: Data, authenticated: Bool = false) async throws -> T {
        guard let url = buildURL(path: path) else { throw APIError.invalidURL }
        var request = URLRequest(url: url)
        request.httpMethod = "PATCH"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = body
        if authenticated { addAuth(&request) }
        let (data, response) = try await network.data(for: request)
        try checkResponse(response, data: data)
        return try JSONDecoder().decode(T.self, from: data)
    }

    private func addAuth(_ request: inout URLRequest) {
        if let tokenData = keychain.load(key: "auth_token"),
           let token = String(data: tokenData, encoding: .utf8)
        {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
    }

    private func checkResponse(_ response: URLResponse, data: Data) throws {
        guard let http = response as? HTTPURLResponse else { throw APIError.invalidResponse }
        guard (200 ..< 300).contains(http.statusCode) else {
            let message = (try? JSONDecoder().decode([String: String].self, from: data))?["error"] ?? "Unknown error"
            throw APIError.httpError(status: http.statusCode, message: message)
        }
    }
}
