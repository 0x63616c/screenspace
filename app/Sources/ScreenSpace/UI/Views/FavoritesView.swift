import SwiftUI

struct FavoritesView: View {
    @Environment(AppState.self) var appState
    @State private var favorites: [WallpaperCardData] = []
    @State private var isLoading = true
    @State private var selectedWallpaper: WallpaperResponse?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Your Favorites")
                    .font(.title2).fontWeight(.bold)
                    .padding(.horizontal)

                if isLoading {
                    ProgressView().frame(maxWidth: .infinity)
                } else if favorites.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "heart")
                            .font(.title)
                            .foregroundStyle(.secondary)
                        Text("No favorites yet")
                            .foregroundStyle(.secondary)
                        Text("Tap the heart icon on any wallpaper to save it here.")
                            .font(.caption)
                            .foregroundStyle(.tertiary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 40)
                } else {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: 12) {
                        ForEach(favorites) { wp in
                            WallpaperCard(data: wp, onTap: {
                                Task { selectedWallpaper = try? await appState.api.getWallpaper(id: wp.id) }
                            })
                        }
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .task { await loadFavorites() }
        .sheet(item: $selectedWallpaper) { wp in
            DetailView(wallpaper: wp)
        }
    }

    private func loadFavorites() async {
        do {
            let response = try await appState.api.listFavorites()
            favorites = response.wallpapers.map { $0.toCardData() }
        } catch {
            favorites = []
        }
        isLoading = false
    }
}
