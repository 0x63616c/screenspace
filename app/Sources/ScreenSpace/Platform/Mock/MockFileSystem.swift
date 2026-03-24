import Foundation

@MainActor
final class MockFileSystem: FileSystemProviding {
    var files: [String: Data] = [:]
    var directories: Set<String> = []
    var shouldThrowOnWrite: Error?
    var shouldThrowOnRemove: Error?

    nonisolated func fileExists(at url: URL) -> Bool {
        MainActor.assumeIsolated {
            files[url.path] != nil || directories.contains(url.path)
        }
    }

    nonisolated func write(data: Data, to url: URL) throws {
        MainActor.assumeIsolated {
            if let error = shouldThrowOnWrite { fatalError("MockFileSystem write error: \(error)") }
            files[url.path] = data
        }
    }

    nonisolated func remove(at url: URL) throws {
        MainActor.assumeIsolated {
            if let error = shouldThrowOnRemove { fatalError("MockFileSystem remove error: \(error)") }
            files.removeValue(forKey: url.path)
            directories.remove(url.path)
        }
    }

    nonisolated func contentsOfDirectory(at url: URL) throws -> [URL] {
        MainActor.assumeIsolated {
            let prefix = url.path.hasSuffix("/") ? url.path : url.path + "/"
            return files.keys
                .filter { $0.hasPrefix(prefix) }
                .map { URL(fileURLWithPath: $0) }
        }
    }

    nonisolated func fileSize(at url: URL) throws -> Int64 {
        MainActor.assumeIsolated {
            Int64(files[url.path]?.count ?? 0)
        }
    }

    nonisolated func createDirectory(at url: URL) throws {
        MainActor.assumeIsolated {
            directories.insert(url.path)
        }
    }
}
