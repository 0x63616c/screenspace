import Foundation
import Testing
@testable import ScreenSpace
@testable import ScreenSpaceAPI

struct APIGeneratedTypesTests {
    @Test("Decode Wallpaper from JSON")
    func decodeWallpaper() throws {
        let json = Data("""
        {"id":"abc","title":"Ocean","uploader_id":"u1","status":"approved",
         "resolution":"3840x2160","width":3840,"height":2160,
         "duration":30.0,"file_size":85000000,"format":"h264","download_count":100,
         "category":"nature","tags":["ocean","waves"],
         "storage_key":"wallpapers/abc.mp4",
         "thumbnail_url":"https://cdn.example.com/thumb.jpg",
         "preview_url":"https://cdn.example.com/preview.mp4",
         "created_at":"2026-03-24T00:00:00Z","updated_at":"2026-03-24T00:00:00Z"}
        """.utf8)

        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .iso8601
        let wallpaper = try decoder.decode(Components.Schemas.Wallpaper.self, from: json)
        #expect(wallpaper.title == "Ocean")
        #expect(wallpaper.width == 3840)
        #expect(wallpaper.fileSize == 85_000_000)
        #expect(wallpaper.status == .approved)
    }

    @Test("Decode AuthResponse from JSON")
    func decodeAuthResponse() throws {
        let json = Data("""
        {"token":"jwt-token-here","role":"admin"}
        """.utf8)

        let auth = try JSONDecoder().decode(Components.Schemas.AuthResponse.self, from: json)
        #expect(auth.token == "jwt-token-here")
        #expect(auth.role == .admin)
    }

    @Test("WallpaperCategory has all expected cases")
    func wallpaperCategoryEnum() {
        let allCases = Components.Schemas.WallpaperCategory.allCases
        #expect(allCases.count == 8)
        #expect(allCases.contains(.nature))
        #expect(allCases.contains(.abstract))
        #expect(allCases.contains(.urban))
        #expect(allCases.contains(.cinematic))
        #expect(allCases.contains(.space))
        #expect(allCases.contains(.underwater))
        #expect(allCases.contains(.minimal))
        #expect(allCases.contains(.other))
    }

    @Test("WallpaperStatus has all expected cases")
    func wallpaperStatusEnum() {
        let allCases = Components.Schemas.WallpaperStatus.allCases
        #expect(allCases.count == 4)
        #expect(allCases.contains(.pending))
        #expect(allCases.contains(.pendingReview))
        #expect(allCases.contains(.approved))
        #expect(allCases.contains(.rejected))
    }

    @Test("UserRole has all expected cases")
    func userRoleEnum() {
        let allCases = Components.Schemas.UserRole.allCases
        #expect(allCases.count == 2)
        #expect(allCases.contains(.user))
        #expect(allCases.contains(.admin))
    }

    @Test("Wallpaper toCardData conversion")
    func wallpaperToCardData() {
        let wallpaper = Components.Schemas.Wallpaper(
            id: "w1",
            title: "Ocean Sunrise",
            uploaderId: "u1",
            status: .approved,
            tags: ["ocean"],
            resolution: "3840x2160",
            width: 3840,
            height: 2160,
            duration: 30.0,
            fileSize: 85_000_000,
            format: "h264",
            downloadCount: 100,
            storageKey: "wallpapers/w1.mp4",
            thumbnailUrl: "https://cdn.example.com/thumb.jpg",
            previewUrl: "https://cdn.example.com/preview.mp4",
            createdAt: Date(),
            updatedAt: Date()
        )

        let card = wallpaper.toCardData()
        #expect(card.id == "w1")
        #expect(card.title == "Ocean Sunrise")
        #expect(card.width == 3840)
        #expect(card.height == 2160)
        #expect(card.duration == 30.0)
        #expect(card.thumbnailURL != nil)
    }
}
