import SwiftUI

struct HeroSection: View {
    let wallpaper: WallpaperCardData?
    var onViewWallpaper: (() -> Void)? = nil
    var onFavorite: (() -> Void)? = nil

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            // Background gradient (will be replaced with video preview)
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
                .overlay {
                    // Noise texture overlay for depth
                    RoundedRectangle(cornerRadius: 20)
                        .fill(.ultraThinMaterial.opacity(0.1))
                }

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
                    Button(action: { onViewWallpaper?() }) {
                        Label("Set as Wallpaper", systemImage: "photo.on.rectangle")
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.regular)
                    .accessibilityLabel("Set featured wallpaper as desktop wallpaper")

                    Button(action: { onFavorite?() }) {
                        Image(systemName: "heart")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)
                    .accessibilityLabel("Add featured wallpaper to favorites")
                }
                .padding(.top, Spacing.xs)
            }
            .padding(Spacing.xl)
        }
    }
}
