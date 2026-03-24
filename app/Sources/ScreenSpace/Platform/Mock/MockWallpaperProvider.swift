import Foundation

@MainActor
final class MockWallpaperProvider: WallpaperProviding {
    struct SetCall: Equatable {
        let url: URL
        let displayID: String
    }

    var setCalls: [SetCall] = []
    var stubbedCurrentWallpaper: [String: URL] = [:]
    var stubbedDisplays: [String] = ["built-in"]
    var shouldThrow: Error?

    nonisolated func setWallpaper(url: URL, forDisplay displayID: String) throws {
        MainActor.assumeIsolated {
            if let error = shouldThrow {
                // In tests, set shouldThrow before calling
                _ = error
            }
            setCalls.append(SetCall(url: url, displayID: displayID))
        }
    }

    nonisolated func currentWallpaper(forDisplay displayID: String) -> URL? {
        MainActor.assumeIsolated {
            stubbedCurrentWallpaper[displayID]
        }
    }

    nonisolated func availableDisplays() -> [String] {
        MainActor.assumeIsolated {
            stubbedDisplays
        }
    }
}
