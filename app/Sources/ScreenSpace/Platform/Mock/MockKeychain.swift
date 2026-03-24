import Foundation

@MainActor
final class MockKeychain: KeychainProviding {
    private var store: [String: Data] = [:]

    nonisolated func save(key: String, data: Data) throws {
        MainActor.assumeIsolated {
            store[key] = data
        }
    }

    nonisolated func load(key: String) -> Data? {
        MainActor.assumeIsolated {
            store[key]
        }
    }

    nonisolated func delete(key: String) {
        MainActor.assumeIsolated {
            store.removeValue(forKey: key)
        }
    }
}
