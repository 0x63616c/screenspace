import Testing
@testable import ScreenSpace

@MainActor
struct FavoritesViewModelTests {
    @Test("starts in loading state")
    func startsLoading() {
        let vm = FavoritesViewModel(api: MockAPI(), eventLog: MockEventLog())
        #expect(vm.isLoading == true)
    }

    @Test("loads favorites and exits loading state")
    func loadsFavorites() async {
        let api = MockAPI()
        api.favoritesResponse = .success(TestFixtures.pagedWallpapers(3))
        let log = MockEventLog()
        let vm = FavoritesViewModel(api: api, eventLog: log)

        await vm.load()

        #expect(vm.favorites.count == 3)
        #expect(vm.isLoading == false)
        #expect(log.events.contains { $0.event == "favorites_loaded" })
    }

    @Test("sets hasMore when more pages exist")
    func setsHasMore() async {
        let api = MockAPI()
        api.favoritesResponse = .success(PagedWallpapers(
            items: Array(repeating: TestFixtures.wallpaperCard(), count: 20),
            total: 25, limit: 20, offset: 0
        ))
        let vm = FavoritesViewModel(api: api, eventLog: MockEventLog())
        await vm.load()

        #expect(vm.hasMore == true)
    }

    @Test("loadMore appends results")
    func loadMoreAppends() async {
        let api = MockAPI()
        api.favoritesResponse = .success(PagedWallpapers(
            items: Array(repeating: TestFixtures.wallpaperCard(), count: 20),
            total: 25, limit: 20, offset: 0
        ))
        let vm = FavoritesViewModel(api: api, eventLog: MockEventLog())
        await vm.load()

        api.favoritesResponse = .success(PagedWallpapers(
            items: (21 ... 25).map { TestFixtures.wallpaperCard(id: "w\($0)") },
            total: 25, limit: 20, offset: 20
        ))
        await vm.loadMore()

        #expect(vm.favorites.count == 25)
        #expect(vm.hasMore == false)
    }

    @Test("sets error on API failure")
    func setsError() async {
        let api = MockAPI()
        api.favoritesResponse = .failure(APIError.httpError(status: 401))
        let vm = FavoritesViewModel(api: api, eventLog: MockEventLog())

        await vm.load()

        #expect(vm.error != nil)
        #expect(vm.favorites.isEmpty)
    }
}
