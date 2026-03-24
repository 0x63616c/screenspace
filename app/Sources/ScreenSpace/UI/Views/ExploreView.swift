import SwiftUI

struct ExploreView: View {
    @Environment(AppState.self) var appState
    @State private var categories: [String] = []
    @State private var selectedCategory: String?
    @State private var searchQuery = ""
    @State private var results: [WallpaperCardData] = []
    @State private var isLoading = false
    @State private var selectedWallpaper: WallpaperResponse?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                // Search
                HStack {
                    Image(systemName: "magnifyingglass")
                    TextField("Search wallpapers", text: $searchQuery)
                        .textFieldStyle(.plain)
                        .onSubmit { Task { await search() } }
                        .accessibilityLabel("Search wallpapers")
                        .accessibilityHint("Type to search community wallpapers")
                }
                .padding(Spacing.sm)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .padding(.horizontal)

                // Categories grid
                if selectedCategory == nil && results.isEmpty {
                    Text("Categories")
                        .font(Typography.sectionTitle)
                        .padding(.horizontal)
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 150))], spacing: Spacing.md) {
                        ForEach(categories, id: \.self) { category in
                            Button(action: { Task { await selectCategory(category) } }) {
                                Text(category.capitalized)
                                    .font(Typography.label)
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 80)
                                    .background(.quaternary)
                                    .clipShape(RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel("\(category.capitalized) category")
                            .accessibilityHint("Browse \(category) wallpapers")
                        }
                    }
                    .padding(.horizontal)
                }

                // Results header with back button
                if let category = selectedCategory {
                    HStack {
                        Button(action: { selectedCategory = nil
                            results = []
                        }) {
                            Image(systemName: "chevron.left")
                        }
                        .accessibilityLabel("Back to categories")
                        .accessibilityHint("Clears current category selection")
                        Text(category.capitalized)
                            .font(Typography.sectionTitle)
                    }
                    .padding(.horizontal)
                }

                if isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding(.top, 40)
                } else if !results.isEmpty {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: Spacing.md) {
                        ForEach(results) { wp in
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
        .task { await loadCategories() }
        .sheet(item: $selectedWallpaper) { wp in
            DetailView(wallpaper: wp)
        }
    }

    private func loadCategories() async {
        do {
            categories = try await appState.api.listCategories()
        } catch {
            // Fallback to known categories if API unavailable
            categories = CategoriesResponse.fallback
        }
    }

    private func selectCategory(_ category: String) async {
        selectedCategory = category
        isLoading = true
        do {
            let typed = Category(rawValue: category)
            let response = try await appState.api.listWallpapers(category: typed)
            results = response.wallpapers.map { $0.toCardData() }
        } catch {
            results = []
        }
        isLoading = false
    }

    private func search() async {
        guard !searchQuery.isEmpty else { return }
        selectedCategory = nil
        isLoading = true
        do {
            let response = try await appState.api.listWallpapers(query: searchQuery)
            results = response.wallpapers.map { $0.toCardData() }
        } catch {
            results = []
        }
        isLoading = false
    }
}
