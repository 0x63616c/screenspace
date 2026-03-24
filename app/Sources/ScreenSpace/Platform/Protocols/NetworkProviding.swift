import Foundation

protocol NetworkProviding: Sendable {
    func data(for request: URLRequest) async throws -> (Data, URLResponse)
}
