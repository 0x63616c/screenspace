import Foundation

@MainActor
final class MockNetwork: NetworkProviding {
    struct Stub {
        let data: Data
        let statusCode: Int
    }

    var stubs: [String: Stub] = [:]
    var defaultStub: Stub = Stub(data: Data(), statusCode: 200)
    var error: Error?

    nonisolated func data(for request: URLRequest) async throws -> (Data, URLResponse) {
        try await MainActor.run {
            if let error { throw error }
            let path = request.url?.path ?? ""
            let stub = stubs[path] ?? defaultStub
            let response = HTTPURLResponse(
                url: request.url!,
                statusCode: stub.statusCode,
                httpVersion: "HTTP/1.1",
                headerFields: nil
            )!
            return (stub.data, response)
        }
    }
}
