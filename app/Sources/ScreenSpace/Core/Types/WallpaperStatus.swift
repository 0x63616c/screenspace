import Foundation

enum WallpaperStatus: String, Codable, Sendable {
    case pending
    case pendingReview = "pending_review"
    case approved
    case rejected
}
