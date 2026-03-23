import XCTest
@testable import ScreenSpace

@MainActor
final class AppStateTests: XCTestCase {
    private func makeAppState() -> AppState {
        let tmpDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("AppStateTests-\(UUID().uuidString)")
        let configManager = ConfigManager(directory: tmpDir)
        let playlistManager = PlaylistManager(directory: tmpDir.appendingPathComponent("playlists"))
        return AppState(
            configManager: configManager,
            playlistManager: playlistManager
        )
    }

    func testIsLoggedInFalseInitially() {
        let state = makeAppState()
        XCTAssertFalse(state.isLoggedIn)
    }

    func testIsAdminFalseWhenNoUser() {
        let state = makeAppState()
        XCTAssertFalse(state.isAdmin)
    }

    func testSetWallpaperUpdatesProperties() {
        let state = makeAppState()
        let url = URL(fileURLWithPath: "/tmp/test-wallpaper.mp4")
        state.setWallpaper(url: url, title: "Test Wallpaper")
        XCTAssertEqual(state.currentWallpaperURL, url)
        XCTAssertEqual(state.currentWallpaperTitle, "Test Wallpaper")
    }

    func testSetWallpaperDefaultsToFilename() {
        let state = makeAppState()
        let url = URL(fileURLWithPath: "/tmp/my-video.mp4")
        state.setWallpaper(url: url)
        XCTAssertEqual(state.currentWallpaperTitle, "my-video.mp4")
    }

    func testRestoreLastWallpaperWithNonexistentFileDoesNothing() {
        let state = makeAppState()
        try? state.configManager.update { $0.lastPlayedURL = "file:///nonexistent/path.mp4" }
        state.restoreLastWallpaper()
        XCTAssertNil(state.currentWallpaperURL)
        XCTAssertNil(state.currentWallpaperTitle)
    }

    func testLogoutClearsUser() {
        let state = makeAppState()
        // Simulate a logged-in user
        state.currentUser = UserResponse(
            id: "1",
            email: "test@example.com",
            role: "user",
            banned: false,
            createdAt: nil
        )
        XCTAssertTrue(state.isLoggedIn)

        state.logout()
        XCTAssertNil(state.currentUser)
        XCTAssertFalse(state.isLoggedIn)
    }

    func testIsAdminTrueForAdminRole() {
        let state = makeAppState()
        state.currentUser = UserResponse(
            id: "1",
            email: "admin@example.com",
            role: "admin",
            banned: false,
            createdAt: nil
        )
        XCTAssertTrue(state.isAdmin)
    }

    func testNowPlayingCallbackFires() {
        let state = makeAppState()
        var receivedTitle: String?
        state.onNowPlayingChanged = { title in
            receivedTitle = title
        }
        let url = URL(fileURLWithPath: "/tmp/callback-test.mp4")
        state.setWallpaper(url: url, title: "Callback Test")
        XCTAssertEqual(receivedTitle, "Callback Test")
    }
}
