import ServiceManagement
import SwiftUI

private extension View {
    func settingsTabStyle() -> some View {
        formStyle(.grouped)
            .scrollContentBackground(.hidden)
            .frame(maxHeight: .infinity, alignment: .top)
    }
}

struct SettingsView: View {
    @Environment(AppState.self) var appState
    @State private var viewModel: SettingsViewModel?
    @State private var showLogin = false

    var body: some View {
        Group {
            if let viewModel {
                SettingsContentView(
                    viewModel: viewModel,
                    appState: appState,
                    showLogin: $showLogin
                )
            } else {
                ProgressView()
            }
        }
        .frame(width: 500, height: 400)
        .padding()
        .task {
            if viewModel == nil {
                viewModel = SettingsViewModel(
                    configStore: appState.configStore,
                    cache: appState.cache,
                    eventLog: appState.eventLog,
                    playlistManager: appState.playlistManager,
                    onLogout: { [weak appState] in
                        appState?.logout()
                    }
                )
            }
        }
    }
}

private struct SettingsContentView: View {
    @Bindable var viewModel: SettingsViewModel
    let appState: AppState
    @Binding var showLogin: Bool

    var body: some View {
        TabView {
            generalTab.tabItem { Label("General", systemImage: "gear") }
            playbackTab.tabItem { Label("Playback", systemImage: "play.circle") }
            storageTab.tabItem { Label("Storage", systemImage: "externaldrive") }
            displaysTab.tabItem { Label("Displays", systemImage: "display.2") }
            accountTab.tabItem { Label("Account", systemImage: "person.circle") }
        }
        .errorAlert(message: Binding(
            get: { viewModel.error },
            set: { viewModel.error = $0 }
        ))
    }

    private var generalTab: some View {
        Form {
            Toggle("Launch at login", isOn: Binding(
                get: { viewModel.config.launchAtLogin },
                set: { viewModel.setLaunchAtLogin($0) }
            ))

            TextField("Server URL", text: $viewModel.serverURL)
                .onSubmit { viewModel.commitServerURL() }

            Text("Version \(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0.1.0-dev")")
                .font(Typography.meta)
                .foregroundStyle(.secondary)
        }
        .settingsTabStyle()
    }

    private var playbackTab: some View {
        Form {
            Toggle("Pause on battery", isOn: Binding(
                get: { viewModel.config.pauseOnBattery },
                set: { newValue in viewModel.updateConfig { $0.pauseOnBattery = newValue } }
            ))

            Toggle("Pause when fullscreen app active", isOn: Binding(
                get: { viewModel.config.pauseOnFullscreen },
                set: { newValue in viewModel.updateConfig { $0.pauseOnFullscreen = newValue } }
            ))

            Picker("Video scaling", selection: Binding(
                get: { viewModel.config.videoGravity },
                set: { newValue in viewModel.updateConfig { $0.videoGravity = newValue } }
            )) {
                Text("Fill (crop edges)").tag(VideoGravityOption.resizeAspectFill)
                Text("Fit (letterbox)").tag(VideoGravityOption.resizeAspect)
            }
        }
        .settingsTabStyle()
    }

    private var storageTab: some View {
        Form {
            HStack {
                Text("Cache size")
                Spacer()
                Text(viewModel.formatSize(viewModel.cacheSize))
                    .foregroundStyle(.secondary)
            }

            Stepper("Cache limit: \(viewModel.formatSize(viewModel.config.cacheSizeLimitMB))", value: Binding(
                get: { viewModel.config.cacheSizeLimitMB },
                set: { newValue in viewModel.updateConfig { $0.cacheSizeLimitMB = newValue } }
            ), in: 1024 ... 20480, step: 1024)

            Button("Clear Cache") {
                viewModel.clearCache()
            }
            .accessibilityLabel("Clear cache")
            .accessibilityHint("Removes all downloaded wallpapers from cache")
        }
        .settingsTabStyle()
    }

    private var displaysTab: some View {
        Form {
            Text("Per-Display Wallpaper Assignment")
                .font(Typography.label)

            ForEach(NSScreen.screens, id: \.self) { screen in
                let displayID = DisplayIdentifier.stableID(for: screen)

                HStack {
                    Text(screen.localizedName)
                        .font(.body)
                    Spacer()

                    Picker("Playlist", selection: Binding(
                        get: { viewModel.config.screenAssignments[displayID] ?? "" },
                        set: { newValue in
                            viewModel.updateConfig { config in
                                config.screenAssignments[displayID] = newValue.isEmpty ? nil : newValue
                            }
                        }
                    )) {
                        Text("None").tag("")
                        ForEach(viewModel.playlists) { playlist in
                            Text(playlist.name).tag(playlist.id)
                        }
                    }
                    .frame(width: 150)
                }
            }
        }
        .settingsTabStyle()
    }

    private var accountTab: some View {
        Form {
            if let user = appState.currentUser {
                LabeledContent("Email", value: user.email)
                LabeledContent("Role", value: user.role.rawValue.capitalized)
                Button("Log Out") {
                    viewModel.logout()
                }
                .buttonStyle(.bordered)
            } else {
                Text("Log in to upload and favorite wallpapers.")
                    .foregroundStyle(.secondary)
                Button("Log In") {
                    showLogin = true
                }
                .buttonStyle(.borderedProminent)
            }
        }
        .settingsTabStyle()
        .sheet(isPresented: $showLogin) {
            LoginView()
        }
    }
}
