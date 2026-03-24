import Foundation

final class CacheManager: Sendable {
    @MainActor static let shared = CacheManager()

    private let cacheDir: URL

    init(cacheDir: URL? = nil) {
        self.cacheDir = cacheDir ?? FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            .appendingPathComponent("ScreenSpace")
            .appendingPathComponent("cache")
            .appendingPathComponent("community")
        try? FileManager.default.createDirectory(at: self.cacheDir, withIntermediateDirectories: true)
    }

    func cachedURL(for wallpaperID: String) -> URL? {
        let url = cacheDir.appendingPathComponent("\(wallpaperID).mp4")
        return FileManager.default.fileExists(atPath: url.path) ? url : nil
    }

    func cacheFile(from sourceURL: URL, wallpaperID: String, cacheSizeLimitMB: Int = AppConfig.default.cacheSizeLimitMB) throws -> URL {
        let destURL = cacheDir.appendingPathComponent("\(wallpaperID).mp4")
        if FileManager.default.fileExists(atPath: destURL.path) {
            try FileManager.default.removeItem(at: destURL)
        }
        try FileManager.default.copyItem(at: sourceURL, to: destURL)
        evictIfNeeded(limitMB: cacheSizeLimitMB)
        return destURL
    }

    func currentCacheSizeMB() -> Int {
        guard let files = try? FileManager.default.contentsOfDirectory(at: cacheDir, includingPropertiesForKeys: [.fileSizeKey]) else {
            return 0
        }
        let totalBytes = files.compactMap { url -> Int? in
            try? url.resourceValues(forKeys: [.fileSizeKey]).fileSize
        }.reduce(0, +)
        return totalBytes / (1024 * 1024)
    }

    func evictIfNeeded(limitMB: Int) {
        let currentMB = currentCacheSizeMB()
        guard currentMB > limitMB else { return }

        guard let files = try? FileManager.default.contentsOfDirectory(
            at: cacheDir,
            includingPropertiesForKeys: [.contentModificationDateKey, .fileSizeKey]
        ) else { return }

        let sorted = files.sorted { a, b in
            let dateA = (try? a.resourceValues(forKeys: [.contentModificationDateKey]).contentModificationDate) ?? .distantPast
            let dateB = (try? b.resourceValues(forKeys: [.contentModificationDateKey]).contentModificationDate) ?? .distantPast
            return dateA < dateB
        }

        var remaining = currentMB
        for file in sorted {
            guard remaining > limitMB else { break }
            let fileSize = (try? file.resourceValues(forKeys: [.fileSizeKey]).fileSize) ?? 0
            try? FileManager.default.removeItem(at: file)
            remaining -= fileSize / (1024 * 1024)
        }
    }

    func clearCache() throws {
        let files = try FileManager.default.contentsOfDirectory(at: cacheDir, includingPropertiesForKeys: nil)
        for file in files {
            try FileManager.default.removeItem(at: file)
        }
    }
}
