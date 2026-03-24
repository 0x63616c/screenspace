import Foundation

protocol EventLogging: Sendable {
    func log(_ event: String, data: [String: String])
}
