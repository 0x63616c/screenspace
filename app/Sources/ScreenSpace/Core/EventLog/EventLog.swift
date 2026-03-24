import Foundation

actor EventLog: EventLogging {
    private let logsDirectory: URL
    private let fileSystem: FileSystemProviding
    private let maxFileSizeBytes: Int64
    private let maxFileCount: Int
    private let sessionID: String
    private let appVersion: String

    static let shared: EventLog = {
        guard let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
        else {
            fatalError("Application Support directory unavailable")
        }
        let dir = appSupport
            .appendingPathComponent("ScreenSpace")
            .appendingPathComponent("logs")
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return EventLog(logsDirectory: dir)
    }()

    private static let logFileName = "events.jsonl"
    private static let defaultMaxFileSizeBytes: Int64 = 5 * 1024 * 1024
    private static let defaultMaxFileCount = 3

    init(
        logsDirectory: URL,
        fileSystem: FileSystemProviding = LiveFileSystem(),
        maxFileSizeBytes: Int64 = EventLog.defaultMaxFileSizeBytes,
        maxFileCount: Int = EventLog.defaultMaxFileCount,
        sessionID: String = UUID().uuidString,
        appVersion: String = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "unknown"
    ) {
        self.logsDirectory = logsDirectory
        self.fileSystem = fileSystem
        self.maxFileSizeBytes = maxFileSizeBytes
        self.maxFileCount = maxFileCount
        self.sessionID = sessionID
        self.appVersion = appVersion
    }

    nonisolated func log(_ event: String, data: [String: String]) {
        Task {
            await self.write(event: event, data: data)
        }
    }

    private func write(event: String, data: [String: String]) {
        let entry = buildEntry(event: event, data: data)
        guard let line = serialize(entry) else { return }
        let logURL = logsDirectory.appendingPathComponent(EventLog.logFileName)
        rotateIfNeeded(logURL: logURL)
        append(line: line, to: logURL)
    }

    private func buildEntry(event: String, data: [String: String]) -> [String: Any] {
        let formatter = ISO8601DateFormatter()
        return [
            "ts": formatter.string(from: Date()),
            "sid": sessionID,
            "v": appVersion,
            "event": event,
            "data": data
        ]
    }

    private func serialize(_ entry: [String: Any]) -> String? {
        guard let data = try? JSONSerialization.data(withJSONObject: entry),
              let line = String(data: data, encoding: .utf8)
        else { return nil }
        return line + "\n"
    }

    private func rotateIfNeeded(logURL: URL) {
        guard let size = try? fileSystem.fileSize(at: logURL),
              size >= maxFileSizeBytes
        else { return }
        rotate()
    }

    private func rotate() {
        let oldest = logsDirectory.appendingPathComponent("events.\(maxFileCount).jsonl")
        try? fileSystem.remove(at: oldest)

        for i in stride(from: maxFileCount - 1, through: 1, by: -1) {
            let src = logsDirectory.appendingPathComponent("events.\(i).jsonl")
            let dst = logsDirectory.appendingPathComponent("events.\(i + 1).jsonl")
            if fileSystem.fileExists(at: src) {
                try? FileManager.default.moveItem(at: src, to: dst)
            }
        }

        let active = logsDirectory.appendingPathComponent(EventLog.logFileName)
        let rotated = logsDirectory.appendingPathComponent("events.1.jsonl")
        try? FileManager.default.moveItem(at: active, to: rotated)
    }

    private func append(line: String, to url: URL) {
        guard let data = line.data(using: .utf8) else { return }
        if fileSystem.fileExists(at: url) {
            guard let handle = try? FileHandle(forWritingTo: url) else { return }
            handle.seekToEndOfFile()
            handle.write(data)
            try? handle.close()
        } else {
            try? fileSystem.write(data: data, to: url)
        }
    }
}
