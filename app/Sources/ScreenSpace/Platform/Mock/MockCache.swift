import Foundation

@MainActor
final class MockCache: CacheProviding {
    var cachedURLs: [String: URL] = [:]
    var currentSize: Int = 0
    var clearCacheCalled = false
    var lastCachedWallpaperID: String?

    nonisolated func cachedURL(for wallpaperID: String) -> URL? {
        MainActor.assumeIsolated { cachedURLs[wallpaperID] }
    }

    nonisolated func cacheFile(from sourceURL: URL, wallpaperID: String) throws -> URL {
        MainActor.assumeIsolated {
            lastCachedWallpaperID = wallpaperID
            let url = sourceURL
            cachedURLs[wallpaperID] = url
            return url
        }
    }

    nonisolated func currentSizeMB() -> Int {
        MainActor.assumeIsolated { currentSize }
    }

    nonisolated func clearCache() {
        MainActor.assumeIsolated {
            clearCacheCalled = true
            cachedURLs.removeAll()
            currentSize = 0
        }
    }
}
