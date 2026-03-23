import SwiftUI

extension WallpaperResponse {
    func toCardData() -> WallpaperCardData {
        WallpaperCardData(
            id: id,
            title: title,
            thumbnailURL: thumbnailURL.flatMap { URL(string: $0) },
            width: width,
            height: height,
            duration: duration
        )
    }
}

struct HomeView: View {
    @Environment(AppState.self) var appState
    @State private var popular: [WallpaperCardData] = []
    @State private var recent: [WallpaperCardData] = []
    @State private var featured: WallpaperCardData?
    @State private var isLoading = true
    @State private var loadError: String?
    @State private var selectedWallpaper: WallpaperResponse?

    var body: some View {
        ScrollView {
            if isLoading {
                VStack {
                    Spacer(minLength: 100)
                    ProgressView("Loading wallpapers...")
                    Spacer(minLength: 100)
                }
                .frame(maxWidth: .infinity)
            } else {
                VStack(alignment: .leading, spacing: Spacing.xxl) {
                    if let error = loadError {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, Spacing.xl)
                    }

                    HeroSection(
                        wallpaper: featured,
                        onViewWallpaper: {
                            if let f = featured { fetchAndShow(id: f.id) }
                        },
                        onFavorite: {
                            guard let f = featured, appState.isLoggedIn else { return }
                            Task { _ = try? await appState.api.toggleFavorite(id: f.id) }
                        }
                    )
                        .padding(.horizontal, Spacing.xl)

                    ShelfRow(title: "Popular", wallpapers: popular, onSelectWallpaper: { data in
                        fetchAndShow(id: data.id)
                    })
                    ShelfRow(title: "Recently Added", wallpapers: recent, onSelectWallpaper: { data in
                        fetchAndShow(id: data.id)
                    })
                }
                .padding(.vertical, Spacing.xl)
            }
        }
        .scrollContentBackground(.hidden)
        .task {
            do {
                let pop = try await appState.api.popularWallpapers(limit: 10)
                let rec = try await appState.api.recentWallpapers(limit: 10)
                popular = pop.wallpapers.map { $0.toCardData() }
                recent = rec.wallpapers.map { $0.toCardData() }
                featured = popular.first
            } catch {
                loadError = "Connect to a server in Settings to browse community wallpapers."
                popular = Self.placeholderData
                recent = Self.placeholderData
                featured = Self.placeholderData.first
            }
            isLoading = false
        }
        .sheet(item: $selectedWallpaper) { wp in
            DetailView(wallpaper: wp)
        }
    }

    private func fetchAndShow(id: String) {
        Task {
            selectedWallpaper = try? await appState.api.getWallpaper(id: id)
        }
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
