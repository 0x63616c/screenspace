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

    func cacheFile(from sourceURL: URL, wallpaperID: String) throws -> URL {
        let destURL = cacheDir.appendingPathComponent("\(wallpaperID).mp4")
        if FileManager.default.fileExists(atPath: destURL.path) {
            try FileManager.default.removeItem(at: destURL)
        }
        try FileManager.default.copyItem(at: sourceURL, to: destURL)
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

    func clearCache() throws {
        let files = try FileManager.default.contentsOfDirectory(at: cacheDir, includingPropertiesForKeys: nil)
        for file in files {
            try FileManager.default.removeItem(at: file)
        }
    }
}
