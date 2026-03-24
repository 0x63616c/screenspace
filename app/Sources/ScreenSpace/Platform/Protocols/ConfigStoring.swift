import Foundation

protocol ConfigStoring: Sendable {
    func load() -> AppConfig
    func save(_ config: AppConfig) throws
}
