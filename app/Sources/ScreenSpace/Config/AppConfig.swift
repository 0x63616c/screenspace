import AVFoundation

enum VideoGravityOption: String, Codable {
    case resizeAspectFill
    case resizeAspect

    var avLayerGravity: AVLayerVideoGravity {
        switch self {
        case .resizeAspectFill: return .resizeAspectFill
        case .resizeAspect: return .resizeAspect
        }
    }
}

struct AppConfig: Codable, Equatable {
    var version: Int
    var launchAtLogin: Bool
    var pauseOnBattery: Bool
    var pauseOnFullscreen: Bool
    var videoQuality: String
    var videoGravity: VideoGravityOption
    var cacheSizeLimitMB: Int
    var serverURL: String
    var screenAssignments: [String: String]
    var lastPlayedURL: String?

    static let `default` = AppConfig(
        version: 1,
        launchAtLogin: true,
        pauseOnBattery: true,
        pauseOnFullscreen: true,
        videoQuality: "original",
        videoGravity: .resizeAspectFill,
        cacheSizeLimitMB: 5120,
        serverURL: "https://api.screenspace.app",
        screenAssignments: [:],
        lastPlayedURL: nil
    )
}
