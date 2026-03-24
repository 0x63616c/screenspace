import SwiftUI

struct AdminView: View {
    enum AdminTab: String, CaseIterable {
        case queue = "Queue"
        case content = "Content"
        case users = "Users"
        case reports = "Reports"
    }

    @State private var selectedTab: AdminTab = .queue
    @State private var pendingWallpapers: [WallpaperResponse] = []
    @State private var users: [UserResponse] = []
    @State private var reports: [ReportResponse] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @Environment(AppState.self) private var appState

    var body: some View {
        VStack(spacing: 0) {
            // Tab bar
            HStack(spacing: 0) {
                ForEach(AdminTab.allCases, id: \.self) { tab in
                    Button(action: { selectedTab = tab }, label: {
                        Text(tab.rawValue)
                            .font(Typography.cardTitle)
                            .fontWeight(selectedTab == tab ? .semibold : .regular)
                            .foregroundStyle(selectedTab == tab ? .primary : .secondary)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                    })
                    .buttonStyle(.plain)
                    .accessibilityLabel("\(tab.rawValue) tab")
                    .accessibilityValue(selectedTab == tab ? "Selected" : "")
                    .accessibilityAddTraits(selectedTab == tab ? [.isButton, .isSelected] : .isButton)
                    .background {
                        if selectedTab == tab {
                            Capsule()
                                .fill(.quaternary)
                        }
                    }
                }
            }
            .padding()

            if let error = errorMessage {
                Text(error)
                    .foregroundStyle(.red)
                    .font(Typography.meta)
                    .padding(.horizontal)
            }

            switch selectedTab {
            case .queue: queueView
            case .content: contentView
            case .users: usersView
            case .reports: reportsView
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
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Approve") { Task { await approve(wp.id) } }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                    .accessibilityLabel("Approve \(wp.title)")
                Button("Reject") { Task { await reject(wp.id) } }
                    .buttonStyle(.bordered)
                    .tint(.red)
                    .controlSize(.small)
                    .accessibilityLabel("Reject \(wp.title)")
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
                    Text(user.role.rawValue)
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if user.banned == true {
                    Button("Unban") { Task { await unban(user.id) } }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .accessibilityLabel("Unban \(user.email)")
                } else {
                    Button("Ban") { Task { await ban(user.id) } }
                        .buttonStyle(.bordered)
                        .tint(.red)
                        .controlSize(.small)
                        .accessibilityLabel("Ban \(user.email)")
                        .accessibilityHint("Prevents this user from accessing the service")
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
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Dismiss") { Task { await dismissReport(report.id) } }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .accessibilityLabel("Dismiss report for wallpaper \(report.wallpaperID)")
            }
        }
        .task { await loadReports() }
    }

    // MARK: - API Calls

    private func loadQueue() async {
        isLoading = true
        defer { isLoading = false }
        do {
            let response = try await appState.api.listQueue()
            pendingWallpapers = response.wallpapers
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func approve(_ id: String) async {
        do {
            try await appState.api.approveWallpaper(id: id)
            pendingWallpapers.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func reject(_ id: String) async {
        do {
            try await appState.api.rejectWallpaper(id: id, reason: "Rejected by admin")
            pendingWallpapers.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadUsers() async {
        do {
            let response = try await appState.api.listUsers()
            users = response.users
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func ban(_ id: String) async {
        do {
            try await appState.api.banUser(id: id)
            await loadUsers()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func unban(_ id: String) async {
        do {
            try await appState.api.unbanUser(id: id)
            await loadUsers()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func loadReports() async {
        do {
            let response = try await appState.api.listReports()
            reports = response.reports
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    private func dismissReport(_ id: String) async {
        do {
            try await appState.api.dismissReport(id: id)
            reports.removeAll { $0.id == id }
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
