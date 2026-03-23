import SwiftUI

struct AdminView: View {
    @State private var selectedTab = 0
    @State private var pendingWallpapers: [WallpaperResponse] = []
    @State private var users: [UserResponse] = []
    @State private var reports: [ReportResponse] = []
    @State private var isLoading = false
    @State private var errorMessage: String?

    private let api = APIClient()

    var body: some View {
        VStack(spacing: 0) {
            Picker("", selection: $selectedTab) {
                Text("Queue").tag(0)
                Text("Content").tag(1)
                Text("Users").tag(2)
                Text("Reports").tag(3)
            }
            .pickerStyle(.segmented)
            .padding()

            if let error = errorMessage {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
                    .padding(.horizontal)
            }

            switch selectedTab {
            case 0: queueView
            case 1: contentView
            case 2: usersView
            case 3: reportsView
            default: EmptyView()
            }
        }
        .task { await loadQueue() }
    }

    private var queueView: some View {
        List(pendingWallpapers) { wp in
            HStack {
                VStack(alignment: .leading) {
                    Text(wp.title).fontWeight(.medium)
                    Text("\(wp.resolution) - \(wp.format)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Approve") { Task { await approve(wp.id) } }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                Button("Reject") { Task { await reject(wp.id) } }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
            }
        }
        .overlay {
            if pendingWallpapers.isEmpty && !isLoading {
                Text("No pending wallpapers")
                    .foregroundStyle(.secondary)
            }
        }
    }

    private var contentView: some View {
        Text("All wallpapers (content management)")
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var usersView: some View {
        List(users) { user in
            HStack {
                VStack(alignment: .leading) {
                    Text(user.email)
                    Text(user.role)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if user.banned == true {
                    Button("Unban") { Task { await unban(user.id) } }
                        .controlSize(.small)
                } else {
                    Button("Ban") { Task { await ban(user.id) } }
                        .controlSize(.small)
                }
            }
        }
        .task { await loadUsers() }
    }

    private var reportsView: some View {
        List(reports) { report in
            HStack {
                VStack(alignment: .leading) {
                    Text(report.reason)
                    Text("Wallpaper: \(report.wallpaperID)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Dismiss") { Task { await dismissReport(report.id) } }
                    .controlSize(.small)
            }
        }
        .task { await loadReports() }
    }

    // MARK: - API Calls

    private func loadQueue() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let response = try await api.listQueue()
            pendingWallpapers = response.wallpapers
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func approve(_ id: String) async {
        do {
            try await api.approveWallpaper(id: id)
            pendingWallpapers.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func reject(_ id: String) async {
        do {
            try await api.rejectWallpaper(id: id, reason: "Rejected by admin")
            pendingWallpapers.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadUsers() async {
        do {
            let response = try await api.listUsers()
            users = response.users
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func ban(_ id: String) async {
        do {
            try await api.banUser(id: id)
            await loadUsers()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func unban(_ id: String) async {
        do {
            try await api.unbanUser(id: id)
            await loadUsers()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadReports() async {
        do {
            let response = try await api.listReports()
            reports = response.reports
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func dismissReport(_ id: String) async {
        do {
            try await api.dismissReport(id: id)
            reports.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
