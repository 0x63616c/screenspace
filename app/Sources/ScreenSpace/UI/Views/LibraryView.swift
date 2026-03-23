import SwiftUI
import UniformTypeIdentifiers

struct LibraryView: View {
    @Environment(AppState.self) var appState
    @State private var localVideos: [URL] = []
    @State private var isDragOver = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Your Library")
                    .font(.title2)
                    .fontWeight(.bold)
                    .padding(.horizontal)

                if localVideos.isEmpty {
                    dropZone
                } else {
                    dropZone
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 200))], spacing: 12) {
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
    }

    private var dropZone: some View {
        VStack(spacing: 8) {
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
            .buttonStyle(.bordered)
            .controlSize(.small)
        }
        .frame(width: 200)
    }

    private func loadLibrary() {
        localVideos = VideoImporter.listLocalVideos()
    }

    private func handleDrop(_ providers: [NSItemProvider]) {
        for provider in providers {
            provider.loadItem(forTypeIdentifier: UTType.fileURL.identifier, options: nil) { item, _ in
                guard let data = item as? Data, let url = URL(dataRepresentation: data, relativeTo: nil) else { return }
                guard VideoImporter.isValidVideo(url: url) else { return }
                if let imported = try? VideoImporter.importVideo(from: url, to: VideoImporter.libraryDirectory()) {
                    DispatchQueue.main.async {
                        localVideos.append(imported)
                    }
                }
            }
        }
    }

    private func setWallpaper(url: URL) {
        appState.setWallpaper(url: url, title: url.lastPathComponent)
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
