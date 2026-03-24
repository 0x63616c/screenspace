import Foundation

final class LiveCache: CacheProviding {
    private let manager: CacheManager

    @MainActor
    init(manager: CacheManager = .shared) {
        self.manager = manager
    }

    func cachedURL(for wallpaperID: String) -> URL? {
        manager.cachedURL(for: wallpaperID)
    }

    func cacheFile(from sourceURL: URL, wallpaperID: String) throws -> URL {
        try manager.cacheFile(from: sourceURL, wallpaperID: wallpaperID)
    }

    func currentSizeMB() -> Int {
        manager.currentCacheSizeMB()
    }

    func clearCache() {
        try? manager.clearCache()
    }
}
