import Testing
@testable import ScreenSpace

@MainActor
struct ExploreViewModelTests {
    @Test("loads categories from API")
    func loadsCategories() async {
        let api = MockAPI()
        api.categoriesResponse = .success([.nature, .abstract, .urban])
        let vm = ExploreViewModel(api: api, eventLog: MockEventLog())

        await vm.loadCategories()

        #expect(vm.categories == [.nature, .abstract, .urban])
    }

    @Test("falls back to all categories on API error")
    func fallsBackToAllCategories() async {
        let api = MockAPI()
        api.categoriesResponse = .failure(APIError.httpError(status: 503))
        let vm = ExploreViewModel(api: api, eventLog: MockEventLog())

        await vm.loadCategories()

        #expect(vm.categories == Category.allCases)
    }

    @Test("selectCategory loads results and sets category")
    func selectCategoryLoadsResults() async {
        let api = MockAPI()
        api.listWallpapersResponse = .success(TestFixtures.pagedWallpapers(3))
        let log = MockEventLog()
        let vm = ExploreViewModel(api: api, eventLog: log)

        await vm.selectCategory(.nature)

        #expect(vm.selectedCategory == .nature)
        #expect(vm.results.count == 3)
        #expect(vm.isLoading == false)
        #expect(log.events.contains { $0.event == "category_browsed" })
    }

    @Test("search clears category and loads results")
    func searchClearsCategory() async {
        let api = MockAPI()
        api.listWallpapersResponse = .success(TestFixtures.pagedWallpapers(2))
        let log = MockEventLog()
        let vm = ExploreViewModel(api: api, eventLog: log)
        vm.selectedCategory = .nature
        vm.searchQuery = "ocean"

        await vm.search()

        #expect(vm.selectedCategory == nil)
        #expect(vm.results.count == 2)
        #expect(log.events.contains { $0.event == "search_performed" })
    }

    @Test("search does nothing with empty query")
    func searchIgnoresEmptyQuery() async {
        let api = MockAPI()
        let vm = ExploreViewModel(api: api, eventLog: MockEventLog())
        vm.searchQuery = ""

        await vm.search()

        #expect(vm.results.isEmpty)
        #expect(api.listWallpapersCalled == false)
    }

    @Test("clearCategory resets state")
    func clearCategoryResetsState() async {
        let api = MockAPI()
        api.listWallpapersResponse = .success(TestFixtures.pagedWallpapers(3))
        let vm = ExploreViewModel(api: api, eventLog: MockEventLog())
        await vm.selectCategory(.nature)

        vm.clearCategory()

        #expect(vm.selectedCategory == nil)
        #expect(vm.results.isEmpty)
    }
}
