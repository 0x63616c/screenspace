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
            VStack(alignment: .leading, spacing: 16) {
                // Video preview area
                ZStack {
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
                    VStack(alignment: .leading, spacing: 12) {
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

                        HStack(spacing: 16) {
                            Label(wallpaper.resolution, systemImage: "rectangle.on.rectangle")
                            Label(formattedSize, systemImage: "doc")
                            Label(formattedDuration, systemImage: "clock")
                            Label("\(wallpaper.downloadCount) downloads", systemImage: "arrow.down.circle")
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

                        HStack(spacing: 12) {
                            Button(action: setAsWallpaper) {
                                Label("Set as Wallpaper", systemImage: "photo.on.rectangle")
                            }
                            .buttonStyle(.borderedProminent)
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
            VStack(spacing: 16) {
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
