import Foundation
import HTTPTypes
import OpenAPIRuntime

struct AuthMiddleware: ClientMiddleware {
    private let keychain: KeychainProviding

    init(keychain: KeychainProviding) {
        self.keychain = keychain
    }

    func intercept(
        _ request: HTTPRequest,
        body: HTTPBody?,
        baseURL: URL,
        operationID: String,
        next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        var request = request
        if let tokenData = keychain.load(key: "auth_token"),
           let token = String(data: tokenData, encoding: .utf8)
        {
            request.headerFields[.authorization] = "Bearer \(token)"
        }
        return try await next(request, body, baseURL)
    }
}
