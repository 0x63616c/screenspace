import Foundation

struct PlaylistItem: Codable, Identifiable, Equatable {
    let id: String
    let source: Source
    var path: String?

    enum Source: String, Codable {
        case local
        case community
    }
}

struct Playlist: Codable, Identifiable, Equatable {
    let id: String
    var name: String
    var items: [PlaylistItem]
    var interval: Int
    var shuffle: Bool

    static func create(name: String) -> Playlist {
        Playlist(
            id: UUID().uuidString,
            name: name,
            items: [],
            interval: 0,
            shuffle: false
        )
    }
}

actor PlaylistManager {
    static let shared = PlaylistManager()

    private let playlistsDir: URL
    private(set) var playlists: [Playlist] = []

    init(directory: URL? = nil) {
        let dir: URL
        if let directory {
            dir = directory
        } else {
            guard let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)
                .first
            else {
                fatalError("Application Support directory unavailable")
            }
            dir = appSupport.appendingPathComponent("ScreenSpace").appendingPathComponent("playlists")
        }
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        playlistsDir = dir
        playlists = Self.loadAll(from: dir)
    }

    func create(name: String) throws -> Playlist {
        let playlist = Playlist.create(name: name)
        try save(playlist)
        playlists.append(playlist)
        return playlist
    }

    func delete(id: String) throws {
        let url = playlistsDir.appendingPathComponent("\(id).json")
        try? FileManager.default.removeItem(at: url)
        playlists.removeAll { $0.id == id }
    }

    func update(_ playlist: Playlist) throws {
        try save(playlist)
        if let idx = playlists.firstIndex(where: { $0.id == playlist.id }) {
            playlists[idx] = playlist
        }
    }

    private func save(_ playlist: Playlist) throws {
        let data = try JSONEncoder().encode(playlist)
        let url = playlistsDir.appendingPathComponent("\(playlist.id).json")
        try data.write(to: url, options: .atomic)
    }

    private static func loadAll(from dir: URL) -> [Playlist] {
        guard let files = try? FileManager.default.contentsOfDirectory(at: dir, includingPropertiesForKeys: nil) else {
            return []
        }
        return files.compactMap { url -> Playlist? in
            guard url.pathExtension == "json" else { return nil }
            guard let data = try? Data(contentsOf: url) else { return nil }
            return try? JSONDecoder().decode(Playlist.self, from: data)
        }
    }
}
