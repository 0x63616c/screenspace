import SwiftUI

struct HomeView: View {
    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 28) {
                HeroSection(wallpaper: Self.placeholderData.first)
                    .padding(.horizontal, 20)

                ShelfRow(title: "Popular", wallpapers: Self.placeholderData)
                ShelfRow(title: "Recently Added", wallpapers: Self.placeholderData)
                ShelfRow(title: "Nature", wallpapers: Self.placeholderData)
                ShelfRow(title: "Abstract", wallpapers: Self.placeholderData)
            }
            .padding(.vertical, 20)
        }
        .scrollContentBackground(.hidden)
    }

    private static let placeholderData: [WallpaperCardData] = (0..<8).map { i in
        WallpaperCardData(
            id: "\(i)",
            title: ["Sea Cliffs", "Mountain Dawn", "City Lights", "Northern Lights", "Ocean Waves", "Forest Rain", "Desert Storm", "Sunset Beach"][i],
            thumbnailURL: nil,
            width: [1920, 2560, 3840, 3840, 2560, 1920, 3840, 2560][i],
            height: [1080, 1440, 2160, 2160, 1440, 1080, 2160, 1440][i],
            duration: Double(20 + i * 5)
        )
    }
}
