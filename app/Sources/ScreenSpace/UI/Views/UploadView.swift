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

    var body: some View {
        VStack(spacing: 16) {
            Text("Upload Wallpaper")
                .font(.title2)
                .fontWeight(.bold)

            // File picker
            GroupBox {
                if let url = selectedFileURL {
                    HStack {
                        Image(systemName: "film")
                        Text(url.lastPathComponent)
                        Spacer()
                        Button("Change") { pickFile() }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                    }
                } else {
                    Button("Select Video File") { pickFile() }
                        .buttonStyle(.bordered)
                }
            }

            // Metadata
            TextField("Title", text: $title)
                .textFieldStyle(.roundedBorder)

            TextField("Category (e.g. nature, abstract, urban)", text: $category)
                .textFieldStyle(.roundedBorder)

            TextField("Tags (comma separated)", text: $tagsText)
                .textFieldStyle(.roundedBorder)

            // Content policy
            Toggle(isOn: $acceptedPolicy) {
                Text("I confirm this content complies with the content policy and I have the rights to upload it")
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
            var uploadRequest = URLRequest(url: URL(string: initResponse.uploadURL)!)
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
