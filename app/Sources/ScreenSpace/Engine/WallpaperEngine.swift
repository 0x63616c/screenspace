import AppKit
import AVFoundation

@MainActor
final class WallpaperEngine {
    private var windows: [String: WallpaperWindow] = [:]
    private let configManager: ConfigManager

    init(configManager: ConfigManager = .shared) {
        self.configManager = configManager
        observeScreenChanges()
    }

    func start() {
        for screen in NSScreen.screens {
            let stableID = DisplayIdentifier.stableID(for: screen)
            let window = WallpaperWindow(screen: screen)
            windows[stableID] = window
        }
    }

    func setWallpaper(url: URL, forDisplay stableID: String) {
        guard let window = windows[stableID] else { return }
        let gravity = configManager.config.videoGravity.avLayerGravity
        window.play(url: url, gravity: gravity)
        try? configManager.update { $0.lastPlayedURL = url.absoluteString }
    }

    func setWallpaperOnAllDisplays(url: URL) {
        let gravity = configManager.config.videoGravity.avLayerGravity
        for window in windows.values {
            window.play(url: url, gravity: gravity)
        }
        try? configManager.update { $0.lastPlayedURL = url.absoluteString }
    }

    func pauseAll() { windows.values.forEach { $0.pause() } }
    func resumeAll() { windows.values.forEach { $0.resume() } }

    func stopAll() {
        windows.values.forEach { $0.stop() }
        windows.removeAll()
    }

    var displayIDs: [String] {
        NSScreen.screens.map { DisplayIdentifier.stableID(for: $0) }
    }

    private func observeScreenChanges() {
        NotificationCenter.default.addObserver(
            self, selector: #selector(screenParametersChanged),
            name: NSApplication.didChangeScreenParametersNotification, object: nil
        )
    }

    @objc private func screenParametersChanged() {
        let currentScreenIDs = Set(NSScreen.screens.map { DisplayIdentifier.stableID(for: $0) })
        let existingIDs = Set(windows.keys)

        for id in existingIDs.subtracting(currentScreenIDs) {
            windows[id]?.stop()
            windows.removeValue(forKey: id)
        }

        for screen in NSScreen.screens {
            let id = DisplayIdentifier.stableID(for: screen)
            if windows[id] == nil {
                windows[id] = WallpaperWindow(screen: screen)
            } else {
                windows[id]?.updateFrame(to: screen)
            }
        }
    }

    deinit { NotificationCenter.default.removeObserver(self) }
}
