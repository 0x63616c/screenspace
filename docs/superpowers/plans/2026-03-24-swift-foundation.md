# Swift Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the Swift infrastructure layer: Platform protocols, EventLog, typed enums, @Observable migration, Sendable conformances, concurrency fixes, and linting configuration.

**Architecture:** All OS interactions are moved behind protocol interfaces with Live (production) and Mock (test/preview) implementations. PauseController is migrated from ObservableObject/Combine to @Observable with async notification sequences. ConfigManager and PlaylistManager become actors to eliminate @unchecked Sendable. EventLog writes JSONL to disk with file rotation.

**Tech Stack:** Swift 6, SwiftUI, macOS 15+, Swift Testing (@Test/@Suite), SwiftFormat (Lockwood), SwiftLint.

**Spec:** `docs/superpowers/specs/2026-03-23-refactor-design.md` Phases 2.1, 2.3, 2.4, 2.5, 2.6, 2.11 and Phase 3.3, 3.4.

---

## Errata (post-review fixes, apply throughout)

1. **All Mock classes must be `@MainActor` isolated** to satisfy Swift 6 strict concurrency. `MockEventLog`, `MockFileSystem`, `MockKeychain`, `MockNetwork`, `MockPlayer` all have mutable state. Mark each as `@MainActor final class` (matching MockWallpaperProvider's pattern). Do NOT use `nonisolated(unsafe)` or leave as a TODO.
2. **`MockWallpaperProvider.setWallpaper` must record the call.** The protocol method should append to `setCalls` directly. Remove the separate `recordSetWallpaper` method.
3. **EventLog test `writesRequiredFields()` must have real assertions.** After calling `log()`, read the written file from MockFileSystem, decode the JSONL line, and `#expect` that `ts`, `sid`, `v`, `event`, `data` keys are present.
4. **`[String: Any]` -> `[String: String]` in EventLogging protocol** is an accepted deviation from the spec for Swift 6 Sendable. This is correct. The spec example is illustrative.
5. **`createDirectory(at:)` added to FileSystemProviding** is an accepted extension (needed by EventLog). Not in the spec's original table but required.

---

## Current State (read before editing)

Key facts about existing code:

- `PauseController` uses `ObservableObject` + `@Published var shouldPause` + Combine sinks for NSWorkspace/NotificationCenter/DistributedNotificationCenter observation. It also uses a `Timer` and `DispatchQueue.main.asyncAfter`.
- `AppState` imports Combine and wires `PauseController.$shouldPause` via `.sink` to call `engine.pauseAll()`/`engine.resumeAll()`. Has `private var cancellables = Set<AnyCancellable>()`.
- `ConfigManager` is `final class: @unchecked Sendable` with a mutable `config` property. No synchronization.
- `PlaylistManager` is `final class: @unchecked Sendable` with a mutable `playlists` array. No synchronization.
- `APIClient` is `final class: @unchecked Sendable`. Uses `URLSession.shared` directly without a `NetworkProviding` protocol.
- `KeychainHelper` is a static enum without a `KeychainProviding` protocol.
- Existing tests use **XCTest** (`XCTestCase`). New tests MUST use **Swift Testing** (`@Test`/`@Suite`).
- `AppConfig`, `WallpaperResponse`, `UserResponse`, `Playlist`, `PlaylistItem`, and all API model structs lack explicit `Sendable` conformance.
- `isAdmin` in `AppState` compares `currentUser?.role == "admin"` as a raw string.
- No `Platform/` directory exists yet.
- No `EventLog` exists yet.
- No `.swiftformat` or `.swiftlint.yml` exists yet.

---

## Track D: Swift Infrastructure

### D1: Platform Protocol Layer

**Files to create:**

```
app/Sources/ScreenSpace/Platform/Protocols/WallpaperProviding.swift   (Create)
app/Sources/ScreenSpace/Platform/Protocols/FileSystemProviding.swift  (Create)
app/Sources/ScreenSpace/Platform/Protocols/KeychainProviding.swift    (Create)
app/Sources/ScreenSpace/Platform/Protocols/NetworkProviding.swift     (Create)
app/Sources/ScreenSpace/Platform/Protocols/PlayerProviding.swift      (Create)
app/Sources/ScreenSpace/Platform/Protocols/ConfigStoring.swift        (Create)
app/Sources/ScreenSpace/Platform/Live/LiveWallpaperProvider.swift     (Create)
app/Sources/ScreenSpace/Platform/Live/LiveFileSystem.swift            (Create)
app/Sources/ScreenSpace/Platform/Live/LiveKeychain.swift              (Create)
app/Sources/ScreenSpace/Platform/Live/LiveNetwork.swift               (Create)
app/Sources/ScreenSpace/Platform/Live/LivePlayer.swift                (Create)
app/Sources/ScreenSpace/Platform/Live/LiveConfigStore.swift           (Create)
app/Sources/ScreenSpace/Platform/Mock/MockWallpaperProvider.swift     (Create)
app/Sources/ScreenSpace/Platform/Mock/MockFileSystem.swift            (Create)
app/Sources/ScreenSpace/Platform/Mock/MockKeychain.swift              (Create)
app/Sources/ScreenSpace/Platform/Mock/MockNetwork.swift               (Create)
app/Sources/ScreenSpace/Platform/Mock/MockPlayer.swift                (Create)
app/Sources/ScreenSpace/Platform/Mock/MockConfigStore.swift           (Create)
app/Tests/ScreenSpaceTests/Platform/WallpaperProvidingTests.swift     (Create)
app/Tests/ScreenSpaceTests/Platform/FileSystemProvidingTests.swift    (Create)
app/Tests/ScreenSpaceTests/Platform/KeychainProvidingTests.swift      (Create)
```

#### D1.1 — Protocol definitions

- [ ] Create `Platform/Protocols/WallpaperProviding.swift`:

```swift
import AppKit

protocol WallpaperProviding: Sendable {
    func setWallpaper(url: URL, forDisplay displayID: String) throws
    func currentWallpaper(forDisplay displayID: String) -> URL?
    func availableDisplays() -> [String]
}
```

- [ ] Create `Platform/Protocols/FileSystemProviding.swift`:

```swift
import Foundation

protocol FileSystemProviding: Sendable {
    func fileExists(at url: URL) -> Bool
    func write(data: Data, to url: URL) throws
    func remove(at url: URL) throws
    func contentsOfDirectory(at url: URL) throws -> [URL]
    func fileSize(at url: URL) throws -> Int64
    func createDirectory(at url: URL) throws
}
```

- [ ] Create `Platform/Protocols/KeychainProviding.swift`:

```swift
import Foundation

protocol KeychainProviding: Sendable {
    func save(key: String, data: Data) throws
    func load(key: String) -> Data?
    func delete(key: String)
}
```

- [ ] Create `Platform/Protocols/NetworkProviding.swift`:

```swift
import Foundation

protocol NetworkProviding: Sendable {
    func data(for request: URLRequest) async throws -> (Data, URLResponse)
}
```

- [ ] Create `Platform/Protocols/PlayerProviding.swift`:

```swift
import Foundation

protocol PlayerProviding: Sendable {
    func play(url: URL)
    func pause()
    func resume()
    func seek(to time: Double)
    func stop()
}
```

- [ ] Create `Platform/Protocols/ConfigStoring.swift`:

```swift
import Foundation

protocol ConfigStoring: Sendable {
    func load() -> AppConfig
    func save(_ config: AppConfig) throws
}
```

- [ ] Commit: `feat(platform): add Platform protocol definitions`

#### D1.2 — Live implementations

- [ ] Create `Platform/Live/LiveWallpaperProvider.swift`:

```swift
import AppKit

struct LiveWallpaperProvider: WallpaperProviding {
    func setWallpaper(url: URL, forDisplay displayID: String) throws {
        // Wraps NSWorkspace desktop picture setting.
        // Finds the NSScreen matching displayID and calls
        // NSWorkspace.shared.setDesktopImageURL(_:for:options:).
        guard let screen = NSScreen.screens.first(where: {
            DisplayIdentifier.stableID(for: $0) == displayID
        }) else { return }
        try NSWorkspace.shared.setDesktopImageURL(url, for: screen, options: [:])
    }

    func currentWallpaper(forDisplay displayID: String) -> URL? {
        guard let screen = NSScreen.screens.first(where: {
            DisplayIdentifier.stableID(for: $0) == displayID
        }) else { return nil }
        return NSWorkspace.shared.desktopImageURL(for: screen)
    }

    func availableDisplays() -> [String] {
        NSScreen.screens.map { DisplayIdentifier.stableID(for: $0) }
    }
}
```

- [ ] Create `Platform/Live/LiveFileSystem.swift`:

```swift
import Foundation

struct LiveFileSystem: FileSystemProviding {
    func fileExists(at url: URL) -> Bool {
        FileManager.default.fileExists(atPath: url.path)
    }

    func write(data: Data, to url: URL) throws {
        try data.write(to: url, options: .atomic)
    }

    func remove(at url: URL) throws {
        try FileManager.default.removeItem(at: url)
    }

    func contentsOfDirectory(at url: URL) throws -> [URL] {
        try FileManager.default.contentsOfDirectory(
            at: url,
            includingPropertiesForKeys: nil
        )
    }

    func fileSize(at url: URL) throws -> Int64 {
        let attrs = try FileManager.default.attributesOfItem(atPath: url.path)
        return attrs[.size] as? Int64 ?? 0
    }

    func createDirectory(at url: URL) throws {
        try FileManager.default.createDirectory(
            at: url,
            withIntermediateDirectories: true
        )
    }
}
```

- [ ] Create `Platform/Live/LiveKeychain.swift`. Wraps `Security` framework. Include `kSecAttrAccessible: kSecAttrAccessibleWhenUnlockedThisDeviceOnly` on all operations. Define `KeychainError: Error` enum with cases `saveFailed(OSStatus)`, `unexpectedData`. This replaces the existing `KeychainHelper.swift` static enum.

```swift
import Foundation
import Security

struct LiveKeychain: KeychainProviding {
    enum KeychainError: Error {
        case saveFailed(OSStatus)
        case unexpectedData
    }

    private let service: String

    init(service: String = "co.worldwidewebb.screenspace") {
        self.service = service
    }

    func save(key: String, data: Data) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(query as CFDictionary)

        var addQuery = query
        addQuery[kSecValueData as String] = data
        addQuery[kSecAttrAccessible as String] = kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        let status = SecItemAdd(addQuery as CFDictionary, nil)
        guard status == errSecSuccess else {
            throw KeychainError.saveFailed(status)
        }
    }

    func load(key: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        guard status == errSecSuccess else { return nil }
        return result as? Data
    }

    func delete(key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(query as CFDictionary)
    }
}
```

- [ ] Create `Platform/Live/LiveNetwork.swift`:

```swift
import Foundation

struct LiveNetwork: NetworkProviding {
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    func data(for request: URLRequest) async throws -> (Data, URLResponse) {
        try await session.data(for: request)
    }
}
```

- [ ] Create `Platform/Live/LivePlayer.swift`. Wraps `AVQueuePlayer`. `@MainActor` because `AVQueuePlayer` must run on main thread for UI integration.

```swift
import AVFoundation

@MainActor
final class LivePlayer: PlayerProviding {
    private let player = AVQueuePlayer()

    nonisolated func play(url: URL) {
        Task { @MainActor in
            let item = AVPlayerItem(url: url)
            player.replaceCurrentItem(with: item)
            player.play()
        }
    }

    nonisolated func pause() {
        Task { @MainActor in player.pause() }
    }

    nonisolated func resume() {
        Task { @MainActor in player.play() }
    }

    nonisolated func seek(to time: Double) {
        Task { @MainActor in
            let cmTime = CMTime(seconds: time, preferredTimescale: 600)
            player.seek(to: cmTime)
        }
    }

    nonisolated func stop() {
        Task { @MainActor in
            player.pause()
            player.replaceCurrentItem(with: nil)
        }
    }
}
```

- [ ] Create `Platform/Live/LiveConfigStore.swift`. Extracts the file-based load/save logic from `ConfigManager`. This is used by the actor version of `ConfigManager` internally.

```swift
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
```

- [ ] Commit: `feat(platform): add Live protocol implementations`

#### D1.3 — Mock implementations

- [ ] Create `Platform/Mock/MockWallpaperProvider.swift`. Records all `setWallpaper` calls. Returns canned values for `currentWallpaper` and `availableDisplays`.

```swift
import Foundation

@MainActor
final class MockWallpaperProvider: WallpaperProviding {
    struct SetCall: Equatable {
        let url: URL
        let displayID: String
    }

    var setCalls: [SetCall] = []
    var stubbedCurrentWallpaper: [String: URL] = [:]
    var stubbedDisplays: [String] = ["built-in"]
    var shouldThrow: Error?

    nonisolated func setWallpaper(url: URL, forDisplay displayID: String) throws {
        // Capture on main actor — callers in tests are @MainActor
        if let error = shouldThrow { throw error }
    }

    func recordSetWallpaper(url: URL, forDisplay displayID: String) {
        setCalls.append(SetCall(url: url, displayID: displayID))
    }

    nonisolated func currentWallpaper(forDisplay displayID: String) -> URL? {
        nil
    }

    nonisolated func availableDisplays() -> [String] {
        ["built-in"]
    }
}
```

> Note: Because `WallpaperProviding` is `Sendable` and mocks need mutation for assertions, use `@MainActor` isolation on the mock class. This is valid in test code where everything runs on MainActor.

- [ ] Create `Platform/Mock/MockFileSystem.swift`. In-memory dictionary keyed by URL path. Supports `fileExists`, `write`, `remove`, `contentsOfDirectory`, `fileSize`, `createDirectory`.

```swift
import Foundation

@MainActor
final class MockFileSystem: FileSystemProviding {
    var files: [String: Data] = [:]
    var directories: Set<String> = []
    var shouldThrowOnWrite: Error?
    var shouldThrowOnRemove: Error?

    nonisolated func fileExists(at url: URL) -> Bool {
        // For test simplicity, check synchronously — mocks are always called from test main actor
        false // overridden by subclasses or checked via MainActor
    }

    // Implement full thread-safe mock using actor isolation.
    // All methods are nonisolated to satisfy Sendable, delegate to internal state.
    // In tests, call from @MainActor context.

    func exists(at url: URL) -> Bool {
        files[url.path] != nil || directories.contains(url.path)
    }

    nonisolated func write(data: Data, to url: URL) throws {}
    nonisolated func remove(at url: URL) throws {}
    nonisolated func contentsOfDirectory(at url: URL) throws -> [URL] { [] }
    nonisolated func fileSize(at url: URL) throws -> Int64 { 0 }
    nonisolated func createDirectory(at url: URL) throws {}
}
```

> Implementation note: The full mock uses an actor to safely store in-memory state. Expose a `MainActor`-isolated helper API (`set(data:at:)`, `removeFile(at:)`) for test setup, plus nonisolated protocol methods that read from internal state. See full implementation in tests.

- [ ] Create `Platform/Mock/MockKeychain.swift`. Dictionary-backed in-memory store.

```swift
import Foundation

final class MockKeychain: KeychainProviding {
    private var store: [String: Data] = [:]

    func save(key: String, data: Data) throws {
        store[key] = data
    }

    func load(key: String) -> Data? {
        store[key]
    }

    func delete(key: String) {
        store.removeValue(forKey: key)
    }
}
```

> `MockKeychain` is a `final class` with no shared mutable state across threads (used only in single-threaded test context). Mark `nonisolated(unsafe)` on `store` if compiler complains, or make it an actor.

- [ ] Create `Platform/Mock/MockNetwork.swift`. Returns pre-configured `(Data, HTTPURLResponse)` pairs keyed by URL path. Supports error injection.

```swift
import Foundation

final class MockNetwork: NetworkProviding {
    struct Stub {
        let data: Data
        let statusCode: Int
    }

    var stubs: [String: Stub] = [:]
    var defaultStub: Stub = Stub(data: Data(), statusCode: 200)
    var error: Error?

    func data(for request: URLRequest) async throws -> (Data, URLResponse) {
        if let error { throw error }
        let path = request.url?.path ?? ""
        let stub = stubs[path] ?? defaultStub
        let response = HTTPURLResponse(
            url: request.url!,
            statusCode: stub.statusCode,
            httpVersion: "HTTP/1.1",
            headerFields: nil
        )!
        return (stub.data, response)
    }
}
```

- [ ] Create `Platform/Mock/MockPlayer.swift`. Records play/pause/resume/seek/stop calls.

```swift
import Foundation

final class MockPlayer: PlayerProviding {
    enum Call: Equatable {
        case play(URL)
        case pause
        case resume
        case seek(Double)
        case stop
    }

    var calls: [Call] = []

    func play(url: URL) { calls.append(.play(url)) }
    func pause() { calls.append(.pause) }
    func resume() { calls.append(.resume) }
    func seek(to time: Double) { calls.append(.seek(time)) }
    func stop() { calls.append(.stop) }
}
```

- [ ] Create `Platform/Mock/MockConfigStore.swift`. In-memory config, starts with `AppConfig.default`.

```swift
import Foundation

final class MockConfigStore: ConfigStoring {
    var storedConfig: AppConfig = .default
    var saveCallCount = 0

    func load() -> AppConfig {
        storedConfig
    }

    func save(_ config: AppConfig) throws {
        storedConfig = config
        saveCallCount += 1
    }
}
```

- [ ] Commit: `feat(platform): add Mock protocol implementations`

#### D1.4 — Tests for mock behaviour

- [ ] Create `Tests/ScreenSpaceTests/Platform/KeychainProvidingTests.swift`:

```swift
import Testing
@testable import ScreenSpace

@Suite("MockKeychain")
struct KeychainProvidingTests {
    @Test("saves and loads data by key")
    func saveAndLoad() throws {
        let keychain = MockKeychain()
        let data = Data("token".utf8)
        try keychain.save(key: "auth_token", data: data)
        let loaded = keychain.load(key: "auth_token")
        #expect(loaded == data)
    }

    @Test("returns nil for missing key")
    func missingKey() {
        let keychain = MockKeychain()
        #expect(keychain.load(key: "missing") == nil)
    }

    @Test("delete removes key")
    func deleteKey() throws {
        let keychain = MockKeychain()
        try keychain.save(key: "auth_token", data: Data("x".utf8))
        keychain.delete(key: "auth_token")
        #expect(keychain.load(key: "auth_token") == nil)
    }

    @Test("overwrite replaces existing value")
    func overwrite() throws {
        let keychain = MockKeychain()
        try keychain.save(key: "k", data: Data("v1".utf8))
        try keychain.save(key: "k", data: Data("v2".utf8))
        #expect(keychain.load(key: "k") == Data("v2".utf8))
    }
}
```

- [ ] Create `Tests/ScreenSpaceTests/Platform/FileSystemProvidingTests.swift` testing `MockFileSystem` write/exists/remove/contentsOfDirectory.
- [ ] Create `Tests/ScreenSpaceTests/Platform/NetworkProvidingTests.swift` testing `MockNetwork` stub routing and error injection.
- [ ] Verify tests pass: `cd /path/to/app && swift test --filter KeychainProvidingTests`

Expected output: `Test Suite 'KeychainProvidingTests' passed`

- [ ] Commit: `test(platform): add Swift Testing tests for mock protocol implementations`

---

### D2: EventLog System

**Files:**

```
app/Sources/ScreenSpace/Core/EventLog/EventLogging.swift      (Create)
app/Sources/ScreenSpace/Core/EventLog/EventLog.swift          (Create)
app/Sources/ScreenSpace/Core/EventLog/MockEventLog.swift      (Create)
app/Tests/ScreenSpaceTests/Core/EventLogTests.swift           (Create)
```

#### D2.1 — EventLogging protocol

- [ ] Create `Core/EventLog/EventLogging.swift`:

```swift
import Foundation

protocol EventLogging: Sendable {
    func log(_ event: String, data: [String: String])
}
```

> Use `[String: String]` for `data` (not `[String: Any]`) to keep EventLogging `Sendable` without custom conformances. All event data values are stringified at the call site. This differs slightly from the spec's `[String: Any]` but is required for Swift 6 strict concurrency.

#### D2.2 — EventLog actor

- [ ] Create `Core/EventLog/EventLog.swift`:

```swift
import Foundation

actor EventLog: EventLogging {
    private let logsDirectory: URL
    private let fileSystem: FileSystemProviding
    private let maxFileSizeBytes: Int64
    private let maxFileCount: Int
    private let sessionID: String
    private let appVersion: String

    private static let logFileName = "events.jsonl"
    private static let defaultMaxFileSizeBytes: Int64 = 5 * 1024 * 1024 // 5MB
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
            "data": data,
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
        // Delete oldest if at max count
        let oldest = logsDirectory.appendingPathComponent("events.\(maxFileCount).jsonl")
        try? fileSystem.remove(at: oldest)

        // Shift existing rotated files: events.2.jsonl -> events.3.jsonl, etc.
        for i in stride(from: maxFileCount - 1, through: 1, by: -1) {
            let src = logsDirectory.appendingPathComponent("events.\(i).jsonl")
            let dst = logsDirectory.appendingPathComponent("events.\(i + 1).jsonl")
            if fileSystem.fileExists(at: src) {
                // Use FileManager directly for rename (atomic, no data copy)
                try? FileManager.default.moveItem(at: src, to: dst)
            }
        }

        // Rename active log to events.1.jsonl
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
```

#### D2.3 — MockEventLog

- [ ] Create `Core/EventLog/MockEventLog.swift`:

```swift
import Foundation

final class MockEventLog: EventLogging {
    struct Entry: Equatable {
        let event: String
        let data: [String: String]
    }

    private(set) var events: [Entry] = []

    func log(_ event: String, data: [String: String]) {
        events.append(Entry(event: event, data: data))
    }

    func reset() {
        events.removeAll()
    }
}
```

> `MockEventLog` is a `final class` used only in tests. Mark `nonisolated(unsafe) private var events` if Swift 6 strict concurrency requires it, or make it an actor. For test use, calling from `@MainActor` test context is sufficient.

#### D2.4 — EventLog tests

- [ ] Create `Tests/ScreenSpaceTests/Core/EventLogTests.swift`:

```swift
import Testing
import Foundation
@testable import ScreenSpace

@Suite("EventLog")
struct EventLogTests {
    private func makeLog(maxFileSizeBytes: Int64 = 5 * 1024 * 1024) -> (EventLog, MockFileSystem, URL) {
        let dir = URL(fileURLWithPath: "/tmp/eventlog-\(UUID().uuidString)")
        let fs = MockFileSystem()
        let log = EventLog(
            logsDirectory: dir,
            fileSystem: fs,
            maxFileSizeBytes: maxFileSizeBytes,
            sessionID: "test-session",
            appVersion: "0.0.1"
        )
        return (log, fs, dir)
    }

    @Test("log writes JSONL entry with required fields")
    func writesRequiredFields() async throws {
        let (log, _, dir) = makeLog()
        log.log("wallpaper_set", data: ["display": "built-in", "source": "community"])
        // Allow actor task to complete
        try await Task.sleep(for: .milliseconds(50))
        let logURL = dir.appendingPathComponent("events.jsonl")
        // File should exist and contain the event
        // In a real test with LiveFileSystem, verify via FileManager.
        // With MockFileSystem, verify via mock state.
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
```

- [ ] Run tests: `cd app && swift test --filter EventLogTests`

Expected output: `Test Suite 'EventLogTests' passed`

- [ ] Commit: `feat(core): add EventLog actor with JSONL rotation and MockEventLog`

---

### D3: Swift Typed Enums

**Files:**

```
app/Sources/ScreenSpace/Core/Types/WallpaperStatus.swift    (Create)
app/Sources/ScreenSpace/Core/Types/UserRole.swift           (Create)
app/Sources/ScreenSpace/Core/Types/Category.swift           (Create)
app/Sources/ScreenSpace/Core/Types/SortOrder.swift          (Create)
app/Sources/ScreenSpace/API/APIModels.swift                 (Modify)
app/Sources/ScreenSpace/AppState.swift                      (Modify)
app/Tests/ScreenSpaceTests/Core/TypedEnumTests.swift        (Create)
```

#### D3.1 — Enum files

- [ ] Create `Core/Types/WallpaperStatus.swift`:

```swift
import Foundation

enum WallpaperStatus: String, Codable, Sendable {
    case pending
    case pendingReview = "pending_review"
    case approved
    case rejected
}
```

- [ ] Create `Core/Types/UserRole.swift`:

```swift
import Foundation

enum UserRole: String, Codable, Sendable {
    case user
    case admin
}
```

- [ ] Create `Core/Types/Category.swift`:

```swift
import Foundation

enum Category: String, Codable, CaseIterable, Sendable {
    case nature
    case abstract
    case urban
    case cinematic
    case space
    case underwater
    case minimal
    case other
}
```

- [ ] Create `Core/Types/SortOrder.swift`:

```swift
import Foundation

enum SortOrder: String, Codable, Sendable {
    case recent
    case popular
}
```

#### D3.2 — Update APIModels.swift

- [ ] Modify `API/APIModels.swift`:
  - Change `WallpaperResponse.status: String?` to `status: WallpaperStatus?`
  - Change `WallpaperResponse.category: String?` to `category: Category?`
  - Change `AuthResponse.role: String` to `role: UserRole`
  - Change `UserResponse.role: String` to `role: UserRole`
  - Change `ReportResponse.status: String` to `status: WallpaperStatus`
  - Add `Sendable` conformance to: `AuthRequest`, `AuthResponse`, `CategoriesResponse`, `WallpaperResponse`, `WallpaperListResponse`, `UploadInitResponse`, `UserResponse`, `UserListResponse`, `ReportResponse`, `ReportListResponse`

- [ ] Update `AppState.isAdmin` to compare against `UserRole.admin` instead of `"admin"` string:

```swift
var isAdmin: Bool { currentUser?.role == .admin }
```

- [ ] Update `APIClient.listWallpapers(sort:)` parameter to accept `SortOrder` instead of `String`.

- [ ] Update `APIClient.initiateUpload(category:)` parameter to accept `Category?` instead of `String?`. Encode as `.rawValue`.

- [ ] Update `APIClient.listAllWallpapers(status:)` parameter to accept `WallpaperStatus?` instead of `String?`.

#### D3.3 — Tests

- [ ] Create `Tests/ScreenSpaceTests/Core/TypedEnumTests.swift`:

```swift
import Testing
@testable import ScreenSpace

@Suite("Typed Enums")
struct TypedEnumTests {
    @Test("WallpaperStatus round-trips through Codable")
    func wallpaperStatusCodable() throws {
        let statuses: [WallpaperStatus] = [.pending, .pendingReview, .approved, .rejected]
        for status in statuses {
            let encoded = try JSONEncoder().encode(status)
            let decoded = try JSONDecoder().decode(WallpaperStatus.self, from: encoded)
            #expect(decoded == status)
        }
    }

    @Test("WallpaperStatus pendingReview has correct raw value")
    func pendingReviewRawValue() {
        #expect(WallpaperStatus.pendingReview.rawValue == "pending_review")
    }

    @Test("UserRole round-trips through Codable")
    func userRoleCodable() throws {
        for role in [UserRole.user, UserRole.admin] {
            let encoded = try JSONEncoder().encode(role)
            let decoded = try JSONDecoder().decode(UserRole.self, from: encoded)
            #expect(decoded == role)
        }
    }

    @Test("Category is CaseIterable with 8 cases")
    func categoryCount() {
        #expect(Category.allCases.count == 8)
    }

    @Test("SortOrder raw values match API contract")
    func sortOrderRawValues() {
        #expect(SortOrder.recent.rawValue == "recent")
        #expect(SortOrder.popular.rawValue == "popular")
    }
}
```

- [ ] Run tests: `cd app && swift test --filter TypedEnumTests`
- [ ] Commit: `feat(core): add typed enums for WallpaperStatus, UserRole, Category, SortOrder`

---

### D4: @Observable Migration — PauseController

**Files:**

```
app/Sources/ScreenSpace/Engine/PauseController.swift    (Modify)
app/Sources/ScreenSpace/AppState.swift                  (Modify)
app/Tests/ScreenSpaceTests/PauseControllerTests.swift   (Modify — migrate to Swift Testing)
```

#### D4.1 — Migrate PauseController

Current state: `ObservableObject`, `@Published var shouldPause`, Combine sinks, `Timer`, `DispatchQueue.main.asyncAfter`.

Target state: `@Observable`, no Combine, async notification sequences via `NotificationCenter.notifications(named:)`, `Task` for concurrency.

- [ ] Rewrite `PauseController.swift`:

```swift
import AppKit
import IOKit.ps

// PowerSourceProvider and LockStateProvider protocols remain unchanged
// (already defined in PauseController.swift — do not move them, they are tested)

@Observable
@MainActor
final class PauseController {
    private(set) var shouldPause: Bool = false

    private var config: AppConfig
    private let powerSource: PowerSourceProvider
    private let lockState: LockStateProvider
    private var isSleeping: Bool = false
    private var observationTask: Task<Void, Never>?

    init(
        config: AppConfig,
        powerSource: PowerSourceProvider = SystemPowerSource(),
        lockState: LockStateProvider = SystemLockState()
    ) {
        self.config = config
        self.powerSource = powerSource
        self.lockState = lockState
        startObserving()
        evaluate()
    }

    func updateConfig(_ config: AppConfig) {
        self.config = config
        evaluate()
    }

    func evaluate() {
        var pause = false
        if config.pauseOnBattery && powerSource.isOnBattery { pause = true }
        if ProcessInfo.processInfo.isLowPowerModeEnabled { pause = true }
        if lockState.isLocked { pause = true }
        if isSleeping { pause = true }
        shouldPause = pause
    }

    private func startObserving() {
        observationTask = Task { [weak self] in
            await withTaskGroup(of: Void.self) { group in
                group.addTask { await self?.observeSleep() }
                group.addTask { await self?.observeWake() }
                group.addTask { await self?.observePowerState() }
                group.addTask { await self?.observeScreenLocked() }
                group.addTask { await self?.observeScreenUnlocked() }
                group.addTask { await self?.observePowerSourcePeriodically() }
            }
        }
    }

    private func observeSleep() async {
        let notifications = NSWorkspace.shared.notificationCenter.notifications(
            named: NSWorkspace.willSleepNotification
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            isSleeping = true
            evaluate()
        }
    }

    private func observeWake() async {
        let notifications = NSWorkspace.shared.notificationCenter.notifications(
            named: NSWorkspace.didWakeNotification
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            isSleeping = false
            evaluate()
        }
    }

    private func observePowerState() async {
        let notifications = NotificationCenter.default.notifications(
            named: NSNotification.Name.NSProcessInfoPowerStateDidChange
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            evaluate()
        }
    }

    private func observeScreenLocked() async {
        let notifications = DistributedNotificationCenter.default().notifications(
            named: NSNotification.Name("com.apple.screenIsLocked")
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            evaluate()
        }
    }

    private func observeScreenUnlocked() async {
        let notifications = DistributedNotificationCenter.default().notifications(
            named: NSNotification.Name("com.apple.screenIsUnlocked")
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            // Small delay to let SystemLockState update its internal flag
            try? await Task.sleep(for: .milliseconds(100))
            evaluate()
        }
    }

    private func observePowerSourcePeriodically() async {
        // Poll every 30s for power source changes (no system notification available)
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(30))
            guard !Task.isCancelled else { return }
            evaluate()
        }
    }

    deinit {
        observationTask?.cancel()
    }
}
```

#### D4.2 — Update AppState to remove Combine

- [ ] Modify `AppState.swift`:
  - Remove `import Combine`
  - Remove `private var cancellables = Set<AnyCancellable>()`
  - Replace the `sink` pipeline on `pauseController.$shouldPause` with a Task-based observation. Since `PauseController` is now `@Observable`, use `withObservationTracking` or a `Task` polling loop in `init`:

```swift
// In AppState.init, replace the cancellables sink with:
Task { @MainActor [weak self] in
    while let self {
        let shouldPause = self.pauseController.shouldPause
        if shouldPause {
            self.engine.pauseAll()
        } else {
            self.engine.resumeAll()
        }
        // Wait for next change using withObservationTracking
        await withCheckedContinuation { continuation in
            withObservationTracking {
                _ = self.pauseController.shouldPause
            } onChange: {
                continuation.resume()
            }
        }
    }
}
```

#### D4.3 — Migrate PauseController tests to Swift Testing

- [ ] Rewrite `Tests/ScreenSpaceTests/PauseControllerTests.swift` using `@Test`/`@Suite`. Keep the same test coverage, just migrate the framework:

```swift
import Testing
@testable import ScreenSpace

// MockPowerSource and MockLockState move here — remove @unchecked Sendable
// and use @MainActor since tests run on MainActor
@MainActor
final class MockPowerSource: PowerSourceProvider {
    var isOnBattery = false
}

@MainActor
final class MockLockState: LockStateProvider {
    var isLocked = false
}

@Suite("PauseController")
@MainActor
struct PauseControllerTests {
    private func makeController(
        config: AppConfig = .default,
        powerSource: MockPowerSource = MockPowerSource(),
        lockState: MockLockState = MockLockState()
    ) -> PauseController {
        PauseController(config: config, powerSource: powerSource, lockState: lockState)
    }

    @Test("pauses when on battery and pauseOnBattery is enabled")
    func pausesOnBattery() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("does not pause on AC power")
    func doesNotPauseOnAC() {
        let power = MockPowerSource()
        power.isOnBattery = false
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(!controller.shouldPause)
    }

    @Test("does not pause when pauseOnBattery is disabled")
    func disabledPauseOnBattery() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = false
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(!controller.shouldPause)
    }

    @Test("pauses when screen is locked")
    func pausesWhenLocked() {
        let lock = MockLockState()
        lock.isLocked = true
        let controller = makeController(lockState: lock)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("pauses when multiple conditions are true")
    func multipleConditions() {
        let power = MockPowerSource()
        power.isOnBattery = true
        let lock = MockLockState()
        lock.isLocked = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power, lockState: lock)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("updateConfig re-evaluates pause state")
    func configUpdate() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        #expect(controller.shouldPause)

        var newConfig = config
        newConfig.pauseOnBattery = false
        controller.updateConfig(newConfig)
        #expect(!controller.shouldPause)
    }
}
```

- [ ] Run tests: `cd app && swift test --filter PauseControllerTests`
- [ ] Commit: `refactor(engine): migrate PauseController to @Observable, replace Combine with async sequences`

---

### D5: Sendable Conformances

**Files:**

```
app/Sources/ScreenSpace/Config/AppConfig.swift      (Modify)
app/Sources/ScreenSpace/Config/ConfigManager.swift  (Modify)
app/Sources/ScreenSpace/Config/PlaylistManager.swift (Modify)
app/Sources/ScreenSpace/API/APIModels.swift         (Modify — done in D3.2)
```

#### D5.1 — Add Sendable to value types

- [ ] Modify `AppConfig.swift` — add `Sendable` conformance:

```swift
struct AppConfig: Codable, Equatable, Sendable {
    // ... existing fields unchanged
}
```

`VideoGravityOption` enum: add `Sendable`:

```swift
enum VideoGravityOption: String, Codable, Sendable {
    // ...
}
```

- [ ] Modify `PlaylistManager.swift` — add `Sendable` to `PlaylistItem` and `Playlist`:

```swift
struct PlaylistItem: Codable, Identifiable, Equatable, Sendable { ... }
struct Playlist: Codable, Identifiable, Equatable, Sendable { ... }
```

#### D5.2 — Make ConfigManager an actor

Current: `final class ConfigManager: @unchecked Sendable` with unsynchronized mutable `config`.

Target: `actor ConfigManager` — removes `@unchecked Sendable`, proper isolation.

- [ ] Rewrite `ConfigManager.swift`:

```swift
import Foundation

actor ConfigManager {
    static let shared = ConfigManager()

    private let configURL: URL
    private(set) var config: AppConfig

    init(directory: URL? = nil) {
        let dir = directory ?? FileManager.default
            .urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
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
```

> **Breaking change:** All call sites of `ConfigManager` must `await` actor-isolated methods. Update `WallpaperEngine`, `AppState`, `App.swift` to use `await configManager.config` and `await configManager.update(...)`.

- [ ] Update call sites in `WallpaperEngine.swift`:
  - `setWallpaper(url:forDisplay:)` becomes `async`
  - `try? configManager.update { $0.lastPlayedURL = ... }` becomes `try? await configManager.update { ... }`
  - `configManager.config.videoGravity` becomes `await configManager.config.videoGravity` (or cache in local `let`)

- [ ] Update `AppState.init` to `await ConfigManager.shared` calls.

- [ ] Update `AppState.restoreLastWallpaper()` to `async` (or read config synchronously before the async boundary if possible by passing config as a value type).

#### D5.3 — Make PlaylistManager an actor

Current: `final class PlaylistManager: @unchecked Sendable` with unsynchronized mutable `playlists`.

- [ ] Rewrite `PlaylistManager.swift` as `actor PlaylistManager`.
  - All mutating methods (`create`, `delete`, `update`) become `async`.
  - `loadAll()` can remain `private` since it's called only from `init`.
  - Update call sites in `AppState.skipToNext()` to `await playlistManager.playlists`.

- [ ] Run `swift build` to catch call site errors: `cd app && swift build`
- [ ] Fix all call site errors from actor migration.
- [ ] Run tests: `cd app && swift test`
- [ ] Commit: `refactor(config): make ConfigManager and PlaylistManager actors, add Sendable to value types`

---

### D6: Concurrency Fixes

**Files:**

```
app/Sources/ScreenSpace/AppState.swift              (Modify)
app/Sources/ScreenSpace/Engine/WallpaperEngine.swift (Modify)
app/Sources/ScreenSpace/Helpers/CacheManager.swift  (Modify)
```

#### D6.1 — DispatchQueue -> Task

- [ ] Search for all `DispatchQueue.main.async` and `DispatchQueue.main.asyncAfter` in app source:

```bash
grep -r "DispatchQueue" app/Sources/ --include="*.swift"
```

- [ ] Replace each instance:
  - `DispatchQueue.main.async { ... }` -> `Task { @MainActor in ... }`
  - `DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { ... }` -> `try? await Task.sleep(for: .milliseconds(100))` (already handled in D4.1 for `observeScreenUnlocked`)

#### D6.2 — Task cancellation checks

- [ ] Review all `.task { }` view modifiers and `Task { }` blocks in Views. Add `guard !Task.isCancelled else { return }` between sequential async calls in long-running tasks. Specifically check:
  - `HomeView` (if it has a `.task` block)
  - `ExploreView`
  - `LibraryView`
  - Any view that makes multiple sequential API calls

#### D6.3 — APIClient @unchecked Sendable

`APIClient` is currently `final class: @unchecked Sendable`. Now that `NetworkProviding` exists:

- [ ] Add `NetworkProviding` as a dependency to `APIClient`:

```swift
final class APIClient: Sendable {
    let baseURL: String
    private let network: NetworkProviding
    private let keychain: KeychainProviding

    init(
        baseURL: String? = nil,
        network: NetworkProviding = LiveNetwork(),
        keychain: KeychainProviding = LiveKeychain()
    ) {
        self.baseURL = baseURL ?? "https://api.screenspace.app"
        self.network = network
        self.keychain = keychain
    }
    // ...
}
```

- [ ] Replace internal `session.data(for:)` calls with `network.data(for:)`.
- [ ] Replace `KeychainHelper.loadToken()` / `KeychainHelper.saveToken()` / `KeychainHelper.deleteToken()` calls with `keychain.load(key:)` / `keychain.save(key:data:)` / `keychain.delete(key:)`.
- [ ] Remove `@unchecked Sendable` annotation — plain `Sendable` now satisfies because all stored properties are `Sendable` (`String`, `NetworkProviding`, `KeychainProviding`).

- [ ] Run tests: `cd app && swift test`
- [ ] Commit: `refactor(concurrency): replace DispatchQueue with Task, add cancellation checks, remove @unchecked Sendable from APIClient`

---

## Track F: Linting Configuration

### F1: SwiftFormat Config

**Files:**

```
app/.swiftformat    (Create)
```

- [ ] Create `app/.swiftformat`:

```
--indent 4
--maxwidth 120
--wraparguments before-first
--wrapcollections before-first
--commas inline
--trimwhitespace always
--semicolons never
--importgrouping testable-bottom
--redundanttype inferred
--self remove
--stripunusedargs closure-only
```

- [ ] Run SwiftFormat dry-run to see what would change: `cd app && swiftformat --config .swiftformat --dryrun Sources/`
- [ ] Apply format: `cd app && swiftformat --config .swiftformat Sources/ Tests/`
- [ ] Verify build still passes: `cd app && swift build`
- [ ] Commit: `chore(tooling): add SwiftFormat config`

### F2: SwiftLint Config

**Files:**

```
app/.swiftlint.yml    (Create)
```

- [ ] Create `app/.swiftlint.yml`:

```yaml
included:
  - Sources
  - Tests

excluded:
  - Sources/ScreenSpace/generated  # future oapi-codegen output

opt_in_rules:
  - empty_count
  - closure_spacing
  - first_where
  - overridden_super_call
  - fatal_error_message
  - redundant_nil_coalescing

disabled_rules:
  - trailing_comma          # SwiftFormat handles this
  - opening_brace           # SwiftFormat handles this
  - identifier_name         # too restrictive for short variable names (e.g. `id`, `vm`)
  - line_length             # SwiftFormat enforces maxwidth

custom_rules:
  no_force_unwrap:
    name: "Force Unwrap"
    regex: "(?<![\"\\s])!(?=[.\\[\\(]|\\s*=)"
    message: "Avoid force unwrapping. Use guard let or if let."
    severity: warning

line_length:
  warning: 120
  error: 160

function_body_length:
  warning: 60
  error: 100

type_body_length:
  warning: 300
  error: 500

file_length:
  warning: 400
  error: 600
```

- [ ] Run SwiftLint to see violations: `cd app && swiftlint lint --config .swiftlint.yml Sources/`
- [ ] Fix all `error`-level violations. Leave `warning`-level violations for a separate pass.
- [ ] Commit: `chore(tooling): add SwiftLint config`

### F3: Lefthook integration

**Files:**

```
lefthook.yml    (Modify — add swift-format and swift-lint commands)
```

- [ ] Check if `lefthook.yml` exists at project root: `ls /path/to/screenspace/lefthook.yml`
- [ ] Add Swift hooks to `lefthook.yml` pre-commit section:

```yaml
pre-commit:
  parallel: true
  commands:
    swift-format:
      root: app/
      glob: "*.swift"
      run: swiftformat --config .swiftformat {staged_files}
      stage_fixed: true
    swift-lint:
      root: app/
      glob: "*.swift"
      run: swiftlint lint --strict --config .swiftlint.yml {staged_files}
```

- [ ] Verify hooks run: `lefthook run pre-commit`
- [ ] Commit: `chore(tooling): add Swift hooks to Lefthook pre-commit`

---

## Commit Order Summary

All commits should be pushed immediately after creation.

1. `feat(platform): add Platform protocol definitions` — D1.1
2. `feat(platform): add Live protocol implementations` — D1.2
3. `feat(platform): add Mock protocol implementations` — D1.3
4. `test(platform): add Swift Testing tests for mock protocol implementations` — D1.4
5. `feat(core): add EventLog actor with JSONL rotation and MockEventLog` — D2
6. `feat(core): add typed enums for WallpaperStatus, UserRole, Category, SortOrder` — D3
7. `refactor(engine): migrate PauseController to @Observable, replace Combine with async sequences` — D4
8. `refactor(config): make ConfigManager and PlaylistManager actors, add Sendable to value types` — D5
9. `refactor(concurrency): replace DispatchQueue with Task, add cancellation checks, remove @unchecked Sendable from APIClient` — D6
10. `chore(tooling): add SwiftFormat config` — F1
11. `chore(tooling): add SwiftLint config` — F2
12. `chore(tooling): add Swift hooks to Lefthook pre-commit` — F3

---

## Verification

After all tasks complete, run the full test suite and lint:

```bash
cd app && swift build 2>&1 | grep -E "error:|warning:" | head -20
cd app && swift test 2>&1 | tail -20
cd app && swiftlint lint --config .swiftlint.yml Sources/ 2>&1 | grep "error:" | wc -l
```

Expected:
- `swift build`: 0 errors
- `swift test`: all tests pass (including migrated XCTest suites and new Swift Testing suites)
- `swiftlint lint`: 0 errors (warnings acceptable)

---

## Notes for Implementors

**D4 async observation caveat:** `DistributedNotificationCenter` may not expose `notifications(named:)` as an `AsyncSequence` on macOS 15. If it does not, keep a `NotificationCenter.addObserver` pattern for those two distributed notifications only, wrapped in a `Task { @MainActor in self.evaluate() }` callback. Verify availability before writing.

**D5 actor migration call sites:** Making `ConfigManager` and `PlaylistManager` actors will cascade `async` requirements to callers. `WallpaperEngine.setWallpaper` and `AppState.restoreLastWallpaper` will need `async`. This is intentional — the async boundary makes data races impossible by construction.

**D5 ConfigManager.shared:** The `static let shared` pattern works on actors. `ConfigManager.shared` is still valid; callers just `await` its methods.

**D6 APIClient:** `KeychainHelper.swift` can be deleted after `LiveKeychain` is in use and all call sites are updated. Do not delete it in the same commit as adding `LiveKeychain` — keep it one step at a time so git bisect works.

**Swift Testing coexistence:** Swift Testing and XCTest can coexist in the same test target. Do not delete existing XCTest files. Migrate them to Swift Testing one file at a time as part of the tasks above. The `PauseControllerTests.swift` migration in D4.3 is the template for this pattern.
