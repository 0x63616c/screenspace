import SwiftUI

struct FavoritesView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: FavoritesViewModel?

    var body: some View {
        Group {
            if let viewModel {
                FavoritesContentView(viewModel: viewModel)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = FavoritesViewModel(api: appState.apiService, eventLog: appState.eventLog)
            }
        }
    }
}

private struct FavoritesContentView: View {
    @Bindable var viewModel: FavoritesViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Your Favorites")
                    .font(Typography.pageTitle)
                    .padding(.horizontal)

                if viewModel.isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .accessibilityLabel("Loading favorites")
                } else if viewModel.favorites.isEmpty {
                    EmptyStateView(
                        icon: "heart",
                        title: "No favorites yet",
                        subtitle: "Tap the heart icon on any wallpaper to save it here."
                    )
                } else {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: Spacing.md) {
                        ForEach(viewModel.favorites) { wp in
                            WallpaperCard(data: wp, onTap: {
                                Task { await viewModel.fetchDetail(id: wp.id) }
                            })
                        }
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .task { await viewModel.load() }
        .sheet(item: $viewModel.selectedDetail) { detail in
            DetailView(wallpaper: detail)
        }
    }
}
