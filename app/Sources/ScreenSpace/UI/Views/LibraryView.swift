import SwiftUI
import UniformTypeIdentifiers

struct LibraryView: View {
    @Environment(AppState.self) var appState
    @State private var localVideos: [URL] = []
    @State private var isDragOver = false
    @State private var importError: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                Text("Your Library")
                    .font(.title2)
                    .fontWeight(.bold)
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
                .font(.caption)
        }
        .foregroundStyle(isDragOver ? .blue : .secondary)
        .frame(maxWidth: .infinity)
        .frame(height: 120)
        .background(.quaternary, in: RoundedRectangle(cornerRadius: 12))
        .onDrop(of: [.fileURL], isTargeted: $isDragOver) { providers in
            handleDrop(providers)
            return true
        }
        .padding(.horizontal)
        .accessibilityLabel("Drop zone for video files. Drop MP4 or MOV files here to add to your library.")
    }

    private func localVideoCard(url: URL) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            LocalVideoThumbnail(url: url)

            Text(url.lastPathComponent)
                .font(.caption)
                .lineLimit(1)

            Button("Set as Wallpaper") {
                setWallpaper(url: url)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.small)
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

    private func handleDrop(_ providers: [NSItemProvider]) {
        for provider in providers {
            provider.loadItem(forTypeIdentifier: UTType.fileURL.identifier, options: nil) { item, _ in
                guard let data = item as? Data, let url = URL(dataRepresentation: data, relativeTo: nil) else { return }
                guard VideoImporter.isValidVideo(url: url) else { return }
                do {
                    let imported = try VideoImporter.importVideo(from: url, to: VideoImporter.libraryDirectory())
                    Task { @MainActor in
                        localVideos.append(imported)
                    }
                } catch {
                    let message = "Failed to import \(url.lastPathComponent): \(error.localizedDescription)"
                    Task { @MainActor in
                        importError = message
                    }
                }
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
                    .aspectRatio(16/9, contentMode: .fit)
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
