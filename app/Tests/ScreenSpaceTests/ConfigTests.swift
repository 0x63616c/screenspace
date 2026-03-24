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
    }

    func testConfigRoundTrip() throws {
        let config = AppConfig.default
        let data = try JSONEncoder().encode(config)
        let decoded = try JSONDecoder().decode(AppConfig.self, from: data)
        XCTAssertEqual(config, decoded)
    }

    func testConfigManagerWithTempDir() async throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: tmpDir) }

        let manager = ConfigManager(directory: tmpDir)
        let config = await manager.config
        XCTAssertEqual(config.version, 1)

        try await manager.update { $0.pauseOnBattery = false }
        let updated = await manager.config
        XCTAssertFalse(updated.pauseOnBattery)

        let manager2 = ConfigManager(directory: tmpDir)
        let reloaded = await manager2.config
        XCTAssertFalse(reloaded.pauseOnBattery)
    }

    @MainActor
    func testPlaylistManagerCRUD() throws {
        let tmpDir = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: tmpDir) }

        let manager = PlaylistManager(directory: tmpDir)
        let initial = manager.playlists
        XCTAssertTrue(initial.isEmpty)

        let playlist = try manager.create(name: "Test")
        let afterCreate = manager.playlists
        XCTAssertEqual(afterCreate.count, 1)
        XCTAssertEqual(playlist.name, "Test")

        var updated = playlist
        updated.name = "Updated"
        try manager.update(updated)
        let afterUpdate = manager.playlists
        XCTAssertEqual(afterUpdate.first?.name, "Updated")

        try manager.delete(id: playlist.id)
        let afterDelete = manager.playlists
        XCTAssertTrue(afterDelete.isEmpty)
    }
}
