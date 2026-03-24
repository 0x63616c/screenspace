import Testing
import Foundation
@testable import ScreenSpace

@Suite("EventLog")
@MainActor
struct EventLogTests {
    @Test("log writes JSONL entry with required fields")
    func writesRequiredFields() async throws {
        let dir = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent("eventlog-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: dir) }

        let log = EventLog(
            logsDirectory: dir,
            fileSystem: LiveFileSystem(),
            sessionID: "test-session",
            appVersion: "0.0.1"
        )
        log.log("wallpaper_set", data: ["display": "built-in", "source": "community"])
        try await Task.sleep(for: .milliseconds(100))

        let logURL = dir.appendingPathComponent("events.jsonl")
        let data = try Data(contentsOf: logURL)
        let line = String(data: data, encoding: .utf8) ?? ""
        let json = try JSONSerialization.jsonObject(with: Data(line.utf8)) as! [String: Any]

        #expect(json["ts"] != nil)
        #expect(json["sid"] as? String == "test-session")
        #expect(json["v"] as? String == "0.0.1")
        #expect(json["event"] as? String == "wallpaper_set")
        #expect(json["data"] != nil)
    }

    @Test("MockEventLog records events in order")
    func mockRecordsEvents() {
        let log = MockEventLog()
        log.log("app_launched", data: [:])
        log.log("wallpaper_set", data: ["display": "built-in"])
        #expect(log.events.count == 2)
        #expect(log.events[0].event == "app_launched")
        #expect(log.events[1].event == "wallpaper_set")
    }

    @Test("MockEventLog reset clears events")
    func mockReset() {
        let log = MockEventLog()
        log.log("app_launched", data: [:])
        log.reset()
        #expect(log.events.isEmpty)
    }

    @Test("MockEventLog event data is recorded")
    func mockEventData() {
        let log = MockEventLog()
        log.log("wallpaper_set", data: ["display": "built-in", "source": "local"])
        #expect(log.events.last?.data["display"] == "built-in")
        #expect(log.events.last?.data["source"] == "local")
    }
}
