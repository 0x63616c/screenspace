import AppKit
import SwiftUI

@MainActor
final class GalleryWindowController {
    private var window: NSWindow?

    func show() {
        if let window = window {
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        let contentView = GalleryContentView()
        let hostingView = NSHostingView(rootView: contentView)

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1000, height: 700),
            styleMask: [.titled, .closable, .resizable, .miniaturizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.center()
        window.title = "ScreenSpace"
        window.titlebarAppearsTransparent = true
        window.contentView = hostingView
        window.makeKeyAndOrderFront(nil)
        window.isReleasedWhenClosed = false
        NSApp.activate(ignoringOtherApps: true)

        self.window = window
    }
}

enum GalleryTab: String, CaseIterable {
    case home = "Home"
    case explore = "Explore"
    case library = "Library"
    case admin = "Admin"
}

struct GalleryContentView: View {
    @State private var selectedTab: GalleryTab = .home
    @State private var showSettings = false

    var body: some View {
        VStack(spacing: 0) {
            HStack {
                Text("ScreenSpace")
                    .font(.title2)
                    .fontWeight(.bold)

                Spacer()

                Picker("", selection: $selectedTab) {
                    ForEach(GalleryTab.allCases, id: \.self) { tab in
                        Text(tab.rawValue).tag(tab)
                    }
                }
                .pickerStyle(.segmented)
                .frame(width: 320)

                Spacer()

                Button("Upload") { }
                    .buttonStyle(.bordered)

                Button(action: { showSettings = true }) {
                    Image(systemName: "gearshape")
                }
            }
            .padding()

            Divider()

            switch selectedTab {
            case .home:
                HomeView()
            case .explore:
                ExploreView()
            case .library:
                LibraryView()
            case .admin:
                AdminView()
            }
        }
        .frame(minWidth: 800, minHeight: 600)
        .sheet(isPresented: $showSettings) {
            SettingsView()
        }
    }
}
