import Foundation
import Testing
@testable import ScreenSpace

@MainActor
struct AppStateTests {
    private func makeAppState() -> AppState {
        let tmpDir = FileManager.default.temporaryDirectory
            .appendingPathComponent("AppStateTests-\(UUID().uuidString)")
        let configManager = ConfigManager(directory: tmpDir)
        let playlistManager = PlaylistManager(directory: tmpDir.appendingPathComponent("playlists"))
        return AppState(
            playlistManager: playlistManager,
            configManager: configManager
        )
    }

    @Test("isLoggedIn is false initially")
    func isLoggedInFalseInitially() {
        #expect(makeAppState().isLoggedIn == false)
    }

    @Test("isAdmin is false when no user")
    func isAdminFalseNoUser() {
        #expect(makeAppState().isAdmin == false)
    }

    @Test("setWallpaper updates properties")
    func setWallpaperUpdatesProperties() async {
        let state = makeAppState()
        let url = URL(fileURLWithPath: "/tmp/test-wallpaper.mp4")
        await state.setWallpaper(url: url, title: "Test Wallpaper")
        #expect(state.currentWallpaperURL == url)
        #expect(state.currentWallpaperTitle == "Test Wallpaper")
    }

    @Test("setWallpaper defaults to filename")
    func setWallpaperDefaultsToFilename() async {
        let state = makeAppState()
        let url = URL(fileURLWithPath: "/tmp/my-video.mp4")
        await state.setWallpaper(url: url)
        #expect(state.currentWallpaperTitle == "my-video.mp4")
    }

    @Test("logout clears currentUser")
    func logoutClearsUser() {
        let state = makeAppState()
        state.currentUser = TestFixtures.userInfo()
        #expect(state.isLoggedIn == true)

        state.logout()

        #expect(state.currentUser == nil)
        #expect(state.isLoggedIn == false)
    }

    @Test("isAdmin is true for admin role")
    func isAdminTrueForAdmin() {
        let state = makeAppState()
        state.currentUser = TestFixtures.userInfo(role: .admin)
        #expect(state.isAdmin == true)
    }

    @Test("now playing callback fires")
    func nowPlayingCallbackFires() async {
        let state = makeAppState()
        var receivedTitle: String?
        state.onNowPlayingChanged = { title in
            receivedTitle = title
        }
        let url = URL(fileURLWithPath: "/tmp/callback-test.mp4")
        await state.setWallpaper(url: url, title: "Callback Test")
        #expect(receivedTitle == "Callback Test")
    }
}
