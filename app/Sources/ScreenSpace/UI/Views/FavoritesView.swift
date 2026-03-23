import SwiftUI

struct FavoritesView: View {
    @Environment(AppState.self) var appState
    @State private var favorites: [WallpaperCardData] = []
    @State private var isLoading = true
    @State private var selectedWallpaper: WallpaperResponse?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Your Favorites")
                    .font(.title2).fontWeight(.bold)
                    .padding(.horizontal)

                if isLoading {
                    ProgressView().frame(maxWidth: .infinity)
                } else if favorites.isEmpty {
                    EmptyStateView(
                        icon: "heart",
                        title: "No favorites yet",
                        subtitle: "Tap the heart icon on any wallpaper to save it here."
                    )
                } else {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: Spacing.md) {
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
