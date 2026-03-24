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
                        VStack(spacing: Spacing.md) {
                            Text(error)
                                .font(.callout)
                                .foregroundStyle(.secondary)
                                .multilineTextAlignment(.center)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.horizontal, Spacing.xl)
                        .padding(.top, Spacing.xxl)
                    }

                    if !popular.isEmpty || !recent.isEmpty {
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
                loadError = "Community gallery unavailable. Connect to a server in Settings."
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

}
