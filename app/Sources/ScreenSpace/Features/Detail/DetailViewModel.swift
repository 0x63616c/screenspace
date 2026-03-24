import Foundation

@Observable
@MainActor
final class DetailViewModel {
    private let api: APIProviding
    private let wallpaperProvider: WallpaperProviding
    private let cache: CacheProviding
    private let eventLog: EventLogging

    let wallpaper: WallpaperDetail

    var isDownloading = false
    var downloadProgress: Double = 0
    var isFavorited = false
    var showReportSheet = false
    var reportReason = ""
    var error: String?

    init(
        wallpaper: WallpaperDetail,
        api: APIProviding,
        wallpaperProvider: WallpaperProviding,
        cache: CacheProviding,
        eventLog: EventLogging
    ) {
        self.wallpaper = wallpaper
        self.api = api
        self.wallpaperProvider = wallpaperProvider
        self.cache = cache
        self.eventLog = eventLog
    }

    func setAsWallpaper() async {
        if let cachedURL = cache.cachedURL(for: wallpaper.id) {
            try? wallpaperProvider.setWallpaper(url: cachedURL, forDisplay: "built-in")
            eventLog.log("wallpaper_set", data: ["source": "cache", "id": wallpaper.id])
            return
        }
        guard let downloadURL = wallpaper.downloadURL else { return }
        isDownloading = true
        downloadProgress = 0
        do {
            let localURL = try await downloadFile(from: downloadURL)
            let cachedURL = try cache.cacheFile(from: localURL, wallpaperID: wallpaper.id)
            try? wallpaperProvider.setWallpaper(url: cachedURL, forDisplay: "built-in")
            eventLog.log("wallpaper_downloaded", data: ["id": wallpaper.id])
        } catch {
            self.error = "Download failed: \(error.localizedDescription)"
            eventLog.log("error", data: ["context": "wallpaper_download", "message": error.localizedDescription])
        }
        isDownloading = false
    }

    func toggleFavorite(isLoggedIn: Bool) async {
        guard isLoggedIn else {
            error = "Log in to favorite wallpapers."
            return
        }
        do {
            isFavorited = try await api.toggleFavorite(id: wallpaper.id)
            eventLog.log("favorite_toggled", data: ["id": wallpaper.id, "favorited": "\(isFavorited)"])
        } catch {
            self.error = "Failed to update favorite."
        }
    }

    func submitReport(isLoggedIn: Bool) async {
        guard isLoggedIn else {
            error = "Log in to report wallpapers."
            return
        }
        let reason = reportReason.trimmingCharacters(in: .whitespaces)
        guard !reason.isEmpty else { return }
        do {
            try await api.reportWallpaper(id: wallpaper.id, reason: reason)
            reportReason = ""
            showReportSheet = false
            eventLog.log("reported", data: ["id": wallpaper.id])
        } catch {
            self.error = "Failed to submit report."
        }
    }

    var formattedSize: String {
        formatFileSize(wallpaper.fileSize)
    }

    var formattedDuration: String {
        "\(Int(wallpaper.duration))s"
    }

    // MARK: - Private

    private func downloadFile(from url: URL) async throws -> URL {
        let (tempURL, _) = try await URLSession.shared.download(from: url)
        return tempURL
    }
}
