import Testing
import Foundation
@testable import ScreenSpace

@Suite("Typed Enums")
struct TypedEnumTests {
    @Test("WallpaperStatus round-trips through Codable")
    func wallpaperStatusCodable() throws {
        let statuses: [WallpaperStatus] = [.pending, .pendingReview, .approved, .rejected]
        for status in statuses {
            let encoded = try JSONEncoder().encode(status)
            let decoded = try JSONDecoder().decode(WallpaperStatus.self, from: encoded)
            #expect(decoded == status)
        }
    }

    @Test("WallpaperStatus pendingReview has correct raw value")
    func pendingReviewRawValue() {
        #expect(WallpaperStatus.pendingReview.rawValue == "pending_review")
    }

    @Test("UserRole round-trips through Codable")
    func userRoleCodable() throws {
        for role in [UserRole.user, UserRole.admin] {
            let encoded = try JSONEncoder().encode(role)
            let decoded = try JSONDecoder().decode(UserRole.self, from: encoded)
            #expect(decoded == role)
        }
    }

    @Test("Category is CaseIterable with 8 cases")
    func categoryCount() {
        #expect(Category.allCases.count == 8)
    }

    @Test("SortOrder raw values match API contract")
    func sortOrderRawValues() {
        #expect(SortOrder.recent.rawValue == "recent")
        #expect(SortOrder.popular.rawValue == "popular")
    }
}
