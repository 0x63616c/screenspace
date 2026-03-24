import Foundation

@Observable
@MainActor
final class ExploreViewModel {
    private let api: APIProviding
    private let eventLog: EventLogging

    var categories: [Category] = []
    var selectedCategory: Category?
    var searchQuery = ""
    var results: [WallpaperCardData] = []
    var isLoading = false
    var selectedDetail: WallpaperDetail?
    var error: String?

    init(api: APIProviding, eventLog: EventLogging) {
        self.api = api
        self.eventLog = eventLog
    }

    func loadCategories() async {
        do {
            categories = try await api.listCategories()
        } catch {
            categories = Category.allCases
        }
    }

    func selectCategory(_ category: Category) async {
        guard !Task.isCancelled else { return }
        selectedCategory = category
        searchQuery = ""
        isLoading = true
        defer { isLoading = false }
        do {
            let response = try await api.listWallpapers(category: category, query: nil, sort: .recent, limit: 20, offset: 0)
            results = response.items
            eventLog.log("category_browsed", data: ["category": category.rawValue, "count": "\(results.count)"])
        } catch {
            results = []
            self.error = "Failed to load wallpapers."
        }
    }

    func search() async {
        guard !searchQuery.isEmpty, !Task.isCancelled else { return }
        selectedCategory = nil
        isLoading = true
        defer { isLoading = false }
        do {
            let response = try await api.listWallpapers(category: nil, query: searchQuery, sort: .recent, limit: 20, offset: 0)
            results = response.items
            eventLog.log("search_performed", data: ["query": searchQuery, "count": "\(results.count)"])
        } catch {
            results = []
            self.error = "Search failed."
        }
    }

    func clearCategory() {
        selectedCategory = nil
        results = []
    }

    func fetchDetail(id: String) async {
        do {
            selectedDetail = try await api.getWallpaper(id: id)
        } catch {
            self.error = "Failed to load wallpaper details."
        }
    }
}
