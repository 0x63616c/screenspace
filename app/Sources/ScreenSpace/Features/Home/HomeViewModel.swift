import Foundation

@Observable
@MainActor
final class HomeViewModel {
    private let api: APIProviding
    private let eventLog: EventLogging

    var popular: [WallpaperCardData] = []
    var recent: [WallpaperCardData] = []
    var featured: WallpaperCardData?
    var isLoading = true
    var error: String?
    var selectedDetail: WallpaperDetail?

    init(api: APIProviding, eventLog: EventLogging) {
        self.api = api
        self.eventLog = eventLog
    }

    func load() async {
        guard !Task.isCancelled else { return }
        isLoading = true
        error = nil
        do {
            let pop = try await api.popularWallpapers(limit: 10, offset: 0)
            guard !Task.isCancelled else { return }
            let rec = try await api.recentWallpapers(limit: 10, offset: 0)
            popular = pop.items
            recent = rec.items
            featured = popular.first
            eventLog.log("wallpapers_loaded", data: ["popular": "\(popular.count)", "recent": "\(recent.count)"])
        } catch {
            self.error = "Community gallery unavailable. Connect to a server in Settings."
            eventLog.log("error", data: ["context": "home_load", "message": error.localizedDescription])
        }
        isLoading = false
    }

    func fetchDetail(id: String) async {
        do {
            selectedDetail = try await api.getWallpaper(id: id)
        } catch {
            self.error = "Failed to load wallpaper details."
        }
    }

    func clearDetail() {
        selectedDetail = nil
    }
}
