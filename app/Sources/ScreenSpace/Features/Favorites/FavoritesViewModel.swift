import Foundation

@Observable
@MainActor
final class FavoritesViewModel {
    private let api: APIProviding
    private let eventLog: EventLogging

    var favorites: [WallpaperCardData] = []
    var isLoading = true
    var selectedDetail: WallpaperDetail?
    var error: String?

    private var currentOffset = 0
    private let pageSize = 20
    var hasMore = false

    init(api: APIProviding, eventLog: EventLogging) {
        self.api = api
        self.eventLog = eventLog
    }

    func load() async {
        guard !Task.isCancelled else { return }
        isLoading = true
        currentOffset = 0
        do {
            let response = try await api.listFavorites(limit: pageSize, offset: 0)
            favorites = response.items
            hasMore = response.offset + response.items.count < response.total
            eventLog.log("favorites_loaded", data: ["count": "\(favorites.count)"])
        } catch {
            self.error = "Failed to load favorites."
        }
        isLoading = false
    }

    func loadMore() async {
        guard hasMore, !Task.isCancelled else { return }
        currentOffset += pageSize
        do {
            let response = try await api.listFavorites(limit: pageSize, offset: currentOffset)
            favorites.append(contentsOf: response.items)
            hasMore = currentOffset + response.items.count < response.total
        } catch {
            self.error = "Failed to load more favorites."
        }
    }

    func fetchDetail(id: String) async {
        do {
            selectedDetail = try await api.getWallpaper(id: id)
        } catch {
            self.error = "Failed to load wallpaper details."
        }
    }
}
