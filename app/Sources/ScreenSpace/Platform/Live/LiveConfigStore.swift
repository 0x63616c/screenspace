import Foundation

struct LiveConfigStore: ConfigStoring {
    private let configURL: URL

    init(configURL: URL) {
        self.configURL = configURL
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
