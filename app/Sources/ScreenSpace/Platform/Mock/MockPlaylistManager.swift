import Foundation

@MainActor
final class MockPlaylistManager: PlaylistManaging {
    var existingPlaylists: [Playlist] = []
    var createdNames: [String] = []
    var updatedPlaylists: [Playlist] = []
    var deletedIDs: [String] = []
    var shouldThrow: Error?

    nonisolated var playlists: [Playlist] {
        MainActor.assumeIsolated { existingPlaylists }
    }

    nonisolated func create(name: String) throws -> Playlist {
        MainActor.assumeIsolated {
            if let error = shouldThrow { fatalError("MockPlaylistManager: \(error)") }
            createdNames.append(name)
            let playlist = Playlist.create(name: name)
            existingPlaylists.append(playlist)
            return playlist
        }
    }

    nonisolated func update(_ playlist: Playlist) throws {
        MainActor.assumeIsolated {
            if let error = shouldThrow { fatalError("MockPlaylistManager: \(error)") }
            updatedPlaylists.append(playlist)
            if let idx = existingPlaylists.firstIndex(where: { $0.id == playlist.id }) {
                existingPlaylists[idx] = playlist
            }
        }
    }

    nonisolated func delete(id: String) throws {
        MainActor.assumeIsolated {
            if let error = shouldThrow { fatalError("MockPlaylistManager: \(error)") }
            deletedIDs.append(id)
            existingPlaylists.removeAll { $0.id == id }
        }
    }
}
