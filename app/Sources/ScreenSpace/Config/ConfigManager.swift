import Foundation

final class ConfigManager: @unchecked Sendable {
    static let shared = ConfigManager()

    private let configURL: URL
    private(set) var config: AppConfig

    init(directory: URL? = nil) {
        let dir = directory ?? FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
            .appendingPathComponent("ScreenSpace")
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        self.configURL = dir.appendingPathComponent("config.json")
        self.config = Self.load(from: configURL) ?? .default
    }

    private static func load(from url: URL) -> AppConfig? {
        guard let data = try? Data(contentsOf: url) else { return nil }
        guard let config = try? JSONDecoder().decode(AppConfig.self, from: data) else {
            let backup = url.deletingLastPathComponent()
                .appendingPathComponent("config.json.backup.\(Int(Date().timeIntervalSince1970))")
            try? FileManager.default.moveItem(at: url, to: backup)
            return nil
        }
        return config
    }

    func save() throws {
        let data = try JSONEncoder().encode(config)
        try data.write(to: configURL, options: .atomic)
    }

    func update(_ transform: (inout AppConfig) -> Void) throws {
        transform(&config)
        try save()
    }
}
