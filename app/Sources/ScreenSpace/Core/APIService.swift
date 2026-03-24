import Foundation

final class APIService: APIProviding {
    private let client: APIClient

    init(client: APIClient) {
        self.client = client
    }

    // MARK: - Auth

    func login(email: String, password: String) async throws -> AuthToken {
        let response = try await client.login(email: email, password: password)
        return AuthToken(token: response.token, role: response.role)
    }

    func register(email: String, password: String) async throws -> AuthToken {
        let response = try await client.register(email: email, password: password)
        return AuthToken(token: response.token, role: response.role)
    }

    func me() async throws -> UserInfo {
        let response = try await client.me()
        return UserInfo(
            id: response.id,
            email: response.email,
            role: response.role,
            banned: response.banned ?? false,
            createdAt: response.createdAt
        )
    }

    func logout() {
        client.logout()
    }

    // MARK: - Wallpapers

    func popularWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.popularWallpapers(limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    func recentWallpapers(limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.recentWallpapers(limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    func listWallpapers(category: Category?, query: String?, sort: SortOrder, limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.listWallpapers(sort: sort, category: category, query: query, limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    func getWallpaper(id: String) async throws -> WallpaperDetail {
        let wp = try await client.getWallpaper(id: id)
        return mapWallpaperDetail(wp)
    }

    func listCategories() async throws -> [Category] {
        let strings = try await client.listCategories()
        return strings.compactMap { Category(rawValue: $0) }
    }

    // MARK: - Favorites

    func toggleFavorite(id: String) async throws -> Bool {
        try await client.toggleFavorite(id: id)
    }

    func listFavorites(limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.listFavorites(limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    // MARK: - Reports

    func reportWallpaper(id: String, reason: String) async throws {
        try await client.reportWallpaper(id: id, reason: reason)
    }

    // MARK: - Upload

    func initiateUpload(title: String, category: Category?, tags: [String]) async throws -> UploadTicket {
        let response = try await client.initiateUpload(title: title, category: category, tags: tags)
        return UploadTicket(id: response.id, uploadURL: response.uploadURL)
    }

    func finalizeUpload(id: String) async throws {
        try await client.finalizeUpload(id: id)
    }

    // MARK: - Admin

    func listQueue(limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.listQueue(limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    func approveWallpaper(id: String) async throws {
        try await client.approveWallpaper(id: id)
    }

    func rejectWallpaper(id: String, reason: String) async throws {
        try await client.rejectWallpaper(id: id, reason: reason)
    }

    func listAdminWallpapers(status: WallpaperStatus?, limit: Int, offset: Int) async throws -> PagedWallpapers {
        let response = try await client.listAllWallpapers(status: status, limit: limit, offset: offset)
        return mapWallpaperList(response)
    }

    func editWallpaper(id: String, title: String?, category: Category?, tags: [String]?) async throws {
        try await client.editWallpaper(id: id, title: title, category: category?.rawValue, tags: tags)
    }

    func listUsers(query: String?, limit: Int, offset: Int) async throws -> PagedUsers {
        let response = try await client.listUsers(query: query, limit: limit, offset: offset)
        return PagedUsers(
            items: response.users.map {
                UserInfo(id: $0.id, email: $0.email, role: $0.role, banned: $0.banned ?? false, createdAt: $0.createdAt)
            },
            total: response.total,
            limit: limit,
            offset: offset
        )
    }

    func banUser(id: String) async throws {
        try await client.banUser(id: id)
    }

    func unbanUser(id: String) async throws {
        try await client.unbanUser(id: id)
    }

    func promoteUser(id: String) async throws {
        try await client.promoteUser(id: id)
    }

    func listReports(limit: Int, offset: Int) async throws -> PagedReports {
        let response = try await client.listReports(limit: limit, offset: offset)
        return PagedReports(
            items: response.reports.map {
                ReportInfo(id: $0.id, wallpaperID: $0.wallpaperID, reporterID: $0.reporterID, reason: $0.reason, createdAt: $0.createdAt)
            },
            total: response.total,
            limit: limit,
            offset: offset
        )
    }

    func dismissReport(id: String) async throws {
        try await client.dismissReport(id: id)
    }

    // MARK: - Private Mapping

    private func mapWallpaperList(_ response: WallpaperListResponse) -> PagedWallpapers {
        PagedWallpapers(
            items: response.wallpapers.map { wp in
                WallpaperCardData(
                    id: wp.id,
                    title: wp.title,
                    thumbnailURL: wp.thumbnailURL.flatMap { URL(string: $0) },
                    width: wp.width,
                    height: wp.height,
                    duration: wp.duration
                )
            },
            total: response.total,
            limit: 20,
            offset: 0
        )
    }

    private func mapWallpaperDetail(_ wp: WallpaperResponse) -> WallpaperDetail {
        WallpaperDetail(
            id: wp.id,
            title: wp.title,
            resolution: wp.resolution,
            width: wp.width,
            height: wp.height,
            duration: wp.duration,
            fileSize: wp.fileSize,
            format: wp.format,
            downloadCount: wp.downloadCount,
            category: wp.category,
            tags: wp.tags ?? [],
            thumbnailURL: wp.thumbnailURL.flatMap { URL(string: $0) },
            previewURL: wp.previewURL.flatMap { URL(string: $0) },
            downloadURL: wp.downloadURL.flatMap { URL(string: $0) },
            uploaderEmail: wp.uploaderEmail,
            status: wp.status,
            createdAt: wp.createdAt
        )
    }
}
