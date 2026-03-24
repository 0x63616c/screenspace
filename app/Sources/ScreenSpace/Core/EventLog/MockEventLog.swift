import Foundation

@MainActor
final class MockEventLog: EventLogging {
    struct Entry: Equatable {
        let event: String
        let data: [String: String]
    }

    private(set) var events: [Entry] = []

    nonisolated func log(_ event: String, data: [String: String]) {
        MainActor.assumeIsolated {
            events.append(Entry(event: event, data: data))
        }
    }

    func reset() {
        events.removeAll()
    }
}
