import SwiftUI

struct ShelfRow: View {
    let title: String
    let wallpapers: [WallpaperCardData]

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.title3)
                .fontWeight(.semibold)
                .padding(.horizontal)

            ScrollView(.horizontal, showsIndicators: false) {
                LazyHStack(spacing: 12) {
                    ForEach(wallpapers) { wallpaper in
                        WallpaperCard(data: wallpaper)
                    }
                }
                .padding(.horizontal)
            }
        }
    }
}
