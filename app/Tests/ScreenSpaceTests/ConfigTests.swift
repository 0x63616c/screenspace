import XCTest
@testable import ScreenSpace

final class ConfigTests: XCTestCase {
    func testDefaultConfig() {
        let config = AppConfig.default
        XCTAssertEqual(config.version, 1)
        XCTAssertTrue(config.pauseOnBattery)
        XCTAssertTrue(config.pauseOnFullscreen)
        XCTAssertEqual(config.videoGravity, .resizeAspectFill)
        XCTAssertEqual(config.cacheSizeLimitMB, 5120)
        XCTAssertEqual(config.serverURL, "https://api.screenspace.app")
    }

    func testConfigRoundTrip() throws {
        let config = AppConfig.default
        let data = try JSONEncoder().encode(config)
        let decoded = try JSONDecoder().decode(AppConfig.self, from: data)
        XCTAssertEqual(config, decoded)
    }

    func testConfigManagerWithTempDir() throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: tmpDir) }

        let manager = ConfigManager(directory: tmpDir)
        XCTAssertEqual(manager.config.version, 1)

        try manager.update { $0.pauseOnBattery = false }
        XCTAssertFalse(manager.config.pauseOnBattery)

        let manager2 = ConfigManager(directory: tmpDir)
        XCTAssertFalse(manager2.config.pauseOnBattery)
    }

    func testPlaylistManagerCRUD() throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: tmpDir) }

        let manager = PlaylistManager(directory: tmpDir)
        XCTAssertTrue(manager.playlists.isEmpty)

        let playlist = try manager.create(name: "Test")
        XCTAssertEqual(manager.playlists.count, 1)
        XCTAssertEqual(playlist.name, "Test")

        var updated = playlist
        updated.name = "Updated"
        try manager.update(updated)
        XCTAssertEqual(manager.playlists.first?.name, "Updated")

        try manager.delete(id: playlist.id)
        XCTAssertTrue(manager.playlists.isEmpty)
    }
}
