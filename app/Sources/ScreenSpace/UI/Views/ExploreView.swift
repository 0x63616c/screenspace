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
            VStack(alignment: .leading, spacing: 16) {
                // Search
                HStack {
                    Image(systemName: "magnifyingglass")
                    TextField("Search wallpapers", text: $searchQuery)
                        .textFieldStyle(.plain)
                        .onSubmit { Task { await search() } }
                }
                .padding(8)
                .background(.quaternary)
                .clipShape(RoundedRectangle(cornerRadius: 8))
                .padding(.horizontal)

                // Categories grid
                if selectedCategory == nil && results.isEmpty {
                    Text("Categories")
                        .font(.title3).fontWeight(.bold)
                        .padding(.horizontal)
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 150))], spacing: 12) {
                        ForEach(categories, id: \.self) { category in
                            Button(action: { Task { await selectCategory(category) } }) {
                                Text(category.capitalized)
                                    .font(.headline)
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 80)
                                    .background(.quaternary)
                                    .clipShape(RoundedRectangle(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.horizontal)
                }

                // Results header with back button
                if let category = selectedCategory {
                    HStack {
                        Button(action: { selectedCategory = nil; results = [] }) {
                            Image(systemName: "chevron.left")
                        }
                        Text(category.capitalized)
                            .font(.title3).fontWeight(.bold)
                    }
                    .padding(.horizontal)
                }

                if isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity)
                        .padding(.top, 40)
                } else if !results.isEmpty {
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: 12) {
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
        categories = ["nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal"]
    }

    private func selectCategory(_ category: String) async {
        selectedCategory = category
        isLoading = true
        do {
            let response = try await appState.api.listWallpapers(category: category)
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
