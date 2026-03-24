import Foundation
import Testing
@testable import ScreenSpace

@MainActor
struct DetailViewModelTests {
    private func makeVM(
        api: APIProviding? = nil,
        provider: WallpaperProviding? = nil,
        cache: CacheProviding? = nil
    ) -> DetailViewModel {
        DetailViewModel(
            wallpaper: TestFixtures.wallpaperDetail(),
            api: api ?? MockAPI(),
            wallpaperProvider: provider ?? MockWallpaperProvider(),
            cache: cache ?? MockCache(),
            eventLog: MockEventLog()
        )
    }

    @Test("setAsWallpaper uses cache when available")
    func usesCache() async {
        let cache = MockCache()
        let cachedURL = URL(fileURLWithPath: "/cache/w1.mp4")
        cache.cachedURLs["w1"] = cachedURL
        let provider = MockWallpaperProvider()
        let vm = makeVM(provider: provider, cache: cache)

        await vm.setAsWallpaper()

        #expect(provider.setCalls.first?.url == cachedURL)
        #expect(vm.isDownloading == false)
    }

    @Test("toggleFavorite updates isFavorited")
    func toggleFavoriteUpdates() async {
        let api = MockAPI()
        api.toggleFavoriteResponse = .success(true)
        let vm = makeVM(api: api)

        await vm.toggleFavorite(isLoggedIn: true)

        #expect(vm.isFavorited == true)
    }

    @Test("toggleFavorite shows login error when logged out")
    func toggleFavoriteRequiresLogin() async {
        let vm = makeVM()

        await vm.toggleFavorite(isLoggedIn: false)

        #expect(vm.error != nil)
        #expect(vm.isFavorited == false)
    }

    @Test("submitReport clears sheet on success")
    func submitReportClearsSheet() async {
        let api = MockAPI()
        api.reportResponse = .success(())
        let vm = makeVM(api: api)
        vm.reportReason = "Inappropriate content"
        vm.showReportSheet = true

        await vm.submitReport(isLoggedIn: true)

        #expect(vm.showReportSheet == false)
        #expect(vm.reportReason.isEmpty)
    }

    @Test("submitReport does nothing with empty reason")
    func submitReportIgnoresEmptyReason() async {
        let api = MockAPI()
        let vm = makeVM(api: api)
        vm.reportReason = "   "

        await vm.submitReport(isLoggedIn: true)

        #expect(api.reportCalled == false)
    }

    @Test("formattedSize formats bytes to MB")
    func formattedSizeFormatsMB() {
        let vm = makeVM()
        #expect(vm.formattedSize == "85.0 MB")
    }

    @Test("formattedDuration formats seconds")
    func formattedDurationFormatsSeconds() {
        let vm = makeVM()
        #expect(vm.formattedDuration == "30s")
    }
}
