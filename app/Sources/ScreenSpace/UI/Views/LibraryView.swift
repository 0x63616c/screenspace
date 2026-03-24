import SwiftUI

struct LibraryView: View {
    @Environment(AppState.self) var appState
    @State private var localVideos: [URL] = []
    @State private var isDragOver = false
    @State private var importError: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Your Library")
                    .font(Typography.pageTitle)
                    .padding(.horizontal)

                if localVideos.isEmpty {
                    dropZone
                } else {
                    dropZone
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: Spacing.md) {
                        ForEach(localVideos, id: \.absoluteString) { url in
                            localVideoCard(url: url)
                        }
                    }
                    .padding(.horizontal)
                }
            }
            .padding(.vertical)
        }
        .onAppear { loadLibrary() }
        .alert("Import Error", isPresented: Binding(
            get: { importError != nil },
            set: { if !$0 { importError = nil } }
        )) {
            Button("OK") { importError = nil }
        } message: {
            if let importError {
                Text(importError)
            }
        }
    }

    private var dropZone: some View {
        VStack(spacing: Spacing.sm) {
            Image(systemName: "arrow.down.doc")
                .font(.title)
            Text("Drop MP4 or MOV files here")
                .font(Typography.meta)
        }
        .foregroundStyle(isDragOver ? .blue : .secondary)
        .frame(maxWidth: .infinity)
        .frame(height: 120)
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 12))
        .dropDestination(for: URL.self) { urls, _ in
            handleDroppedURLs(urls)
            return true
        } isTargeted: { targeted in
            isDragOver = targeted
        }
        .padding(.horizontal)
        .accessibilityLabel("Drop zone for video files. Drop MP4 or MOV files here to add to your library.")
    }

    private func localVideoCard(url: URL) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            LocalVideoThumbnail(url: url)

            Text(url.lastPathComponent)
                .font(Typography.meta)
                .lineLimit(1)

            Button("Set as Wallpaper") {
                setWallpaper(url: url)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.small)
            .accessibilityLabel("Set \(url.lastPathComponent) as wallpaper")
            .accessibilityHint("Plays this video as your desktop wallpaper")
        }
        .frame(width: 200)
        .contextMenu {
            Button("Set as Wallpaper") { setWallpaper(url: url) }
            Divider()
            Button("Remove from Library", role: .destructive) {
                try? FileManager.default.removeItem(at: url)
                localVideos.removeAll { $0 == url }
            }
        }
    }

    private func loadLibrary() {
        localVideos = VideoImporter.listLocalVideos()
    }

    private func handleDroppedURLs(_ urls: [URL]) {
        for url in urls {
            guard VideoImporter.isValidVideo(url: url) else { continue }
            do {
                let imported = try VideoImporter.importVideo(from: url, to: VideoImporter.libraryDirectory())
                localVideos.append(imported)
            } catch {
                importError = "Failed to import \(url.lastPathComponent): \(error.localizedDescription)"
            }
        }
    }

    private func setWallpaper(url: URL) {
        Task { await appState.setWallpaper(url: url, title: url.lastPathComponent) }
    }
}

private struct LocalVideoThumbnail: View {
    let url: URL
    @State private var thumbnail: NSImage?

    var body: some View {
        Group {
            if let thumbnail {
                Image(nsImage: thumbnail)
                    .resizable()
                    .scaledToFill()
                    .frame(height: 112)
                    .clipShape(RoundedRectangle(cornerRadius: 12))
            } else {
                RoundedRectangle(cornerRadius: 12)
                    .fill(Color.gray.opacity(0.3))
                    .aspectRatio(16 / 9, contentMode: .fit)
                    .overlay {
                        Image(systemName: "play.circle")
                            .font(.title)
                            .foregroundStyle(.white.opacity(0.7))
                    }
            }
        }
        .task {
            thumbnail = try? await ThumbnailGenerator.generateThumbnail(for: url)
        }
    }
}
