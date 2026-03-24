import Foundation

protocol CacheProviding: Sendable {
    func cachedURL(for wallpaperID: String) -> URL?
    func cacheFile(from sourceURL: URL, wallpaperID: String) throws -> URL
    func currentSizeMB() -> Int
    func clearCache()
}
