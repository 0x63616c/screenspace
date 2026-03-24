import AppKit

protocol WallpaperProviding: Sendable {
    func setWallpaper(url: URL, forDisplay displayID: String) throws
    func currentWallpaper(forDisplay displayID: String) -> URL?
    func availableDisplays() -> [String]
}
