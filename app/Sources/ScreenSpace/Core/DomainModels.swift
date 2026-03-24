import Foundation

struct AuthToken: Sendable {
    let token: String
    let role: UserRole
}

struct UserInfo: Identifiable, Sendable, Equatable {
    let id: String
    let email: String
    let role: UserRole
    let banned: Bool
    let createdAt: String?
}

struct WallpaperCardData: Identifiable, Sendable, Equatable {
    let id: String
    let title: String
    let thumbnailURL: URL?
    let width: Int
    let height: Int
    let duration: Double

    var durationLabel: String {
        "\(Int(duration))s"
    }
}

struct WallpaperDetail: Identifiable, Sendable, Equatable {
    let id: String
    let title: String
    let resolution: String
    let width: Int
    let height: Int
    let duration: Double
    let fileSize: Int64
    let format: String
    let downloadCount: Int64
    let category: Category?
    let tags: [String]
    let thumbnailURL: URL?
    let previewURL: URL?
    let downloadURL: URL?
    let uploaderEmail: String?
    let status: WallpaperStatus?
    let createdAt: String?
}

struct PagedWallpapers: Sendable {
    let items: [WallpaperCardData]
    let total: Int
    let limit: Int
    let offset: Int
}

struct PagedUsers: Sendable {
    let items: [UserInfo]
    let total: Int
    let limit: Int
    let offset: Int
}

struct ReportInfo: Identifiable, Sendable, Equatable {
    let id: String
    let wallpaperID: String
    let reporterID: String
    let reason: String
    let createdAt: String?
}

struct PagedReports: Sendable {
    let items: [ReportInfo]
    let total: Int
    let limit: Int
    let offset: Int
}

struct UploadTicket: Sendable {
    let id: String
    let uploadURL: String
}
