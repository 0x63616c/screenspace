import AppKit

struct LiveWallpaperProvider: WallpaperProviding {
    func setWallpaper(url: URL, forDisplay displayID: String) throws {
        guard let screen = NSScreen.screens.first(where: {
            DisplayIdentifier.stableID(for: $0) == displayID
        }) else { return }
        try NSWorkspace.shared.setDesktopImageURL(url, for: screen, options: [:])
    }

    func currentWallpaper(forDisplay displayID: String) -> URL? {
        guard let screen = NSScreen.screens.first(where: {
            DisplayIdentifier.stableID(for: $0) == displayID
        }) else { return nil }
        return NSWorkspace.shared.desktopImageURL(for: screen)
    }

    func availableDisplays() -> [String] {
        NSScreen.screens.map { DisplayIdentifier.stableID(for: $0) }
    }
}
