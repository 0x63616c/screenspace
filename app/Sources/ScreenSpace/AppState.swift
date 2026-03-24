import SwiftUI

@Observable
@MainActor
final class AppState {
    let apiService: APIProviding
    let wallpaperProvider: WallpaperProviding
    let fileSystem: FileSystemProviding
    let keychain: KeychainProviding
    let configStore: ConfigStoring
    let playlistManager: any PlaylistManaging
    let eventLog: EventLogging
    let cache: CacheProviding

    let engine: WallpaperEngine
    let pauseController: PauseController
    let lockScreen: LockScreenManager
    let configManager: ConfigManager

    var currentUser: UserInfo?
    var isLoggedIn: Bool {
        currentUser != nil
    }

    var isAdmin: Bool {
        currentUser?.role == .admin
    }

    var currentWallpaperURL: URL?
    var currentWallpaperTitle: String?

    var onNowPlayingChanged: ((String?) -> Void)?

    init(
        api: APIProviding? = nil,
        wallpaperProvider: WallpaperProviding? = nil,
        fileSystem: FileSystemProviding? = nil,
        keychain: KeychainProviding? = nil,
        configStore: ConfigStoring? = nil,
        playlistManager: (any PlaylistManaging)? = nil,
        eventLog: EventLogging? = nil,
        cache: CacheProviding? = nil,
        engine: WallpaperEngine? = nil,
        configManager: ConfigManager? = nil
    ) {
        let kc = keychain ?? LiveKeychain()
        let cs = configStore ?? LiveConfigStore()
        let cm = configManager ?? .shared
        let apiClient = APIClient(keychain: kc)

        self.keychain = kc
        self.configStore = cs
        self.fileSystem = fileSystem ?? LiveFileSystem()
        self.playlistManager = playlistManager ?? PlaylistManager.shared
        self.eventLog = eventLog ?? EventLog.shared
        self.cache = cache ?? LiveCache()
        self.wallpaperProvider = wallpaperProvider ?? LiveWallpaperProvider()
        apiService = api ?? APIService(client: apiClient)
        self.configManager = cm
        lockScreen = LockScreenManager()
        self.engine = engine ?? WallpaperEngine(configManager: cm)
        let config = cs.load()
        pauseController = PauseController(config: config)

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

    func setWallpaper(url: URL, title: String? = nil) async {
        await engine.setWallpaperOnAllDisplays(url: url)
        currentWallpaperURL = url
        currentWallpaperTitle = title ?? url.lastPathComponent
        onNowPlayingChanged?(currentWallpaperTitle)
    }

    func setWallpaper(url: URL, title: String? = nil, forDisplay displayID: String) async {
        await engine.setWallpaper(url: url, forDisplay: displayID)
        currentWallpaperURL = url
        currentWallpaperTitle = title ?? url.lastPathComponent
        onNowPlayingChanged?(currentWallpaperTitle)
    }

    func login(email: String, password: String) async throws {
        _ = try await apiService.login(email: email, password: password)
        currentUser = try await apiService.me()
    }

    func register(email: String, password: String) async throws {
        _ = try await apiService.register(email: email, password: password)
        currentUser = try await apiService.me()
    }

    func logout() {
        apiService.logout()
        currentUser = nil
    }

    func restoreSession() async {
        guard keychain.load(key: "auth_token") != nil else { return }
        currentUser = try? await apiService.me()
    }

    func restoreLastWallpaper() async {
        let config = configStore.load()
        guard let urlString = config.lastPlayedURL,
              let url = URL(string: urlString),
              FileManager.default.fileExists(atPath: url.path) else { return }
        await setWallpaper(url: url)
    }

    func skipToNext() async {
        let allPlaylists = playlistManager.playlists
        guard let playlist = allPlaylists.first, !playlist.items.isEmpty else { return }
        if let firstItem = playlist.items.first,
           let path = firstItem.path,
           firstItem.source == .local
        {
            let url = URL(fileURLWithPath: path)
            if FileManager.default.fileExists(atPath: url.path) {
                await setWallpaper(url: url, title: url.lastPathComponent)
            }
        }
    }
}
