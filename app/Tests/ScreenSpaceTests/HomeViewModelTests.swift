import Testing
@testable import ScreenSpace

@MainActor
struct HomeViewModelTests {
    @Test("starts in loading state")
    func startsLoading() {
        let vm = HomeViewModel(api: MockAPI(), eventLog: MockEventLog())
        #expect(vm.isLoading == true)
        #expect(vm.popular.isEmpty)
        #expect(vm.recent.isEmpty)
    }

    @Test("loads popular and recent wallpapers")
    func loadsWallpapers() async {
        let api = MockAPI()
        api.popularResponse = .success(TestFixtures.pagedWallpapers(5))
        api.recentResponse = .success(TestFixtures.pagedWallpapers(10))
        let log = MockEventLog()
        let vm = HomeViewModel(api: api, eventLog: log)

        await vm.load()

        #expect(vm.popular.count == 5)
        #expect(vm.recent.count == 10)
        #expect(vm.featured?.id == "w1")
        #expect(vm.isLoading == false)
        #expect(vm.error == nil)
        #expect(log.events.contains { $0.event == "wallpapers_loaded" })
    }

    @Test("sets error message on API failure")
    func setsErrorOnFailure() async {
        let api = MockAPI()
        api.popularResponse = .failure(APIError.httpError(status: 500))
        let vm = HomeViewModel(api: api, eventLog: MockEventLog())

        await vm.load()

        #expect(vm.error != nil)
        #expect(vm.popular.isEmpty)
        #expect(vm.isLoading == false)
    }

    @Test("fetchDetail sets selectedDetail")
    func fetchDetailSetsSelection() async {
        let api = MockAPI()
        let detail = TestFixtures.wallpaperDetail(id: "w1")
        api.wallpaperDetailResponse = .success(detail)
        let vm = HomeViewModel(api: api, eventLog: MockEventLog())

        await vm.fetchDetail(id: "w1")

        #expect(vm.selectedDetail?.id == "w1")
    }

    @Test("clearDetail removes selection")
    func clearDetailRemovesSelection() async {
        let api = MockAPI()
        api.wallpaperDetailResponse = .success(TestFixtures.wallpaperDetail())
        let vm = HomeViewModel(api: api, eventLog: MockEventLog())
        await vm.fetchDetail(id: "w1")

        vm.clearDetail()

        #expect(vm.selectedDetail == nil)
    }
}
