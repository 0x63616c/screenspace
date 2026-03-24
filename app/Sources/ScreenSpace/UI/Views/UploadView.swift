import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct UploadView: View {
    @Environment(AppState.self) var appState
    @Environment(\.dismiss) private var dismiss
    @State private var viewModel: UploadViewModel?

    var body: some View {
        Group {
            if let viewModel {
                UploadContentView(viewModel: viewModel, dismiss: dismiss)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = UploadViewModel(
                    api: appState.apiService,
                    fileSystem: appState.fileSystem,
                    eventLog: appState.eventLog
                )
            }
        }
    }
}

private struct UploadContentView: View {
    @Bindable var viewModel: UploadViewModel
    let dismiss: DismissAction

    var body: some View {
        VStack(spacing: Spacing.lg) {
            Text("Upload Wallpaper")
                .font(Typography.pageTitle)

            // File picker
            GroupBox {
                if let url = viewModel.selectedFileURL {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Image(systemName: "film")
                            Text(url.lastPathComponent)
                            Spacer()
                            if let size = viewModel.formattedFileSize {
                                Text(size)
                                    .font(Typography.meta)
                                    .foregroundStyle(viewModel.fileTooLarge ? .red : .secondary)
                            }
                            Button("Change") { pickFile() }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                        }
                        if viewModel.fileTooLarge {
                            Text("File exceeds 200 MB limit")
                                .font(Typography.meta).foregroundStyle(.red)
                        }
                    }
                } else {
                    Button("Select Video File") { pickFile() }
                        .buttonStyle(.bordered)
                        .accessibilityLabel("Select video file")
                        .accessibilityHint("Opens file picker for MP4 or MOV files")
                }
            }

            // Metadata
            TextField("Title", text: $viewModel.title)
                .textFieldStyle(.roundedBorder)

            Picker("Category", selection: $viewModel.category) {
                Text("Select category").tag(nil as Category?)
                ForEach(viewModel.categories, id: \.self) { cat in
                    Text(cat.rawValue.capitalized).tag(cat as Category?)
                }
            }

            TextField("Tags (comma separated)", text: $viewModel.tagsText)
                .textFieldStyle(.roundedBorder)

            // Content policy
            Toggle(isOn: $viewModel.acceptedPolicy) {
                HStack(spacing: 4) {
                    Text("I confirm this content complies with the")
                    Text("content policy")
                        .foregroundStyle(.blue)
                        .underline()
                        .onTapGesture {
                            if let url =
                                URL(string: "https://github.com/0x63616c/screenspace/blob/main/CONTENT_POLICY.md")
                            {
                                NSWorkspace.shared.open(url)
                            }
                        }
                    Text("and I have the rights to upload it")
                }
                .font(Typography.meta)
            }
            .accessibilityLabel("Content policy agreement")
            .accessibilityValue(viewModel.acceptedPolicy ? "Agreed" : "Not agreed")

            if viewModel.isUploading {
                ProgressView(value: viewModel.uploadProgress)
                Text("Uploading...")
                    .font(Typography.meta)
                    .foregroundStyle(.secondary)
            }

            if viewModel.uploadComplete {
                Label("Upload complete! Pending review.", systemImage: "checkmark.circle")
                    .foregroundStyle(.green)
            }

            HStack {
                Button("Cancel") { dismiss() }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)

                Button("Upload") { Task { await viewModel.upload() } }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.regular)
                    .disabled(!viewModel.canUpload)
                    .accessibilityLabel("Upload wallpaper")
                    .accessibilityHint("Submits your wallpaper for review")
            }
        }
        .padding()
        .frame(width: 400)
        .task { await viewModel.loadCategories() }
        .errorAlert(message: $viewModel.errorMessage)
    }

    private func pickFile() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.mpeg4Movie, .quickTimeMovie]
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK {
            viewModel.selectedFileURL = panel.url
        }
    }
}
