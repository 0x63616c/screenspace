import Foundation

struct LiveConfigStore: ConfigStoring {
    private let configURL: URL

    init(configURL: URL? = nil) {
        self.configURL = configURL ?? {
            guard let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)
                .first
            else {
                fatalError("Application Support directory unavailable")
            }
            let dir = appSupport.appendingPathComponent("ScreenSpace")
            try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
            return dir.appendingPathComponent("config.json")
        }()
    }

    func load() -> AppConfig {
        guard let data = try? Data(contentsOf: configURL),
              let config = try? JSONDecoder().decode(AppConfig.self, from: data)
        else {
            return .default
        }
        return config
    }

    func save(_ config: AppConfig) throws {
        let data = try JSONEncoder().encode(config)
        try data.write(to: configURL, options: .atomic)
    }
}
