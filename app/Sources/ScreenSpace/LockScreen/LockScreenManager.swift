import AVFoundation
import AppKit

final class LockScreenManager {
    enum LockScreenError: Error, LocalizedError {
        case noGeneratedUID
        case frameExtractionFailed
        case writePermissionDenied

        var errorDescription: String? {
            switch self {
            case .noGeneratedUID: return "Could not determine user ID for lock screen"
            case .frameExtractionFailed: return "Failed to extract frame from video"
            case .writePermissionDenied: return "Permission denied writing lock screen image"
            }
        }
    }

    func setLockScreen(from videoURL: URL, at time: CMTime = CMTime(seconds: 2, preferredTimescale: 600)) async throws {
        let image = try await extractFrame(from: videoURL, at: time)
        let uid = try getUserGeneratedUID()
        let lockScreenPath = "/Library/Caches/Desktop Pictures/\(uid)/lockscreen.png"
        try writeWithElevatedPermissions(image: image, to: lockScreenPath)
    }

    private func extractFrame(from url: URL, at time: CMTime) async throws -> CGImage {
        let asset = AVURLAsset(url: url)
        let generator = AVAssetImageGenerator(asset: asset)
        generator.appliesPreferredTrackTransform = true
        generator.requestedTimeToleranceBefore = .zero
        generator.requestedTimeToleranceAfter = .zero

        let (image, _) = try await generator.image(at: time)
        return image
    }

    private func getUserGeneratedUID() throws -> String {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/dscl")
        process.arguments = [".", "-read", "/Users/\(NSUserName())", "GeneratedUID"]

        let pipe = Pipe()
        process.standardOutput = pipe
        try process.run()
        process.waitUntilExit()

        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        guard let output = String(data: data, encoding: .utf8),
              let uid = output.split(separator: " ").last?.trimmingCharacters(in: .whitespacesAndNewlines) else {
            throw LockScreenError.noGeneratedUID
        }
        return uid
    }

    private func writeWithElevatedPermissions(image: CGImage, to path: String) throws {
        let tempURL = FileManager.default.temporaryDirectory.appendingPathComponent("lockscreen-\(UUID().uuidString).png")
        let dest = CGImageDestinationCreateWithURL(tempURL as CFURL, "public.png" as CFString, 1, nil)
        guard let dest = dest else { throw LockScreenError.frameExtractionFailed }
        CGImageDestinationAddImage(dest, image, nil)
        guard CGImageDestinationFinalize(dest) else { throw LockScreenError.frameExtractionFailed }

        let dirPath = URL(fileURLWithPath: path).deletingLastPathComponent().path
        let script = "do shell script \"mkdir -p '\(dirPath)' && cp '\(tempURL.path)' '\(path)'\" with administrator privileges"
        guard let appleScript = NSAppleScript(source: script) else { throw LockScreenError.writePermissionDenied }
        var error: NSDictionary?
        appleScript.executeAndReturnError(&error)
        if error != nil { throw LockScreenError.writePermissionDenied }
        try? FileManager.default.removeItem(at: tempURL)
    }
}
