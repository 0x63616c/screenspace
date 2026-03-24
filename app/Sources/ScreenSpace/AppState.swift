import SwiftUI

@Observable
@MainActor
final class AppState {
    let engine: WallpaperEngine
    let api: APIClient
    let configManager: ConfigManager
    let playlistManager: PlaylistManager
    let lockScreen: LockScreenManager
    let pauseController: PauseController

    var currentUser: UserResponse?
    var isLoggedIn: Bool { currentUser != nil }
    var isAdmin: Bool { currentUser?.role == .admin }
    var currentWallpaperURL: URL?
    var currentWallpaperTitle: String?

    /// Called when the now-playing title changes so the menu bar can update.
    var onNowPlayingChanged: ((String?) -> Void)?

    init(
        engine: WallpaperEngine? = nil,
        api: APIClient? = nil,
        configManager: ConfigManager? = nil,
        playlistManager: PlaylistManager? = nil
    ) {
        let cm = configManager ?? .shared
        self.configManager = cm
        self.playlistManager = playlistManager ?? .shared
        self.api = api ?? APIClient()
        self.lockScreen = LockScreenManager()
        self.engine = engine ?? WallpaperEngine(configManager: cm)
        self.pauseController = PauseController(config: cm.config)

        // Wire PauseController to Engine via @Observable observation
        Task { @MainActor [weak self] in
            while let self {
                let shouldPause = self.pauseController.shouldPause
                if shouldPause {
                    self.engine.pauseAll()
                } else {
                    self.engine.resumeAll()
                }
                await withCheckedContinuation { continuation in
                    withObservationTracking {
                        _ = self.pauseController.shouldPause
                    } onChange: {
                        continuation.resume()
                    }
                }
            }
        }
    }

    func setWallpaper(url: URL, title: String? = nil) {
        engine.setWallpaperOnAllDisplays(url: url)
        currentWallpaperURL = url
        currentWallpaperTitle = title ?? url.lastPathComponent
        onNowPlayingChanged?(currentWallpaperTitle)
    }

    func setWallpaper(url: URL, title: String? = nil, forDisplay displayID: String) {
        engine.setWallpaper(url: url, forDisplay: displayID)
        currentWallpaperURL = url
        currentWallpaperTitle = title ?? url.lastPathComponent
        onNowPlayingChanged?(currentWallpaperTitle)
    }

    func login(email: String, password: String) async throws {
        _ = try await api.login(email: email, password: password)
        currentUser = try await api.me()
    }

    func register(email: String, password: String) async throws {
        _ = try await api.register(email: email, password: password)
        currentUser = try await api.me()
    }

    func logout() {
        api.logout()
        currentUser = nil
    }

    func restoreSession() async {
        guard KeychainHelper.loadToken() != nil else { return }
        currentUser = try? await api.me()
    }

    func restoreLastWallpaper() {
        guard let urlString = configManager.config.lastPlayedURL,
              let url = URL(string: urlString),
              FileManager.default.fileExists(atPath: url.path) else { return }
        setWallpaper(url: url)
    }

    func skipToNext() {
        let playlists = playlistManager.playlists
        guard let playlist = playlists.first, !playlist.items.isEmpty else { return }
        if let firstItem = playlist.items.first,
           let path = firstItem.path,
           firstItem.source == .local {
            let url = URL(fileURLWithPath: path)
            if FileManager.default.fileExists(atPath: url.path) {
                setWallpaper(url: url, title: url.lastPathComponent)
            }
        }
    }
}
