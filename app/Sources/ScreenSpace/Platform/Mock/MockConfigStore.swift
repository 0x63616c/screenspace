import Foundation

@MainActor
final class MockConfigStore: ConfigStoring {
    var storedConfig: AppConfig = .default
    var saveCallCount = 0

    nonisolated func load() -> AppConfig {
        MainActor.assumeIsolated {
            storedConfig
        }
    }

    nonisolated func save(_ config: AppConfig) throws {
        MainActor.assumeIsolated {
            storedConfig = config
            saveCallCount += 1
        }
    }
}
