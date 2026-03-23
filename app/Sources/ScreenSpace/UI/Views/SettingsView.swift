import SwiftUI
import ServiceManagement

struct SettingsView: View {
    @Environment(AppState.self) var appState
    @State private var config = ConfigManager.shared.config
    @State private var cacheSize = CacheManager.shared.currentCacheSizeMB()
    @State private var serverURL: String = ConfigManager.shared.config.serverURL
    @State private var showLogin = false

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
    }

    private var generalTab: some View {
        Form {
            Toggle("Launch at login", isOn: Binding(
                get: { config.launchAtLogin },
                set: { newValue in
                    config.launchAtLogin = newValue
                    if newValue {
                        try? SMAppService.mainApp.register()
                    } else {
                        try? SMAppService.mainApp.unregister()
                    }
                    saveConfig()
                }
            ))

            TextField("Server URL", text: $serverURL)
                .onSubmit {
                    config.serverURL = serverURL
                    saveConfig()
                }

            Text("Version 0.1.0")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private var playbackTab: some View {
        Form {
            Toggle("Pause on battery", isOn: Binding(
                get: { config.pauseOnBattery },
                set: { config.pauseOnBattery = $0; saveConfig() }
            ))

            Toggle("Pause when fullscreen app active", isOn: Binding(
                get: { config.pauseOnFullscreen },
                set: { config.pauseOnFullscreen = $0; saveConfig() }
            ))

            Picker("Video scaling", selection: Binding(
                get: { config.videoGravity },
                set: { config.videoGravity = $0; saveConfig() }
            )) {
                Text("Fill (crop edges)").tag(VideoGravityOption.resizeAspectFill)
                Text("Fit (letterbox)").tag(VideoGravityOption.resizeAspect)
            }
        }
    }

    private var storageTab: some View {
        Form {
            HStack {
                Text("Cache size")
                Spacer()
                Text("\(cacheSize) MB")
                    .foregroundStyle(.secondary)
            }

            Stepper("Cache limit: \(config.cacheSizeLimitMB) MB", value: Binding(
                get: { config.cacheSizeLimitMB },
                set: { config.cacheSizeLimitMB = $0; saveConfig() }
            ), in: 1024...20480, step: 1024)

            Button("Clear Cache") {
                try? CacheManager.shared.clearCache()
                cacheSize = 0
            }
        }
    }

    private var displaysTab: some View {
        Form {
            Text("Per-Display Wallpaper Assignment")
                .font(.headline)

            ForEach(NSScreen.screens, id: \.self) { screen in
                let displayID = DisplayIdentifier.stableID(for: screen)
                let name = screen.localizedName
                HStack {
                    Text(name)
                    Spacer()
                    Text(displayID)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            Text("Assign wallpapers to specific displays from the Library tab.")
                .font(.caption)
                .foregroundStyle(.tertiary)
        }
    }

    private var accountTab: some View {
        Form {
            if let user = appState.currentUser {
                LabeledContent("Email", value: user.email)
                LabeledContent("Role", value: user.role.capitalized)
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
        .sheet(isPresented: $showLogin) {
            LoginView()
        }
    }

    private func saveConfig() {
        try? ConfigManager.shared.update { $0 = config }
    }
}
