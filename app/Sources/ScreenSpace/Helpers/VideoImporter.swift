import Foundation
import UniformTypeIdentifiers

enum VideoImporter {
    static let supportedTypes: [UTType] = [.mpeg4Movie, .quickTimeMovie]
    static let supportedExtensions = Set(["mp4", "mov"])

    static func isValidVideo(url: URL) -> Bool {
        supportedExtensions.contains(url.pathExtension.lowercased()) && FileManager.default.isReadableFile(atPath: url.path)
    }

    static func importVideo(from sourceURL: URL, to libraryDir: URL) throws -> URL {
        try FileManager.default.createDirectory(at: libraryDir, withIntermediateDirectories: true)
        let filename = "\(UUID().uuidString).\(sourceURL.pathExtension)"
        let destURL = libraryDir.appendingPathComponent(filename)
        try FileManager.default.copyItem(at: sourceURL, to: destURL)
        return destURL
    }

    static func libraryDirectory() -> URL {
        FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            .appendingPathComponent("ScreenSpace")
            .appendingPathComponent("Library")
    }

    static func listLocalVideos() -> [URL] {
        let dir = libraryDirectory()
        guard let files = try? FileManager.default.contentsOfDirectory(at: dir, includingPropertiesForKeys: nil) else {
            return []
        }
        return files.filter { supportedExtensions.contains($0.pathExtension.lowercased()) }
    }
}
