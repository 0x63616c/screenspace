import SwiftUI
import UniformTypeIdentifiers

struct LibraryView: View {
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
                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 180))], spacing: 12) {
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
        RoundedRectangle(cornerRadius: 12)
            .strokeBorder(style: StrokeStyle(lineWidth: 2, dash: [8]))
            .foregroundStyle(isDragOver ? .blue : .secondary)
            .frame(height: 120)
            .overlay {
                VStack(spacing: 8) {
                    Image(systemName: "arrow.down.doc")
                        .font(.title)
                    Text("Drop MP4 or MOV files here")
                        .font(.caption)
                }
                .foregroundStyle(isDragOver ? .blue : .secondary)
            }
            .onDrop(of: [.fileURL], isTargeted: $isDragOver) { providers in
                handleDrop(providers)
                return true
            }
            .padding(.horizontal)
    }

    private func localVideoCard(url: URL) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            RoundedRectangle(cornerRadius: 8)
                .fill(Color.gray.opacity(0.3))
                .aspectRatio(16/9, contentMode: .fit)
                .overlay {
                    Image(systemName: "play.circle")
                        .font(.title)
                        .foregroundStyle(.white.opacity(0.7))
                }

            Text(url.lastPathComponent)
                .font(.caption)
                .lineLimit(1)

            Button("Set as Wallpaper") {
                setWallpaper(url: url)
            }
            .buttonStyle(.bordered)
            .controlSize(.small)
        }
        .frame(width: 180)
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
        // This will be wired to WallpaperEngine via environment or shared state
    }
}
