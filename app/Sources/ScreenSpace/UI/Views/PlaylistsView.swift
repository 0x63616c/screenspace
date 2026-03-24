import SwiftUI

struct PlaylistsView: View {
    @Environment(AppState.self) var appState
    @State private var playlists: [Playlist] = []
    @State private var newPlaylistName = ""
    @State private var selectedPlaylist: Playlist?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Playlists")
                    .font(Typography.pageTitle)
                    .padding(.horizontal)

                // Create new playlist
                HStack {
                    TextField("New playlist name", text: $newPlaylistName)
                        .textFieldStyle(.roundedBorder)
                        .accessibilityLabel("New playlist name")
                    Button("Create") {
                        guard !newPlaylistName.isEmpty else { return }
                        Task {
                            if let playlist = try? await appState.playlistManager.create(name: newPlaylistName) {
                                playlists.append(playlist)
                                newPlaylistName = ""
                            }
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(newPlaylistName.isEmpty)
                    .accessibilityLabel("Create playlist")
                    .accessibilityHint("Creates a new playlist with the entered name")
                }
                .padding(.horizontal)

                if playlists.isEmpty {
                    EmptyStateView(
                        icon: "music.note.list",
                        title: "No playlists yet",
                        subtitle: "Create a playlist to rotate wallpapers automatically."
                    )
                } else {
                    ForEach(playlists) { playlist in
                        playlistCard(playlist)
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .task { playlists = await appState.playlistManager.playlists }
    }

    private func playlistCard(_ playlist: Playlist) -> some View {
        VStack(alignment: .leading, spacing: Spacing.sm) {
            HStack {
                VStack(alignment: .leading) {
                    Text(playlist.name)
                        .font(Typography.label)
                    Text("\(playlist.items.count) items")
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()

                if playlist.interval > 0 {
                    Text("Every \(playlist.interval / 60)min")
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }

                Toggle("Shuffle", isOn: Binding(
                    get: { playlist.shuffle },
                    set: { newValue in
                        var updated = playlist
                        updated.shuffle = newValue
                        Task {
                            try? await appState.playlistManager.update(updated)
                            playlists = await appState.playlistManager.playlists
                        }
                    }
                ))
                .toggleStyle(.switch)
                .labelsHidden()
                .accessibilityLabel("Shuffle \(playlist.name)")
                .accessibilityValue(playlist.shuffle ? "On" : "Off")

                Button(action: {
                    Task {
                        try? await appState.playlistManager.delete(id: playlist.id)
                        playlists = await appState.playlistManager.playlists
                    }
                }, label: {
                    Image(systemName: "trash")
                })
                .buttonStyle(.bordered)
                .tint(.red)
                .accessibilityLabel("Delete \(playlist.name)")
                .accessibilityHint("Permanently removes this playlist")
            }
        }
        .padding()
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 12))
    }
}
