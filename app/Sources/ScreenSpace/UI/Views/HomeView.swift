import SwiftUI

struct HomeView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                HeroSection(wallpaper: Self.placeholderData.first)
                ShelfRow(title: "Popular", wallpapers: Self.placeholderData)
                ShelfRow(title: "Recently Added", wallpapers: Self.placeholderData)
                ShelfRow(title: "Nature", wallpapers: Self.placeholderData)
            }
            .padding(.vertical)
        }
    }

    private static let placeholderData: [WallpaperCardData] = (0..<8).map { i in
        WallpaperCardData(
            id: "\(i)",
            title: "Wallpaper \(i + 1)",
            thumbnailURL: nil,
            width: [1920, 2560, 3840][i % 3],
            height: [1080, 1440, 2160][i % 3],
            duration: Double(20 + i * 5)
        )
    }
}
