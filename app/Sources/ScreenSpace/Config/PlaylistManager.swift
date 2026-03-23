import Foundation

/// Manages wallpaper playlists and scheduling
@MainActor
final class PlaylistManager {
    static let shared = PlaylistManager()

    private var playlists: [Playlist] = []

    private init() {}

    // MARK: - Public API

    func addPlaylist(_ playlist: Playlist) {
        playlists.append(playlist)
    }

    func removePlaylist(id: UUID) {
        playlists.removeAll { $0.id == id }
    }

    func getAllPlaylists() -> [Playlist] {
        playlists
    }
}

// MARK: - Models

struct Playlist: Identifiable, Codable {
    let id: UUID
    var name: String
    var wallpaperIDs: [String]
    var shuffleEnabled: Bool
    var intervalSeconds: TimeInterval

    init(
        id: UUID = UUID(),
        name: String,
        wallpaperIDs: [String] = [],
        shuffleEnabled: Bool = false,
        intervalSeconds: TimeInterval = 3600
    ) {
        self.id = id
        self.name = name
        self.wallpaperIDs = wallpaperIDs
        self.shuffleEnabled = shuffleEnabled
        self.intervalSeconds = intervalSeconds
    }
}
