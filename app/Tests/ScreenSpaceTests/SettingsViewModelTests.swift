import Testing
@testable import ScreenSpace

@MainActor
struct SettingsViewModelTests {
    struct TestHarness {
        let vm: SettingsViewModel
        let store: MockConfigStore
        let cache: MockCache
        var logoutCalled = false
    }

    private func makeVM() -> TestHarness {
        let store = MockConfigStore()
        let cache = MockCache()
        return TestHarness(vm: SettingsViewModel(
            configStore: store,
            cache: cache,
            eventLog: MockEventLog(),
            playlistManager: MockPlaylistManager(),
            onLogout: {}
        ), store: store, cache: cache)
    }

    @Test("initializes from config store")
    func initializesFromStore() {
        let h = makeVM()
        #expect(h.vm.config == h.store.storedConfig)
        #expect(h.vm.serverURL == h.store.storedConfig.serverURL)
    }

    @Test("updateConfig persists changes")
    func updateConfigPersists() {
        let h = makeVM()

        h.vm.updateConfig { $0.pauseOnBattery = false }

        #expect(h.store.storedConfig.pauseOnBattery == false)
    }

    @Test("commitServerURL saves to config")
    func commitServerURLSaves() {
        let h = makeVM()
        h.vm.serverURL = "http://localhost:9090"

        h.vm.commitServerURL()

        #expect(h.store.storedConfig.serverURL == "http://localhost:9090")
    }

    @Test("clearCache resets cacheSize to zero")
    func clearCacheResetsSizeToZero() {
        let h = makeVM()
        h.cache.currentSize = 500

        h.vm.clearCache()

        #expect(h.vm.cacheSize == 0)
        #expect(h.cache.clearCacheCalled == true)
    }

    @Test("formatSize formats megabytes correctly")
    func formatSizeMB() {
        let h = makeVM()
        #expect(h.vm.formatSize(512) == "512 MB")
    }

    @Test("formatSize converts to GB for large values")
    func formatSizeGB() {
        let h = makeVM()
        #expect(h.vm.formatSize(2048) == "2.0 GB")
    }
}
