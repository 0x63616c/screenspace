import SwiftUI

struct ShelfRow: View {
    let title: String
    let wallpapers: [WallpaperCardData]
    var onSelectWallpaper: ((WallpaperCardData) -> Void)? = nil
    var onSeeAll: (() -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text(title)
                    .font(.title3)
                    .fontWeight(.bold)

                Spacer()

                Button("See All") { onSeeAll?() }
                    .buttonStyle(.plain)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 20)

            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(spacing: 14) {
                    ForEach(wallpapers) { wallpaper in
                        WallpaperCard(data: wallpaper, onTap: { onSelectWallpaper?(wallpaper) })
                    }
                }
                .padding(.horizontal, 20)
            }
        }
    }
}
