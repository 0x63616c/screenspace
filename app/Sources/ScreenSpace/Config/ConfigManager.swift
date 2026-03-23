import Foundation

/// Manages persistent user configuration
@MainActor
final class ConfigManager {
    static let shared = ConfigManager()

    private var config: UserConfig

    private init() {
        config = Self.loadConfig() ?? UserConfig()
    }

    // MARK: - Public API

    var apiBaseURL: URL {
        get { config.apiBaseURL ?? AppConfig.defaultAPIBaseURL }
        set {
            config.apiBaseURL = newValue
            save()
        }
    }

    var autoplayEnabled: Bool {
        get { config.autoplayEnabled }
        set {
            config.autoplayEnabled = newValue
            save()
        }
    }

    var launchAtLogin: Bool {
        get { config.launchAtLogin }
        set {
            config.launchAtLogin = newValue
            save()
        }
    }

    // MARK: - Persistence

    private func save() {
        do {
            let dir = AppConfig.configFileURL.deletingLastPathComponent()
            try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
            let data = try JSONEncoder().encode(config)
            try data.write(to: AppConfig.configFileURL)
        } catch {
            print("Failed to save config: \(error)")
        }
    }

    private static func loadConfig() -> UserConfig? {
        guard let data = try? Data(contentsOf: AppConfig.configFileURL) else { return nil }
        return try? JSONDecoder().decode(UserConfig.self, from: data)
    }
}

// MARK: - Config Model

struct UserConfig: Codable {
    var apiBaseURL: URL?
    var autoplayEnabled: Bool = true
    var launchAtLogin: Bool = false
}
