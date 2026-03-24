import SwiftUI

struct PlaylistsView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: PlaylistsViewModel?

    var body: some View {
        Group {
            if let viewModel {
                PlaylistsContentView(viewModel: viewModel)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                let vm = PlaylistsViewModel(
                    playlistManager: appState.playlistManager,
                    eventLog: appState.eventLog
                )
                vm.load()
                viewModel = vm
            }
        }
    }
}

private struct PlaylistsContentView: View {
    @Bindable var viewModel: PlaylistsViewModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Playlists")
                    .font(Typography.pageTitle)
                    .padding(.horizontal)

                HStack {
                    TextField("New playlist name", text: $viewModel.newPlaylistName)
                        .textFieldStyle(.roundedBorder)
                        .accessibilityLabel("New playlist name")
                    Button("Create") {
                        viewModel.create()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.newPlaylistName.isEmpty)
                    .accessibilityLabel("Create playlist")
                    .accessibilityHint("Creates a new playlist with the entered name")
                }
                .padding(.horizontal)

                if viewModel.playlists.isEmpty {
                    EmptyStateView(
                        icon: "music.note.list",
                        title: "No playlists yet",
                        subtitle: "Create a playlist to rotate wallpapers automatically."
                    )
                } else {
                    ForEach(viewModel.playlists) { playlist in
                        playlistCard(playlist)
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .errorAlert(message: Binding(
            get: { viewModel.error },
            set: { viewModel.error = $0 }
        ))
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
                        viewModel.updateShuffle(playlist, enabled: newValue)
                    }
                ))
                .toggleStyle(.switch)
                .labelsHidden()
                .accessibilityLabel("Shuffle \(playlist.name)")
                .accessibilityValue(playlist.shuffle ? "On" : "Off")

                Button(action: {
                    viewModel.delete(playlist: playlist)
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
