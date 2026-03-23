import Foundation

// MARK: - Auth
struct AuthRequest: Codable {
    let email: String
    let password: String
}

struct AuthResponse: Codable {
    let token: String
    let role: String
}

// MARK: - Categories
struct CategoriesResponse: Codable {
    let categories: [String]

    /// Fallback categories used when the API is unavailable.
    static let fallback = ["nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal", "other"]
}

// MARK: - Wallpapers
struct WallpaperResponse: Codable, Identifiable {
    let id: String
    let title: String
    let resolution: String
    let width: Int
    let height: Int
    let duration: Double
    let fileSize: Int64
    let format: String
    let downloadCount: Int64
    let category: String?
    let tags: [String]?
    let thumbnailURL: String?
    let previewURL: String?
    let downloadURL: String?
    let uploaderEmail: String?
    let status: String?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id, title, resolution, width, height, duration, format, category, tags, status
        case fileSize = "file_size"
        case downloadCount = "download_count"
        case thumbnailURL = "thumbnail_url"
        case previewURL = "preview_url"
        case downloadURL = "download_url"
        case uploaderEmail = "uploader_email"
        case createdAt = "created_at"
    }
}

struct WallpaperListResponse: Codable {
    let wallpapers: [WallpaperResponse]
    let total: Int
}

struct UploadInitResponse: Codable {
    let id: String
    let uploadURL: String

    enum CodingKeys: String, CodingKey {
        case id
        case uploadURL = "upload_url"
    }
}

// MARK: - User
struct UserResponse: Codable, Identifiable {
    let id: String
    let email: String
    let role: String
    let banned: Bool?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id, email, role, banned
        case createdAt = "created_at"
    }
}

struct UserListResponse: Codable {
    let users: [UserResponse]
    let total: Int
}

// MARK: - Reports
struct ReportResponse: Codable, Identifiable {
    let id: String
    let wallpaperID: String
    let reporterID: String
    let reason: String
    let status: String
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case id, reason, status
        case wallpaperID = "wallpaper_id"
        case reporterID = "reporter_id"
        case createdAt = "created_at"
    }
}

struct ReportListResponse: Codable {
    let reports: [ReportResponse]
    let total: Int
}
