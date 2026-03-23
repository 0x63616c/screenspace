import SwiftUI

struct PlaylistsView: View {
    @Environment(AppState.self) var appState
    @State private var playlists: [Playlist] = []
    @State private var newPlaylistName = ""
    @State private var selectedPlaylist: Playlist?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Playlists")
                    .font(.title2).fontWeight(.bold)
                    .padding(.horizontal)

                // Create new playlist
                HStack {
                    TextField("New playlist name", text: $newPlaylistName)
                        .textFieldStyle(.roundedBorder)
                    Button("Create") {
                        guard !newPlaylistName.isEmpty else { return }
                        if let playlist = try? appState.playlistManager.create(name: newPlaylistName) {
                            playlists.append(playlist)
                            newPlaylistName = ""
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(newPlaylistName.isEmpty)
                }
                .padding(.horizontal)

                if playlists.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "music.note.list")
                            .font(.title)
                            .foregroundStyle(.secondary)
                        Text("No playlists yet")
                            .foregroundStyle(.secondary)
                        Text("Create a playlist to rotate wallpapers automatically.")
                            .font(.caption)
                            .foregroundStyle(.tertiary)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.top, 40)
                } else {
                    ForEach(playlists) { playlist in
                        playlistCard(playlist)
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .onAppear { playlists = appState.playlistManager.playlists }
    }

    private func playlistCard(_ playlist: Playlist) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                VStack(alignment: .leading) {
                    Text(playlist.name)
                        .font(.headline)
                    Text("\(playlist.items.count) items")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()

                if playlist.interval > 0 {
                    Text("Every \(playlist.interval / 60)min")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Toggle("Shuffle", isOn: Binding(
                    get: { playlist.shuffle },
                    set: { newValue in
                        var updated = playlist
                        updated.shuffle = newValue
                        try? appState.playlistManager.update(updated)
                        playlists = appState.playlistManager.playlists
                    }
                ))
                .toggleStyle(.switch)
                .labelsHidden()

                Button(role: .destructive) {
                    try? appState.playlistManager.delete(id: playlist.id)
                    playlists = appState.playlistManager.playlists
                } label: {
                    Image(systemName: "trash")
                }
                .buttonStyle(.bordered)
            }
        }
        .padding()
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 12))
    }
}
