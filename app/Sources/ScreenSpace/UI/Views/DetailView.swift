import SwiftUI

struct DetailView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: DetailViewModel?
    let wallpaper: WallpaperDetail

    var body: some View {
        Group {
            if let viewModel {
                DetailContentView(viewModel: viewModel, appState: appState)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = DetailViewModel(
                    wallpaper: wallpaper,
                    api: appState.apiService,
                    wallpaperProvider: appState.wallpaperProvider,
                    cache: appState.cache,
                    eventLog: appState.eventLog
                )
            }
        }
    }
}

private struct DetailContentView: View {
    @Bindable var viewModel: DetailViewModel
    let appState: AppState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: Spacing.lg) {
                // Video preview area
                if let previewURL = viewModel.wallpaper.previewURL {
                    VideoPreview(url: previewURL)
                        .aspectRatio(16 / 9, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                } else {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(Color.black)
                        .aspectRatio(16 / 9, contentMode: .fit)
                        .overlay {
                            Image(systemName: "play.circle.fill")
                                .font(.system(size: 48))
                                .foregroundStyle(.white.opacity(0.8))
                        }
                }

                // Metadata overlay
                GlassCard {
                    VStack(alignment: .leading, spacing: Spacing.md) {
                        Text(viewModel.wallpaper.title)
                            .font(Typography.pageTitle)

                        if let category = viewModel.wallpaper.category {
                            PillBadge(text: category.rawValue.capitalized)
                        }

                        HStack(spacing: Spacing.lg) {
                            Label(viewModel.wallpaper.resolution, systemImage: "rectangle.on.rectangle")
                                .accessibilityLabel("Resolution: \(viewModel.wallpaper.resolution)")
                            Label(viewModel.formattedSize, systemImage: "doc")
                                .accessibilityLabel("File size: \(viewModel.formattedSize)")
                            Label(viewModel.formattedDuration, systemImage: "clock")
                                .accessibilityLabel("Duration: \(viewModel.formattedDuration)")
                            Label("\(viewModel.wallpaper.downloadCount) downloads", systemImage: "arrow.down.circle")
                                .accessibilityLabel("\(viewModel.wallpaper.downloadCount) downloads")
                        }
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                        .accessibilityElement(children: .combine)

                        if !viewModel.wallpaper.tags.isEmpty {
                            FlowLayout(spacing: 4) {
                                ForEach(viewModel.wallpaper.tags, id: \.self) { tag in
                                    PillBadge(text: tag)
                                        .accessibilityLabel("Tag: \(tag)")
                                }
                            }
                        }

                        HStack(spacing: Spacing.md) {
                            Button(action: { Task { await viewModel.setAsWallpaper() } }) {
                                Label("Set as Wallpaper", systemImage: "photo.on.rectangle")
                            }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.regular)
                            .accessibilityLabel("Set \(viewModel.wallpaper.title) as wallpaper")
                            .accessibilityHint("Downloads and plays this wallpaper on your desktop")

                            Button(action: setAsLockScreen) {
                                Label("Lock Screen", systemImage: "lock.rectangle")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.regular)
                            .accessibilityLabel("Set as lock screen")
                            .accessibilityHint("Uses a still frame from this wallpaper as your lock screen")

                            if viewModel.isDownloading {
                                ProgressView(value: viewModel.downloadProgress)
                                    .frame(width: 100)
                                    .accessibilityLabel("Downloading wallpaper")
                                    .accessibilityValue("\(Int(viewModel.downloadProgress * 100))%")
                            }

                            Button(action: {
                                Task { await viewModel.toggleFavorite(isLoggedIn: appState.isLoggedIn) }
                            }, label: {
                                Image(systemName: viewModel.isFavorited ? "heart.fill" : "heart")
                            })
                            .buttonStyle(.bordered)
                            .controlSize(.regular)
                            .accessibilityLabel(viewModel.isFavorited ? "Remove from favorites" : "Add to favorites")
                            .accessibilityValue(viewModel.isFavorited ? "Favorited" : "Not favorited")

                            Button(action: {
                                guard appState.isLoggedIn else { return }
                                viewModel.showReportSheet = true
                            }, label: {
                                Image(systemName: "flag")
                            })
                            .buttonStyle(.bordered)
                            .controlSize(.regular)
                            .accessibilityLabel("Report wallpaper")
                            .accessibilityHint("Report this content as inappropriate")
                        }
                    }
                    .padding()
                }
            }
            .padding()
        }
        .sheet(isPresented: $viewModel.showReportSheet) {
            VStack(spacing: Spacing.lg) {
                Text("Report Wallpaper")
                    .font(Typography.label)
                TextField("Reason for reporting", text: $viewModel.reportReason)
                    .textFieldStyle(.roundedBorder)
                HStack {
                    Button("Cancel") {
                        viewModel.reportReason = ""
                        viewModel.showReportSheet = false
                    }
                    .buttonStyle(.bordered)
                    Button("Submit") {
                        Task { await viewModel.submitReport(isLoggedIn: appState.isLoggedIn) }
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(viewModel.reportReason.trimmingCharacters(in: .whitespaces).isEmpty)
                }
            }
            .padding()
            .frame(width: 350)
        }
        .errorAlert(message: Binding(
            get: { viewModel.error },
            set: { viewModel.error = $0 }
        ))
    }

    private func setAsLockScreen() {
        Task {
            guard let cached = appState.cache.cachedURL(for: viewModel.wallpaper.id) else {
                viewModel.error = "Download the wallpaper first before setting it as lock screen."
                return
            }
            do {
                try await appState.lockScreen.setLockScreen(from: cached)
            } catch {
                viewModel.error = "Failed to set lock screen: \(error.localizedDescription)"
            }
        }
    }
}
