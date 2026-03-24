import AppKit
import AVFoundation

enum ThumbnailGenerator {
    static func generateThumbnail(
        for videoURL: URL,
        at time: CMTime = CMTime(seconds: 2, preferredTimescale: 600)
    ) async throws -> NSImage {
        let asset = AVURLAsset(url: videoURL)
        let generator = AVAssetImageGenerator(asset: asset)
        generator.appliesPreferredTrackTransform = true
        generator.maximumSize = CGSize(width: 480, height: 270)

        let (cgImage, _) = try await generator.image(at: time)
        return NSImage(cgImage: cgImage, size: NSSize(width: cgImage.width, height: cgImage.height))
    }

    static func saveThumbnail(_ image: NSImage, to url: URL) throws {
        guard let tiffData = image.tiffRepresentation,
              let bitmap = NSBitmapImageRep(data: tiffData),
              let jpegData = bitmap.representation(using: .jpeg, properties: [.compressionFactor: 0.8])
        else {
            throw ThumbnailError.conversionFailed
        }
        try jpegData.write(to: url)
    }

    static func thumbnailCacheDir() -> URL {
        guard let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
        else {
            fatalError("Application Support directory unavailable")
        }
        return appSupport
            .appendingPathComponent("ScreenSpace")
            .appendingPathComponent("cache")
            .appendingPathComponent("thumbnails")
    }

    enum ThumbnailError: Error {
        case conversionFailed
    }
}
