import XCTest
@testable import ScreenSpace

final class CacheManagerTests: XCTestCase {
    private var tempDir: URL!
    private var cacheManager: CacheManager!

    override func setUp() {
        super.setUp()
        tempDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("CacheManagerTests-\(UUID().uuidString)")
        try? FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
        cacheManager = CacheManager(cacheDir: tempDir)
    }

    override func tearDown() {
        try? FileManager.default.removeItem(at: tempDir)
        super.tearDown()
    }

    func testEvictIfNeededRemovesOldestFiles() throws {
        // Create 3 files, each ~1MB, with staggered modification dates
        let oneKB = Data(repeating: 0, count: 1024)
        let oneMB = Data(repeating: 0, count: 1024 * 1024)

        let file1 = tempDir.appendingPathComponent("old.mp4")
        let file2 = tempDir.appendingPathComponent("mid.mp4")
        let file3 = tempDir.appendingPathComponent("new.mp4")

        try oneMB.write(to: file1)
        try oneMB.write(to: file2)
        try oneMB.write(to: file3)

        // Set modification dates: file1 oldest, file3 newest
        let now = Date()
        try FileManager.default.setAttributes(
            [.modificationDate: now.addingTimeInterval(-300)], ofItemAtPath: file1.path)
        try FileManager.default.setAttributes(
            [.modificationDate: now.addingTimeInterval(-100)], ofItemAtPath: file2.path)
        try FileManager.default.setAttributes(
            [.modificationDate: now], ofItemAtPath: file3.path)

        // Total is ~3MB, evict to 2MB limit
        cacheManager.evictIfNeeded(limitMB: 2)

        // Oldest file should be removed
        XCTAssertFalse(FileManager.default.fileExists(atPath: file1.path), "Oldest file should be evicted")
        // Newer files should remain
        XCTAssertTrue(FileManager.default.fileExists(atPath: file3.path), "Newest file should remain")
    }

    func testEvictIfNeededDoesNothingUnderLimit() throws {
        let smallData = Data(repeating: 0, count: 1024)
        let file = tempDir.appendingPathComponent("small.mp4")
        try smallData.write(to: file)

        cacheManager.evictIfNeeded(limitMB: 100)

        XCTAssertTrue(FileManager.default.fileExists(atPath: file.path), "File should not be evicted when under limit")
    }

    func testCacheFileAndRetrieve() throws {
        let sourceDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("CacheSource-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: sourceDir, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: sourceDir) }

        let sourceFile = sourceDir.appendingPathComponent("test.mp4")
        try Data(repeating: 42, count: 100).write(to: sourceFile)

        let cached = try cacheManager.cacheFile(from: sourceFile, wallpaperID: "test-123")
        XCTAssertTrue(FileManager.default.fileExists(atPath: cached.path))
        XCTAssertNotNil(cacheManager.cachedURL(for: "test-123"))
        XCTAssertNil(cacheManager.cachedURL(for: "nonexistent"))
    }
}
