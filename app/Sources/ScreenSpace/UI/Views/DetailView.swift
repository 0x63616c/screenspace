import AVKit
import SwiftUI

struct DetailView: View {
    @Environment(AppState.self) var appState
    let wallpaper: WallpaperResponse
    @State private var isDownloading = false
    @State private var downloadProgress: Double = 0
    @State private var isFavorited = false
    @State private var showReportSheet = false
    @State private var reportReason = ""

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                // Video preview area
                if let previewURLString = wallpaper.previewURL,
                   let previewURL = URL(string: previewURLString) {
                    VideoPreview(url: previewURL)
                        .aspectRatio(16/9, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                } else {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(Color.black)
                        .aspectRatio(16/9, contentMode: .fit)
                        .overlay {
                            Image(systemName: "play.circle.fill")
                                .font(.system(size: 48))
                                .foregroundStyle(.white.opacity(0.8))
                        }
                }

                // Metadata overlay
                GlassCard {
                    VStack(alignment: .leading, spacing: Spacing.md) {
                        Text(wallpaper.title)
                            .font(.title2)
                            .fontWeight(.bold)

                        if let category = wallpaper.category {
                            Text(category.capitalized)
                                .font(.caption)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(.quaternary)
                                .clipShape(Capsule())
                        }

                        HStack(spacing: Spacing.lg) {
                            Label(wallpaper.resolution, systemImage: "rectangle.on.rectangle")
                                .accessibilityLabel("Resolution: \(wallpaper.resolution)")
                            Label(formattedSize, systemImage: "doc")
                                .accessibilityLabel("File size: \(formattedSize)")
                            Label(formattedDuration, systemImage: "clock")
                                .accessibilityLabel("Duration: \(formattedDuration)")
                            Label("\(wallpaper.downloadCount) downloads", systemImage: "arrow.down.circle")
                                .accessibilityLabel("\(wallpaper.downloadCount) downloads")
                        }
                        .font(.caption)
                        .foregroundStyle(.secondary)

                        if let tags = wallpaper.tags, !tags.isEmpty {
                            FlowLayout(spacing: 4) {
                                ForEach(tags, id: \.self) { tag in
                                    Text(tag)
                                        .font(.caption)
                                        .padding(.horizontal, 8)
                                        .padding(.vertical, 4)
                                        .background(.quaternary)
                                        .clipShape(Capsule())
                                }
                            }
                        }

                        HStack(spacing: Spacing.md) {
                            Button(action: setAsWallpaper) {
                                Label("Set as Wallpaper", systemImage: "photo.on.rectangle")
                            }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.regular)

                            Button(action: setAsLockScreen) {
                                Label("Lock Screen", systemImage: "lock.rectangle")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.regular)

                            if isDownloading {
                                ProgressView(value: downloadProgress)
                                    .frame(width: 100)
                            }

                            Button(action: {
                                guard appState.isLoggedIn else { return }
                                Task {
                                    isFavorited = try await appState.api.toggleFavorite(id: wallpaper.id)
                                }
                            }) {
                                Image(systemName: isFavorited ? "heart.fill" : "heart")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.regular)

                            Button(action: {
                                guard appState.isLoggedIn else { return }
                                showReportSheet = true
                            }) {
                                Image(systemName: "flag")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.regular)
                        }
                    }
                    .padding()
                }
            }
            .padding()
        }
        .sheet(isPresented: $showReportSheet) {
            VStack(spacing: Spacing.lg) {
                Text("Report Wallpaper")
                    .font(.headline)
                TextField("Reason for reporting", text: $reportReason)
                    .textFieldStyle(.roundedBorder)
                HStack {
                    Button("Cancel") {
                        reportReason = ""
                        showReportSheet = false
                    }
                    .buttonStyle(.bordered)
                    Button("Submit") {
                        Task {
                            try? await appState.api.reportWallpaper(id: wallpaper.id, reason: reportReason)
                            reportReason = ""
                            showReportSheet = false
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(reportReason.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .padding()
            .frame(width: 350)
        }
    }

    private var formattedSize: String {
        let mb = Double(wallpaper.fileSize) / 1_000_000
        return String(format: "%.0fMB", mb)
    }

    private var formattedDuration: String {
        "\(Int(wallpaper.duration))s"
    }

    private func setAsLockScreen() {
        Task {
            guard let cached = CacheManager.shared.cachedURL(for: wallpaper.id) else {
                return
            }
            let lockScreenManager = appState.lockScreen
            try? await lockScreenManager.setLockScreen(from: cached)
        }
    }

    private func setAsWallpaper() {
        Task {
            if let cachedURL = CacheManager.shared.cachedURL(for: wallpaper.id) {
                appState.setWallpaper(url: cachedURL, title: wallpaper.title)
                return
            }

            guard let downloadURLString = wallpaper.downloadURL,
                  let downloadURL = URL(string: downloadURLString) else { return }

            isDownloading = true
            downloadProgress = 0

            do {
                let (tempURL, _) = try await URLSession.shared.download(from: downloadURL, delegate: nil)
                let cachedURL = try CacheManager.shared.cacheFile(from: tempURL, wallpaperID: wallpaper.id)
                appState.setWallpaper(url: cachedURL, title: wallpaper.title)
            } catch {
                // Download failed silently for now
            }

            isDownloading = false
        }
    }
}

struct VideoPreview: NSViewRepresentable {
    let url: URL

    func makeNSView(context: Context) -> AVPlayerView {
        let view = AVPlayerView()
        view.controlsStyle = .inline
        view.showsFullScreenToggleButton = false
        let player = AVPlayer(url: url)
        player.isMuted = true
        view.player = player
        player.play()
        NotificationCenter.default.addObserver(
            forName: .AVPlayerItemDidPlayToEndTime,
            object: player.currentItem,
            queue: .main
        ) { _ in
            player.seek(to: .zero)
            player.play()
        }
        return view
    }

    func updateNSView(_ nsView: AVPlayerView, context: Context) {}
}

// Simple flow layout for tags
struct FlowLayout: Layout {
    var spacing: CGFloat = 4

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = arrange(proposal: proposal, subviews: subviews)
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = arrange(proposal: proposal, subviews: subviews)
        for (index, position) in result.positions.enumerated() {
            subviews[index].place(at: CGPoint(x: bounds.minX + position.x, y: bounds.minY + position.y), proposal: .unspecified)
        }
    }

    private func arrange(proposal: ProposedViewSize, subviews: Subviews) -> (positions: [CGPoint], size: CGSize) {
        let maxWidth = proposal.width ?? .infinity
        var positions: [CGPoint] = []
        var x: CGFloat = 0
        var y: CGFloat = 0
        var rowHeight: CGFloat = 0
        var maxX: CGFloat = 0

        for subview in subviews {
            let size = subview.sizeThatFits(.unspecified)
            if x + size.width > maxWidth && x > 0 {
                x = 0
                y += rowHeight + spacing
                rowHeight = 0
            }
            positions.append(CGPoint(x: x, y: y))
            rowHeight = max(rowHeight, size.height)
            x += size.width + spacing
            maxX = max(maxX, x)
        }

        return (positions, CGSize(width: maxX, height: y + rowHeight))
    }
}
