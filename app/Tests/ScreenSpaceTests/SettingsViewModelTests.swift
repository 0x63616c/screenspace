import Testing
@testable import ScreenSpace

@MainActor
struct SettingsViewModelTests {
    private func makeVM() -> (SettingsViewModel, MockConfigStore, MockCache) {
        let store = MockConfigStore()
        let cache = MockCache()
        let vm = SettingsViewModel(configStore: store, cache: cache, eventLog: MockEventLog())
        return (vm, store, cache)
    }

    @Test("initializes from config store")
    func initializesFromStore() {
        let (vm, store, _) = makeVM()
        #expect(vm.config == store.storedConfig)
        #expect(vm.serverURL == store.storedConfig.serverURL)
    }

    @Test("updateConfig persists changes")
    func updateConfigPersists() {
        let (vm, store, _) = makeVM()

        vm.updateConfig { $0.pauseOnBattery = false }

        #expect(store.storedConfig.pauseOnBattery == false)
    }

    @Test("commitServerURL saves to config")
    func commitServerURLSaves() {
        let (vm, store, _) = makeVM()
        vm.serverURL = "http://localhost:9090"

        vm.commitServerURL()

        #expect(store.storedConfig.serverURL == "http://localhost:9090")
    }

    @Test("clearCache resets cacheSize to zero")
    func clearCacheResetsSizeToZero() {
        let (vm, _, cache) = makeVM()
        cache.currentSize = 500

        vm.clearCache()

        #expect(vm.cacheSize == 0)
        #expect(cache.clearCacheCalled == true)
    }

    @Test("formatSize formats megabytes correctly")
    func formatSizeMB() {
        let (vm, _, _) = makeVM()
        #expect(vm.formatSize(512) == "512 MB")
    }

    @Test("formatSize converts to GB for large values")
    func formatSizeGB() {
        let (vm, _, _) = makeVM()
        #expect(vm.formatSize(2048) == "2.0 GB")
    }
}
