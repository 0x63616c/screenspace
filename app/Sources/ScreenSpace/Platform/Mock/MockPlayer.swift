import Foundation

@MainActor
final class MockPlayer: PlayerProviding {
    enum Call: Equatable {
        case play(URL)
        case pause
        case resume
        case seek(Double)
        case stop
    }

    var calls: [Call] = []

    nonisolated func play(url: URL) {
        MainActor.assumeIsolated {
            calls.append(.play(url))
        }
    }

    nonisolated func pause() {
        MainActor.assumeIsolated {
            calls.append(.pause)
        }
    }

    nonisolated func resume() {
        MainActor.assumeIsolated {
            calls.append(.resume)
        }
    }

    nonisolated func seek(to time: Double) {
        MainActor.assumeIsolated {
            calls.append(.seek(time))
        }
    }

    nonisolated func stop() {
        MainActor.assumeIsolated {
            calls.append(.stop)
        }
    }
}
