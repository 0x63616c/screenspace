import AppKit
import SwiftUI
import UniformTypeIdentifiers

struct UploadView: View {
    @Environment(AppState.self) var appState
    @Environment(\.dismiss) private var dismiss
    @State private var selectedFileURL: URL?
    @State private var title = ""
    @State private var category = ""
    @State private var tagsText = ""
    @State private var acceptedPolicy = false
    @State private var isUploading = false
    @State private var uploadProgress: Double = 0
    @State private var errorMessage: String?
    @State private var uploadComplete = false
    @State private var categories: [String] = []

    var body: some View {
        VStack(spacing: Spacing.lg) {
            Text("Upload Wallpaper")
                .font(.title2)
                .fontWeight(.bold)

            // File picker
            GroupBox {
                if let url = selectedFileURL {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Image(systemName: "film")
                            Text(url.lastPathComponent)
                            Spacer()
                            if let attrs = try? FileManager.default.attributesOfItem(atPath: url.path),
                               let size = attrs[.size] as? Int {
                                let mb = Double(size) / 1_000_000
                                Text(String(format: "%.1f MB", mb))
                                    .font(.caption)
                                    .foregroundStyle(mb > 200 ? .red : .secondary)
                            }
                            Button("Change") { pickFile() }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                        }
                        if let attrs = try? FileManager.default.attributesOfItem(atPath: url.path),
                           let size = attrs[.size] as? Int, Double(size) / 1_000_000 > 200 {
                            Text("File exceeds 200 MB limit")
                                .font(.caption).foregroundStyle(.red)
                        }
                    }
                } else {
                    Button("Select Video File") { pickFile() }
                        .buttonStyle(.bordered)
                }
            }

            // Metadata
            TextField("Title", text: $title)
                .textFieldStyle(.roundedBorder)

            Picker("Category", selection: $category) {
                Text("Select category").tag("")
                ForEach(categories, id: \.self) { cat in
                    Text(cat.capitalized).tag(cat)
                }
            }

            TextField("Tags (comma separated)", text: $tagsText)
                .textFieldStyle(.roundedBorder)

            // Content policy
            Toggle(isOn: $acceptedPolicy) {
                HStack(spacing: 4) {
                    Text("I confirm this content complies with the")
                    Text("content policy")
                        .foregroundStyle(.blue)
                        .underline()
                        .onTapGesture {
                            NSWorkspace.shared.open(URL(string: "https://github.com/0x63616c/screenspace/blob/main/CONTENT_POLICY.md")!)
                        }
                    Text("and I have the rights to upload it")
                }
                .font(.caption)
            }

            if let error = errorMessage {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
            }

            if isUploading {
                ProgressView(value: uploadProgress)
                Text("Uploading...")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if uploadComplete {
                Label("Upload complete! Pending review.", systemImage: "checkmark.circle")
                    .foregroundStyle(.green)
            }

            HStack {
                Button("Cancel") { dismiss() }
                    .buttonStyle(.bordered)
                    .controlSize(.regular)

                Button("Upload") { Task { await upload() } }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.regular)
                    .disabled(!canUpload)
            }
        }
        .padding()
        .frame(width: 400)
        .task {
            do {
                categories = try await appState.api.listCategories()
            } catch {
                categories = ["nature", "abstract", "urban", "cinematic", "space", "underwater", "minimal", "other"]
            }
        }
    }

    private var canUpload: Bool {
        selectedFileURL != nil && !title.isEmpty && acceptedPolicy && !isUploading
    }

    private func pickFile() {
        let panel = NSOpenPanel()
        panel.allowedContentTypes = [.mpeg4Movie, .quickTimeMovie]
        panel.allowsMultipleSelection = false
        if panel.runModal() == .OK {
            selectedFileURL = panel.url
        }
    }

    private func upload() async {
        guard let fileURL = selectedFileURL else { return }
        isUploading = true
        errorMessage = nil

        let tags = tagsText.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }

        do {
            // Step 1: Initiate upload
            uploadProgress = 0.1
            let initResponse = try await appState.api.initiateUpload(
                title: title,
                category: category.isEmpty ? nil : category,
                tags: tags
            )

            // Step 2: Upload file to pre-signed URL
            uploadProgress = 0.3
            guard let uploadURL = URL(string: initResponse.uploadURL) else {
                errorMessage = "Invalid upload URL from server"
                isUploading = false
                return
            }
            var uploadRequest = URLRequest(url: uploadURL)
            uploadRequest.httpMethod = "PUT"
            uploadRequest.setValue("video/mp4", forHTTPHeaderField: "Content-Type")
            let fileData = try Data(contentsOf: fileURL)
            let (_, response) = try await URLSession.shared.upload(for: uploadRequest, from: fileData)
            guard let httpResponse = response as? HTTPURLResponse, (200..<300).contains(httpResponse.statusCode) else {
                throw APIClient.APIError.httpError(statusCode: 0, message: "Upload to storage failed")
            }

            // Step 3: Finalize
            uploadProgress = 0.8
            try await appState.api.finalizeUpload(id: initResponse.id)

            uploadProgress = 1.0
            uploadComplete = true
        } catch {
            errorMessage = error.localizedDescription
        }

        isUploading = false
    }
}
