import Foundation

protocol PlaylistManaging: Sendable {
    var playlists: [Playlist] { get }
    func create(name: String) throws -> Playlist
    func update(_ playlist: Playlist) throws
    func delete(id: String) throws
}
