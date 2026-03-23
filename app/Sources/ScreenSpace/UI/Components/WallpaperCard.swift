import SwiftUI

struct WallpaperCardData: Identifiable {
    let id: String
    let title: String
    let thumbnailURL: URL?
    let width: Int
    let height: Int
    let duration: Double

    var durationLabel: String {
        let seconds = Int(duration)
        return "\(seconds)s"
    }
}

struct WallpaperCard: View {
    let data: WallpaperCardData
    @State private var isHovered = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            ZStack(alignment: .topTrailing) {
                // Thumbnail
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color.gray.opacity(0.3))
                    .aspectRatio(16/9, contentMode: .fit)
                    .overlay {
                        if let url = data.thumbnailURL {
                            AsyncImage(url: url) { image in
                                image.resizable().scaledToFill()
                            } placeholder: {
                                ProgressView()
                            }
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }

                // Resolution badge
                ResolutionBadge(width: data.width, height: data.height)
                    .padding(6)
            }
            .scaleEffect(isHovered ? 1.05 : 1.0)
            .animation(.easeInOut(duration: 0.2), value: isHovered)
            .onHover { hovering in
                isHovered = hovering
            }

            Text(data.title)
                .font(.caption)
                .lineLimit(1)

            Text(data.durationLabel)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(width: 180)
    }
}
