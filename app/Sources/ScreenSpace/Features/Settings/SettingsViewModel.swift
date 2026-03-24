import Foundation
import ServiceManagement

@Observable
@MainActor
final class SettingsViewModel {
    private let configStore: ConfigStoring
    private let cache: CacheProviding
    private let eventLog: EventLogging

    var config: AppConfig
    var cacheSize: Int = 0
    var serverURL: String
    var error: String?

    init(configStore: ConfigStoring, cache: CacheProviding, eventLog: EventLogging) {
        self.configStore = configStore
        self.cache = cache
        self.eventLog = eventLog
        let loaded = configStore.load()
        self.config = loaded
        self.serverURL = loaded.serverURL
        self.cacheSize = cache.currentSizeMB()
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
