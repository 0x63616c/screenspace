import Foundation

@Observable
@MainActor
final class UploadViewModel {
    private let api: APIProviding
    private let fileSystem: FileSystemProviding
    private let eventLog: EventLogging

    var selectedFileURL: URL?
    var title = ""
    var category: Category?
    var tagsText = ""
    var acceptedPolicy = false
    var isUploading = false
    var uploadProgress: Double = 0
    var uploadComplete = false
    var errorMessage: String?
    var categories: [Category] = []

    init(api: APIProviding, fileSystem: FileSystemProviding, eventLog: EventLogging) {
        self.api = api
        self.fileSystem = fileSystem
        self.eventLog = eventLog
    }

    var canUpload: Bool {
        selectedFileURL != nil && !title.trimmingCharacters(in: .whitespaces).isEmpty && acceptedPolicy && !isUploading
    }

    var selectedFileSizeMB: Double? {
        guard let url = selectedFileURL,
              let size = try? fileSystem.fileSize(at: url) else { return nil }
        return Double(size) / 1_000_000
    }

    var fileTooLarge: Bool {
        (selectedFileSizeMB ?? 0) > 200
    }

    var parsedTags: [String] {
        tagsText.split(separator: ",").map { $0.trimmingCharacters(in: .whitespaces) }.filter { !$0.isEmpty }
    }

    func loadCategories() async {
        do {
            categories = try await api.listCategories()
        } catch {
            categories = Category.allCases
        }
    }

    func upload() async {
        guard canUpload, let fileURL = selectedFileURL else { return }
        isUploading = true
        errorMessage = nil
        uploadProgress = 0.1

        do {
            let ticket = try await api.initiateUpload(title: title, category: category, tags: parsedTags)
            uploadProgress = 0.3

            guard let uploadURL = URL(string: ticket.uploadURL) else {
                errorMessage = "Invalid upload URL from server."
                isUploading = false
                return
            }
            var request = URLRequest(url: uploadURL)
            request.httpMethod = "PUT"
            request.setValue("video/mp4", forHTTPHeaderField: "Content-Type")
            let fileData = try Data(contentsOf: fileURL)
            let (_, response) = try await URLSession.shared.upload(for: request, from: fileData)
            guard let http = response as? HTTPURLResponse, (200 ..< 300).contains(http.statusCode) else {
                throw APIError.httpError(status: 0)
            }

            uploadProgress = 0.8
            try await api.finalizeUpload(id: ticket.id)
            uploadProgress = 1.0
            uploadComplete = true
            eventLog.log("wallpaper_uploaded", data: ["id": ticket.id, "title": title])
        } catch {
            errorMessage = error.localizedDescription
            eventLog.log("error", data: ["context": "upload", "message": error.localizedDescription])
        }

        isUploading = false
    }
}
