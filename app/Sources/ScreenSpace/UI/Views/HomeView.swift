import SwiftUI

struct HomeView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: HomeViewModel?

    var body: some View {
        Group {
            if let viewModel {
                HomeContentView(viewModel: viewModel, appState: appState)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = HomeViewModel(api: appState.apiService, eventLog: appState.eventLog)
            }
        }
    }
}

private struct HomeContentView: View {
    @Bindable var viewModel: HomeViewModel
    let appState: AppState

    var body: some View {
        ScrollView {
            if viewModel.isLoading {
                VStack {
                    Spacer(minLength: 100)
                    ProgressView("Loading wallpapers...")
                    Spacer(minLength: 100)
                }
                .frame(maxWidth: .infinity)
            } else {
                VStack(alignment: .leading, spacing: Spacing.xxl) {
                    if let error = viewModel.error {
                        VStack(spacing: Spacing.md) {
                            Text(error)
                                .font(Typography.meta)
                                .foregroundStyle(.secondary)
                                .multilineTextAlignment(.center)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.horizontal, Spacing.xl)
                        .padding(.top, Spacing.xxl)
                    }

                    if !viewModel.popular.isEmpty || !viewModel.recent.isEmpty {
                        HeroSection(
                            wallpaper: viewModel.featured,
                            onViewWallpaper: {
                                if let f = viewModel.featured {
                                    Task { await viewModel.fetchDetail(id: f.id) }
                                }
                            },
                            onFavorite: {
                                if let f = viewModel.featured {
                                    Task { await viewModel.toggleFavorite(id: f.id, isLoggedIn: appState.isLoggedIn) }
                                }
                            }
                        )
                        .padding(.horizontal, Spacing.xl)

                        ShelfRow(title: "Popular", wallpapers: viewModel.popular, onSelectWallpaper: { data in
                            Task { await viewModel.fetchDetail(id: data.id) }
                        })
                        ShelfRow(title: "Recently Added", wallpapers: viewModel.recent, onSelectWallpaper: { data in
                            Task { await viewModel.fetchDetail(id: data.id) }
                        })
                    }
                }
                .padding(.vertical, Spacing.xl)
            }
        }
        .scrollContentBackground(.hidden)
        .task { await viewModel.load() }
        .sheet(item: $viewModel.selectedDetail) { detail in
            DetailView(wallpaper: detail)
        }
    }
}
