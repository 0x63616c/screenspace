import AVFoundation

enum VideoGravityOption: String, Codable, Sendable {
    case resizeAspectFill
    case resizeAspect

    var avLayerGravity: AVLayerVideoGravity {
        switch self {
        case .resizeAspectFill: return .resizeAspectFill
        case .resizeAspect: return .resizeAspect
        }
    }
}

struct AppConfig: Codable, Equatable, Sendable {
    var version: Int
    var launchAtLogin: Bool
    var pauseOnBattery: Bool
    var pauseOnFullscreen: Bool
    var videoGravity: VideoGravityOption
    var cacheSizeLimitMB: Int
    var serverURL: String
    var screenAssignments: [String: String]
    var lastPlayedURL: String?

    #if DEBUG
    static let defaultServerURL = "http://localhost:8080"
    #else
    static let defaultServerURL = "https://api.screenspace.app"
    #endif

    static let `default` = AppConfig(
        version: 1,
        launchAtLogin: true,
        pauseOnBattery: true,
        pauseOnFullscreen: true,
        videoGravity: .resizeAspectFill,
        cacheSizeLimitMB: 5120,
        serverURL: defaultServerURL,
        screenAssignments: [:],
        lastPlayedURL: nil
    )
}
