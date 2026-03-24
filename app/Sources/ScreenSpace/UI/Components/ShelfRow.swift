import SwiftUI

struct ShelfRow: View {
    let title: String
    let wallpapers: [WallpaperCardData]
    var onSelectWallpaper: ((WallpaperCardData) -> Void)? = nil
    var onSeeAll: (() -> Void)? = nil

    var body: some View {
        VStack(alignment: .leading, spacing: Spacing.md) {
            HStack {
                Text(title)
                    .font(Typography.sectionTitle)

                Spacer()

                Button("See All") { onSeeAll?() }
                    .buttonStyle(.plain)
                    .font(Typography.cardTitle)
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, Spacing.xl)

            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(spacing: Spacing.md) {
                    ForEach(wallpapers) { wallpaper in
                        WallpaperCard(data: wallpaper, onTap: { onSelectWallpaper?(wallpaper) })
                    }
                }
                .padding(.horizontal, Spacing.xl)
            }
        }
    }
}
