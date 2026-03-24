import Foundation

protocol KeychainProviding: Sendable {
    func save(key: String, data: Data) throws
    func load(key: String) -> Data?
    func delete(key: String)
}
