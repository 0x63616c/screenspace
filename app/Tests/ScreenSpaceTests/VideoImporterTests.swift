import XCTest
@testable import ScreenSpace

final class VideoImporterTests: XCTestCase {
    func testIsValidVideo_MP4() {
        let tmpDir = FileManager.default.temporaryDirectory
        let mp4 = tmpDir.appendingPathComponent("test.mp4")
        FileManager.default.createFile(atPath: mp4.path, contents: Data("fake".utf8))
        defer { try? FileManager.default.removeItem(at: mp4) }

        XCTAssertTrue(VideoImporter.isValidVideo(url: mp4))
    }

    func testIsValidVideo_MOV() {
        let tmpDir = FileManager.default.temporaryDirectory
        let mov = tmpDir.appendingPathComponent("test.mov")
        FileManager.default.createFile(atPath: mov.path, contents: Data("fake".utf8))
        defer { try? FileManager.default.removeItem(at: mov) }

        XCTAssertTrue(VideoImporter.isValidVideo(url: mov))
    }

    func testIsValidVideo_RejectsOtherFormats() {
        let tmpDir = FileManager.default.temporaryDirectory
        let txt = tmpDir.appendingPathComponent("test.txt")
        FileManager.default.createFile(atPath: txt.path, contents: Data("fake".utf8))
        defer { try? FileManager.default.removeItem(at: txt) }

        XCTAssertFalse(VideoImporter.isValidVideo(url: txt))
    }

    func testImportVideo() throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let sourceDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: sourceDir, withIntermediateDirectories: true)
        defer {
            try? FileManager.default.removeItem(at: tmpDir)
            try? FileManager.default.removeItem(at: sourceDir)
        }

        let source = sourceDir.appendingPathComponent("video.mp4")
        try Data("video content".utf8).write(to: source)

        let imported = try VideoImporter.importVideo(from: source, to: tmpDir)
        XCTAssertTrue(FileManager.default.fileExists(atPath: imported.path))
        XCTAssertEqual(imported.pathExtension, "mp4")
    }

    func testCacheManager() throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: tmpDir) }

        let cache = CacheManager(cacheDir: tmpDir)
        XCTAssertNil(cache.cachedURL(for: "test-id"))
        XCTAssertEqual(cache.currentCacheSizeMB(), 0)

        let source = FileManager.default.temporaryDirectory.appendingPathComponent("source.mp4")
        try Data(repeating: 0, count: 1024).write(to: source)
        defer { try? FileManager.default.removeItem(at: source) }

        let cached = try cache.cacheFile(from: source, wallpaperID: "test-id")
        XCTAssertTrue(FileManager.default.fileExists(atPath: cached.path))
        XCTAssertNotNil(cache.cachedURL(for: "test-id"))

        try cache.clearCache()
        XCTAssertNil(cache.cachedURL(for: "test-id"))
    }
}
