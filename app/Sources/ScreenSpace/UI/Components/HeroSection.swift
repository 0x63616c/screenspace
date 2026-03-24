import SwiftUI

struct HeroSection: View {
    let wallpaper: WallpaperCardData?
    var onViewWallpaper: (() -> Void)?
    var onFavorite: (() -> Void)?

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            // Thumbnail background with gradient fallback
            if let thumbnailURL = wallpaper?.thumbnailURL {
                AsyncImage(url: thumbnailURL) { phase in
                    switch phase {
                    case let .success(image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fill)
                            .frame(height: 340)
                            .clipped()
                    default:
                        heroGradientFallback
                    }
                }
                .clipShape(RoundedRectangle(cornerRadius: 20))
                .frame(height: 340)
            } else {
                heroGradientFallback
            }

            // Scrim so text is readable over bright thumbnails
            LinearGradient(
                colors: [.clear, .black.opacity(0.7)],
                startPoint: .center,
                endPoint: .bottom
            )
            .clipShape(RoundedRectangle(cornerRadius: 20))
            .frame(height: 340)

            // Content overlay - glass card floating at bottom
            VStack(alignment: .leading, spacing: Spacing.md) {
                Text("FEATURED")
                    .font(Typography.meta)
                    .fontWeight(.heavy)
                    .tracking(1.5)
                    .foregroundStyle(.white.opacity(0.6))

                Text(wallpaper?.title ?? "No wallpapers yet")
                    .font(Typography.sectionTitle)
                    .foregroundStyle(.white)
                    .lineLimit(2)

                if let wp = wallpaper {
                    HStack(spacing: Spacing.md) {
                        ResolutionBadge(width: wp.width, height: wp.height)
                        Text(wp.durationLabel)
                            .font(Typography.meta)
                            .foregroundStyle(.white.opacity(0.5))
                    }
                }

                HStack(spacing: Spacing.md) {
                    Button(action: { onViewWallpaper?() }, label: {
                        Label("Set as Wallpaper", systemImage: "photo.on.rectangle")
                    })
                    .buttonStyle(.borderedProminent)
                    .controlSize(.regular)
                    .accessibilityLabel("Set featured wallpaper as desktop wallpaper")

                    Button(action: { onFavorite?() }, label: {
                        Image(systemName: "heart")
                    })
                    .buttonStyle(.bordered)
                    .controlSize(.regular)
                    .accessibilityLabel("Add featured wallpaper to favorites")
                }
                .padding(.top, Spacing.xs)
            }
            .padding(Spacing.xl)
        }
    }

    private var heroGradientFallback: some View {
        RoundedRectangle(cornerRadius: 20)
            .fill(
                LinearGradient(
                    colors: [
                        Color(red: 0.1, green: 0.15, blue: 0.3),
                        Color(red: 0.15, green: 0.1, blue: 0.25)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .frame(height: 340)
    }
}
