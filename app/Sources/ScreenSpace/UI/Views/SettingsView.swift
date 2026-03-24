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
    @State private var config: AppConfig = .default
    @State private var cacheSize = 0
    @State private var serverURL: String = AppConfig.defaultServerURL
    @State private var showLogin = false
    @State private var settingsError: String?
    @State private var playlists: [Playlist] = []

    var body: some View {
        TabView {
            generalTab.tabItem { Label("General", systemImage: "gear") }
            playbackTab.tabItem { Label("Playback", systemImage: "play.circle") }
            storageTab.tabItem { Label("Storage", systemImage: "externaldrive") }
            displaysTab.tabItem { Label("Displays", systemImage: "display.2") }
            accountTab.tabItem { Label("Account", systemImage: "person.circle") }
        }
        .frame(width: 500, height: 400)
        .padding()
        .errorAlert(message: $settingsError)
        .task {
            config = await appState.configManager.config
            serverURL = config.serverURL
            playlists = await appState.playlistManager.playlists
            cacheSize = appState.cache.currentSizeMB()
        }
    }

    private var generalTab: some View {
        Form {
            Toggle("Launch at login", isOn: Binding(
                get: { config.launchAtLogin },
                set: { newValue in
                    config.launchAtLogin = newValue
                    do {
                        if newValue {
                            try SMAppService.mainApp.register()
                        } else {
                            try SMAppService.mainApp.unregister()
                        }
                    } catch {
                        settingsError = "Failed to \(newValue ? "enable" : "disable") launch at login: \(error.localizedDescription)"
                    }
                    saveConfig()
                }
            ))

            TextField("Server URL", text: $serverURL)
                .onSubmit {
                    config.serverURL = serverURL
                    saveConfig()
                }

            Text("Version \(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0.1.0-dev")")
                .font(Typography.meta)
                .foregroundStyle(.secondary)
        }
        .settingsTabStyle()
    }

    private var playbackTab: some View {
        Form {
            Toggle("Pause on battery", isOn: Binding(
                get: { config.pauseOnBattery },
                set: { config.pauseOnBattery = $0
                    saveConfig()
                }
            ))

            Toggle("Pause when fullscreen app active", isOn: Binding(
                get: { config.pauseOnFullscreen },
                set: { config.pauseOnFullscreen = $0
                    saveConfig()
                }
            ))

            Picker("Video scaling", selection: Binding(
                get: { config.videoGravity },
                set: { config.videoGravity = $0
                    saveConfig()
                }
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
                Text(formatSize(cacheSize))
                    .foregroundStyle(.secondary)
            }

            Stepper("Cache limit: \(formatSize(config.cacheSizeLimitMB))", value: Binding(
                get: { config.cacheSizeLimitMB },
                set: { config.cacheSizeLimitMB = $0
                    saveConfig()
                }
            ), in: 1024 ... 20480, step: 1024)

            Button("Clear Cache") {
                appState.cache.clearCache()
                cacheSize = 0
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
                        get: { config.screenAssignments[displayID] ?? "" },
                        set: { newValue in
                            config.screenAssignments[displayID] = newValue.isEmpty ? nil : newValue
                            saveConfig()
                        }
                    )) {
                        Text("None").tag("")
                        ForEach(playlists) { playlist in
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
                    appState.logout()
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

    private func saveConfig() {
        let snapshot = config
        Task { try? await appState.configManager.setConfig(snapshot) }
    }

    private func formatSize(_ mb: Int) -> String {
        if mb >= 1024 {
            return String(format: "%.1f GB", Double(mb) / 1024.0)
        }
        return "\(mb) MB"
    }
}
