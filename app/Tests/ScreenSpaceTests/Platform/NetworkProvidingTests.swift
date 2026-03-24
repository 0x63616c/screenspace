import Foundation
import Testing
@testable import ScreenSpace

@MainActor
struct NetworkProvidingTests {
    @Test("returns stubbed response for matching path")
    func stubbedResponse() async throws {
        let network = MockNetwork()
        let responseData = Data("{\"ok\":true}".utf8)
        network.stubs["/api/v1/test"] = MockNetwork.Stub(data: responseData, statusCode: 200)

        let request = try URLRequest(url: #require(URL(string: "https://example.com/api/v1/test")))
        let (data, response) = try await network.data(for: request)
        let httpResponse = try #require(response as? HTTPURLResponse)

        #expect(data == responseData)
        #expect(httpResponse.statusCode == 200)
    }

    @Test("returns default stub for unknown path")
    func defaultStub() async throws {
        let network = MockNetwork()
        network.defaultStub = MockNetwork.Stub(data: Data("default".utf8), statusCode: 404)

        let request = try URLRequest(url: #require(URL(string: "https://example.com/unknown")))
        let (data, response) = try await network.data(for: request)
        let httpResponse = try #require(response as? HTTPURLResponse)

        #expect(data == Data("default".utf8))
        #expect(httpResponse.statusCode == 404)
    }

    @Test("throws injected error")
    func errorInjection() async throws {
        let network = MockNetwork()
        network.error = URLError(.notConnectedToInternet)

        let request = try URLRequest(url: #require(URL(string: "https://example.com/api")))
        await #expect(throws: URLError.self) {
            _ = try await network.data(for: request)
        }
    }
}
