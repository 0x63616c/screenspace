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
    var onTap: (() -> Void)? = nil
    @State private var isHovered = false

    var body: some View {
        Button(action: { onTap?() }) {
        VStack(alignment: .leading, spacing: 6) {
            ZStack(alignment: .topTrailing) {
                RoundedRectangle(cornerRadius: 12)
                    .fill(
                        LinearGradient(
                            colors: [
                                Color.gray.opacity(0.2),
                                Color.gray.opacity(0.1),
                            ],
                            startPoint: .top,
                            endPoint: .bottom
                        )
                    )
                    .aspectRatio(16/9, contentMode: .fit)
                    .overlay {
                        if let url = data.thumbnailURL {
                            AsyncImage(url: url) { image in
                                image.resizable().scaledToFill()
                            } placeholder: {
                                ProgressView()
                                    .scaleEffect(0.5)
                            }
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        } else {
                            Image(systemName: "play.circle.fill")
                                .font(.title2)
                                .foregroundStyle(.white.opacity(0.3))
                        }
                    }

                ResolutionBadge(width: data.width, height: data.height)
                    .padding(8)
            }
            .scaleEffect(isHovered ? 1.04 : 1.0)
            .shadow(
                color: .black.opacity(isHovered ? 0.2 : 0.05),
                radius: isHovered ? 12 : 4,
                x: 0,
                y: isHovered ? 6 : 2
            )
            .animation(.easeOut(duration: 0.15), value: isHovered)
            .onHover { hovering in
                isHovered = hovering
            }

            HStack(spacing: 4) {
                Text(data.title)
                    .font(.subheadline)
                    .fontWeight(.medium)
                    .lineLimit(1)

                Spacer()

                Text(data.durationLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(width: 200)
        }
        .buttonStyle(.plain)
    }
}
