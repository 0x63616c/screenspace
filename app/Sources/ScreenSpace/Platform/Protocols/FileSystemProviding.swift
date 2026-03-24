import Foundation

protocol FileSystemProviding: Sendable {
    func fileExists(at url: URL) -> Bool
    func write(data: Data, to url: URL) throws
    func remove(at url: URL) throws
    func contentsOfDirectory(at url: URL) throws -> [URL]
    func fileSize(at url: URL) throws -> Int64
    func createDirectory(at url: URL) throws
}
