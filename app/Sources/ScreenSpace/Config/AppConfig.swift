import Foundation

/// Global app configuration constants
enum AppConfig {
    static let appName = "ScreenSpace"
    static let bundleIdentifier = "com.screenspace.app"

    /// Minimum macOS version supported
    static let minimumMacOSVersion = "15.0"

    /// API base URL (will be configurable for self-hosting)
    static let defaultAPIBaseURL = URL(string: "https://api.screenspace.live")!

    /// Local storage directory for downloaded wallpapers
    static var wallpaperCacheDirectory: URL {
        let appSupport = FileManager.default.urls(
            for: .applicationSupportDirectory,
            in: .userDomainMask
        ).first!
        return appSupport.appendingPathComponent(appName).appendingPathComponent("Wallpapers")
    }

    /// Configuration file location
    static var configFileURL: URL {
        let appSupport = FileManager.default.urls(
            for: .applicationSupportDirectory,
            in: .userDomainMask
        ).first!
        return appSupport.appendingPathComponent(appName).appendingPathComponent("config.json")
    }
}
