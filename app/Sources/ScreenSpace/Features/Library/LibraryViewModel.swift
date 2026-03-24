import Foundation

@Observable
@MainActor
final class LibraryViewModel {
    private let fileSystem: FileSystemProviding
    private let wallpaperProvider: WallpaperProviding
    private let eventLog: EventLogging

    var localVideos: [URL] = []
    var isDragOver = false
    var importError: String?
    var currentWallpaperURL: URL?

    init(fileSystem: FileSystemProviding, wallpaperProvider: WallpaperProviding, eventLog: EventLogging) {
        self.fileSystem = fileSystem
        self.wallpaperProvider = wallpaperProvider
        self.eventLog = eventLog
    }

    func loadLibrary() {
        localVideos = VideoImporter.listLocalVideos()
    }

    func setWallpaper(url: URL) {
        try? wallpaperProvider.setWallpaper(url: url, forDisplay: "built-in")
        currentWallpaperURL = url
        eventLog.log("wallpaper_set", data: ["source": "local", "file": url.lastPathComponent])
    }

    func removeVideo(url: URL) {
        try? fileSystem.remove(at: url)
        localVideos.removeAll { $0 == url }
    }

    func handleDroppedURLs(_ urls: [URL]) {
        for url in urls {
            guard VideoImporter.isValidVideo(url: url) else { continue }
            do {
                let imported = try VideoImporter.importVideo(from: url, to: VideoImporter.libraryDirectory())
                localVideos.append(imported)
                eventLog.log("wallpaper_cached", data: ["source": "local_import", "file": url.lastPathComponent])
            } catch {
                importError = "Failed to import \(url.lastPathComponent): \(error.localizedDescription)"
            }
        }
    }
}
