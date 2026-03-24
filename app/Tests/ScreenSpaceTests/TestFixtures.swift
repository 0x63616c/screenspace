import Foundation
@testable import ScreenSpace

enum TestFixtures {
    static func wallpaperCard(
        id: String = "w1",
        title: String = "Ocean Sunrise",
        thumbnailURL: URL? = nil
    ) -> WallpaperCardData {
        WallpaperCardData(id: id, title: title, thumbnailURL: thumbnailURL, width: 3840, height: 2160, duration: 30)
    }

    static func wallpaperDetail(
        id: String = "w1",
        title: String = "Ocean Sunrise"
    ) -> WallpaperDetail {
        WallpaperDetail(
            id: id, title: title, resolution: "3840x2160",
            width: 3840, height: 2160, duration: 30, fileSize: 85_000_000,
            format: "h264", downloadCount: 100, category: .nature,
            tags: ["ocean", "waves"], thumbnailURL: nil, previewURL: nil,
            downloadURL: URL(string: "https://example.com/\(id).mp4"),
            uploaderEmail: nil, status: .approved, createdAt: nil
        )
    }

    static func userInfo(
        id: String = "u1",
        email: String = "test@example.com",
        role: UserRole = .user
    ) -> UserInfo {
        UserInfo(id: id, email: email, role: role, banned: false, createdAt: nil)
    }

    static func pagedWallpapers(_ count: Int = 3) -> PagedWallpapers {
        let items = (1 ... count).map { i in wallpaperCard(id: "w\(i)", title: "Wallpaper \(i)") }
        return PagedWallpapers(items: items, total: count, limit: 20, offset: 0)
    }

    static func playlist(
        id: String = "p1",
        name: String = "Nature"
    ) -> Playlist {
        Playlist(id: id, name: name, items: [], interval: 0, shuffle: false)
    }
}
