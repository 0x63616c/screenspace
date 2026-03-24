import Testing
@testable import ScreenSpace

@MainActor
struct AdminViewModelTests {
    @Test("loadQueue populates pendingWallpapers")
    func loadQueuePopulates() async {
        let api = MockAPI()
        api.queueResponse = .success(TestFixtures.pagedWallpapers(3))
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())

        await vm.loadQueue()

        #expect(vm.pendingWallpapers.count == 3)
        #expect(vm.isLoading == false)
    }

    @Test("approve removes wallpaper from queue")
    func approveRemovesFromQueue() async {
        let api = MockAPI()
        api.queueResponse = .success(TestFixtures.pagedWallpapers(2))
        api.approveResponse = .success(())
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())
        await vm.loadQueue()

        await vm.approve(id: "w1")

        #expect(vm.pendingWallpapers.count == 1)
        #expect(vm.pendingWallpapers.first?.id != "w1")
    }

    @Test("reject removes wallpaper from queue")
    func rejectRemovesFromQueue() async {
        let api = MockAPI()
        api.queueResponse = .success(TestFixtures.pagedWallpapers(2))
        api.rejectResponse = .success(())
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())
        await vm.loadQueue()

        await vm.reject(id: "w1")

        #expect(vm.pendingWallpapers.count == 1)
    }

    @Test("ban calls API and reloads users")
    func banUpdatesUsers() async {
        let api = MockAPI()
        api.usersResponse = .success(PagedUsers(
            items: [TestFixtures.userInfo(id: "u1"), TestFixtures.userInfo(id: "u2")],
            total: 2, limit: 50, offset: 0
        ))
        api.banResponse = .success(())
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())

        await vm.ban(id: "u1")

        #expect(api.banCalled == true)
        #expect(api.loadUsersCalled == true)
    }

    @Test("dismissReport removes from reports list")
    func dismissReportRemoves() async {
        let api = MockAPI()
        let report = ReportInfo(id: "r1", wallpaperID: "w1", reporterID: "u1", reason: "Spam", createdAt: nil)
        api.reportsResponse = .success(PagedReports(items: [report], total: 1, limit: 50, offset: 0))
        api.dismissReportResponse = .success(())
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())
        await vm.loadReports()

        await vm.dismissReport(id: "r1")

        #expect(vm.reports.isEmpty)
    }

    @Test("sets error on API failure")
    func setsErrorOnFailure() async {
        let api = MockAPI()
        api.queueResponse = .failure(APIError.forbidden)
        let vm = AdminViewModel(api: api, eventLog: MockEventLog())

        await vm.loadQueue()

        #expect(vm.error != nil)
    }
}
