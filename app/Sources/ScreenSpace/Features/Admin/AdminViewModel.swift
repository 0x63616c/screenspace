import Foundation

@Observable
@MainActor
final class AdminViewModel {
    private let api: APIProviding
    private let eventLog: EventLogging

    enum AdminTab: String, CaseIterable {
        case queue = "Queue"
        case content = "Content"
        case users = "Users"
        case reports = "Reports"
    }

    var selectedTab: AdminTab = .queue
    var pendingWallpapers: [WallpaperCardData] = []
    var allWallpapers: [WallpaperCardData] = []
    var users: [UserInfo] = []
    var reports: [ReportInfo] = []
    var isLoading = false
    var error: String?

    init(api: APIProviding, eventLog: EventLogging) {
        self.api = api
        self.eventLog = eventLog
    }

    func loadQueue() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let response = try await api.listQueue(limit: 50, offset: 0)
            pendingWallpapers = response.items
        } catch {
            self.error = error.localizedDescription
        }
    }

    func approve(id: String) async {
        do {
            try await api.approveWallpaper(id: id)
            pendingWallpapers.removeAll { $0.id == id }
            eventLog.log("wallpaper_set", data: ["action": "approved", "id": id])
        } catch {
            self.error = error.localizedDescription
        }
    }

    func reject(id: String) async {
        do {
            try await api.rejectWallpaper(id: id, reason: "Rejected by admin")
            pendingWallpapers.removeAll { $0.id == id }
            eventLog.log("wallpaper_set", data: ["action": "rejected", "id": id])
        } catch {
            self.error = error.localizedDescription
        }
    }

    func loadUsers() async {
        do {
            let response = try await api.listUsers(query: nil, limit: 50, offset: 0)
            users = response.items
        } catch {
            self.error = error.localizedDescription
        }
    }

    func ban(id: String) async {
        do {
            try await api.banUser(id: id)
            await loadUsers()
        } catch {
            self.error = error.localizedDescription
        }
    }

    func unban(id: String) async {
        do {
            try await api.unbanUser(id: id)
            await loadUsers()
        } catch {
            self.error = error.localizedDescription
        }
    }

    func promote(id: String) async {
        do {
            try await api.promoteUser(id: id)
            await loadUsers()
        } catch {
            self.error = error.localizedDescription
        }
    }

    func loadReports() async {
        do {
            let response = try await api.listReports(limit: 50, offset: 0)
            reports = response.items
        } catch {
            self.error = error.localizedDescription
        }
    }

    func dismissReport(id: String) async {
        do {
            try await api.dismissReport(id: id)
            reports.removeAll { $0.id == id }
        } catch {
            self.error = error.localizedDescription
        }
    }
}
