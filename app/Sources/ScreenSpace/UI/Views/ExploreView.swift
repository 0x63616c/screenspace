import SwiftUI

struct ExploreView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: ExploreViewModel?

    var body: some View {
        Group {
            if let viewModel {
                ExploreContentView(viewModel: viewModel)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = ExploreViewModel(api: appState.apiService, eventLog: appState.eventLog)
            }
        }
    }
}

private struct ExploreContentView: View {
    @Bindable var viewModel: ExploreViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                // Search
                HStack {
                    Image(systemName: "magnifyingglass")
                    TextField("Search wallpapers", text: $viewModel.searchQuery)
                        .textFieldStyle(.plain)
                        .onSubmit { Task { await viewModel.search() } }
                        .accessibilityLabel("Search wallpapers")
                        .accessibilityHint("Type to search community wallpapers")
                }
                .padding(Spacing.sm)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .padding(.horizontal)

                // Categories grid
                if viewModel.selectedCategory == nil && viewModel.results.isEmpty {
                    Text("Categories")
                        .font(Typography.sectionTitle)
                        .padding(.horizontal)
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 150))], spacing: Spacing.md) {
                        ForEach(viewModel.categories, id: \.self) { category in
                            Button(action: { Task { await viewModel.selectCategory(category) } }) {
                                Text(category.rawValue.capitalized)
                                    .font(Typography.label)
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 80)
                                    .background(.quaternary)
                                    .clipShape(RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel("\(category.rawValue.capitalized) category")
                            .accessibilityHint("Browse \(category.rawValue) wallpapers")
                        }
                    }
                    .padding(.horizontal)
                }

                // Results header with back button
                if let category = viewModel.selectedCategory {
                    HStack {
                        Button(action: { viewModel.clearCategory() }) {
                            Image(systemName: "chevron.left")
                        }
                        .accessibilityLabel("Back to categories")
                        .accessibilityHint("Clears current category selection")
                        Text(category.rawValue.capitalized)
                            .font(Typography.sectionTitle)
                    }
                    .padding(.horizontal)
                }

                if viewModel.isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding(.top, 40)
                } else if !viewModel.results.isEmpty {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: Spacing.md) {
                        ForEach(viewModel.results) { wp in
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
        .task { await viewModel.loadCategories() }
        .sheet(item: $viewModel.selectedDetail) { detail in
            DetailView(wallpaper: detail)
        }
    }
}
