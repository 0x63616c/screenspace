import Foundation
import ServiceManagement

@Observable
@MainActor
final class SettingsViewModel {
    private let configStore: ConfigStoring
    private let cache: CacheProviding
    private let eventLog: EventLogging
    private let playlistManager: any PlaylistManaging
    private let onLogout: () -> Void

    var config: AppConfig
    var cacheSize = 0
    var serverURL: String
    var playlists: [Playlist] = []
    var error: String?

    init(
        configStore: ConfigStoring,
        cache: CacheProviding,
        eventLog: EventLogging,
        playlistManager: any PlaylistManaging,
        onLogout: @escaping () -> Void
    ) {
        self.configStore = configStore
        self.cache = cache
        self.eventLog = eventLog
        self.playlistManager = playlistManager
        self.onLogout = onLogout
        let loaded = configStore.load()
        config = loaded
        serverURL = loaded.serverURL
        cacheSize = cache.currentSizeMB()
        playlists = playlistManager.playlists
    }

    func logout() {
        onLogout()
        eventLog.log("logged_out", data: [:])
    }

    func setLaunchAtLogin(_ enabled: Bool) {
        config.launchAtLogin = enabled
        do {
            if enabled {
                try SMAppService.mainApp.register()
            } else {
                try SMAppService.mainApp.unregister()
            }
            save()
        } catch {
            self.error = "Failed to \(enabled ? "enable" : "disable") launch at login: \(error.localizedDescription)"
        }
    }

    func commitServerURL() {
        config.serverURL = serverURL
        save()
    }

    func clearCache() {
        cache.clearCache()
        cacheSize = 0
        eventLog.log("cache_evicted", data: [:])
    }

    func updateConfig(_ update: (inout AppConfig) -> Void) {
        update(&config)
        save()
        eventLog.log("config_changed", data: [:])
    }

    func formatSize(_ mb: Int) -> String {
        if mb >= 1024 {
            return String(format: "%.1f GB", Double(mb) / 1024.0)
        }
        return "\(mb) MB"
    }

    private func save() {
        try? configStore.save(config)
    }
}
