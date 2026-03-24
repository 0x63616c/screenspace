import SwiftUI

struct AdminView: View {
    @Environment(AppState.self) private var appState
    @State private var viewModel: AdminViewModel?

    var body: some View {
        Group {
            if let viewModel {
                AdminContentView(viewModel: viewModel)
            } else {
                ProgressView()
            }
        }
        .task {
            if viewModel == nil {
                viewModel = AdminViewModel(api: appState.apiService, eventLog: appState.eventLog)
            }
        }
    }
}

private struct AdminContentView: View {
    @Bindable var viewModel: AdminViewModel

    var body: some View {
        VStack(spacing: 0) {
            // Tab bar
            HStack(spacing: 0) {
                ForEach(AdminViewModel.AdminTab.allCases, id: \.self) { tab in
                    Button(action: { viewModel.selectedTab = tab }, label: {
                        Text(tab.rawValue)
                            .font(Typography.cardTitle)
                            .fontWeight(viewModel.selectedTab == tab ? .semibold : .regular)
                            .foregroundStyle(viewModel.selectedTab == tab ? .primary : .secondary)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                    })
                    .buttonStyle(.plain)
                    .accessibilityLabel("\(tab.rawValue) tab")
                    .accessibilityValue(viewModel.selectedTab == tab ? "Selected" : "")
                    .accessibilityAddTraits(viewModel.selectedTab == tab ? [.isButton, .isSelected] : .isButton)
                    .background {
                        if viewModel.selectedTab == tab {
                            Capsule()
                                .fill(.quaternary)
                        }
                    }
                }
            }
            .padding()

            if let error = viewModel.error {
                Text(error)
                    .foregroundStyle(.red)
                    .font(Typography.meta)
                    .padding(.horizontal)
            }

            switch viewModel.selectedTab {
            case .queue: queueView
            case .content: contentView
            case .users: usersView
            case .reports: reportsView
            }
        }
        .task { await viewModel.loadQueue() }
    }

    private var queueView: some View {
        List(viewModel.pendingWallpapers) { wp in
            HStack {
                VStack(alignment: .leading) {
                    Text(wp.title).fontWeight(.medium)
                    Text("\(wp.width)x\(wp.height) - \(wp.durationLabel)")
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Approve") { Task { await viewModel.approve(id: wp.id) } }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                    .accessibilityLabel("Approve \(wp.title)")
                Button("Reject") { Task { await viewModel.reject(id: wp.id) } }
                    .buttonStyle(.bordered)
                    .tint(.red)
                    .controlSize(.small)
                    .accessibilityLabel("Reject \(wp.title)")
            }
        }
        .overlay {
            if viewModel.pendingWallpapers.isEmpty && !viewModel.isLoading {
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
        List(viewModel.users) { user in
            HStack {
                VStack(alignment: .leading) {
                    Text(user.email)
                    Text(user.role.rawValue)
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if user.banned {
                    Button("Unban") { Task { await viewModel.unban(id: user.id) } }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .accessibilityLabel("Unban \(user.email)")
                } else {
                    Button("Ban") { Task { await viewModel.ban(id: user.id) } }
                        .buttonStyle(.bordered)
                        .tint(.red)
                        .controlSize(.small)
                        .accessibilityLabel("Ban \(user.email)")
                        .accessibilityHint("Prevents this user from accessing the service")
                }
            }
        }
        .task { await viewModel.loadUsers() }
    }

    private var reportsView: some View {
        List(viewModel.reports) { report in
            HStack {
                VStack(alignment: .leading) {
                    Text(report.reason)
                    Text("Wallpaper: \(report.wallpaperID)")
                        .font(Typography.meta)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Button("Dismiss") { Task { await viewModel.dismissReport(id: report.id) } }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .accessibilityLabel("Dismiss report for wallpaper \(report.wallpaperID)")
            }
        }
        .task { await viewModel.loadReports() }
    }
}
