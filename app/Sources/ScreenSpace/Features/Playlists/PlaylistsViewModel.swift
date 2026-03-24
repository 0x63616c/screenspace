import Foundation

@Observable
@MainActor
final class PlaylistsViewModel {
    private let playlistManager: PlaylistManaging
    private let eventLog: EventLogging

    var playlists: [Playlist] = []
    var newPlaylistName = ""
    var error: String?

    init(playlistManager: PlaylistManaging, eventLog: EventLogging) {
        self.playlistManager = playlistManager
        self.eventLog = eventLog
    }

    func load() {
        playlists = playlistManager.playlists
    }

    func create() {
        let name = newPlaylistName.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else { return }
        do {
            let playlist = try playlistManager.create(name: name)
            playlists.append(playlist)
            newPlaylistName = ""
            eventLog.log("playlist_advanced", data: ["action": "created", "name": name])
        } catch {
            self.error = "Failed to create playlist."
        }
    }

    func updateShuffle(_ playlist: Playlist, enabled: Bool) {
        var updated = playlist
        updated.shuffle = enabled
        do {
            try playlistManager.update(updated)
            playlists = playlistManager.playlists
        } catch {
            self.error = "Failed to update playlist."
        }
    }

    func delete(playlist: Playlist) {
        do {
            try playlistManager.delete(id: playlist.id)
            playlists = playlistManager.playlists
            eventLog.log("playlist_advanced", data: ["action": "deleted", "id": playlist.id])
        } catch {
            self.error = "Failed to delete playlist."
        }
    }
}
