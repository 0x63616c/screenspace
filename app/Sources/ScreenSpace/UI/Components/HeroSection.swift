import SwiftUI

struct HeroSection: View {
    let wallpaper: WallpaperCardData?

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            // Background
            RoundedRectangle(cornerRadius: 16)
                .fill(
                    LinearGradient(
                        colors: [.blue.opacity(0.3), .purple.opacity(0.3)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
                .frame(height: 300)

            // Content overlay
            GlassCard {
                VStack(alignment: .leading, spacing: 8) {
                    Text("FEATURED")
                        .font(.caption)
                        .fontWeight(.bold)
                        .foregroundStyle(.secondary)

                    Text(wallpaper?.title ?? "No wallpapers yet")
                        .font(.title)
                        .fontWeight(.bold)

                    if let wp = wallpaper {
                        HStack(spacing: 8) {
                            ResolutionBadge(width: wp.width, height: wp.height)
                            Text(wp.durationLabel)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }

                    Button("View Wallpaper") { }
                        .buttonStyle(.bordered)
                }
                .padding()
            }
            .padding()
        }
        .padding(.horizontal)
    }
}
