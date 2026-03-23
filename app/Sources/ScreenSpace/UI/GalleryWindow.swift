import AppKit
import SwiftUI

@MainActor
final class GalleryWindowController {
    private var window: NSWindow?

    func show(appState: AppState) {
        if let window = window {
            window.makeKeyAndOrderFront(nil)
            NSApp.activate(ignoringOtherApps: true)
            return
        }

        let contentView = GalleryContentView().environment(appState)
        let hostingView = NSHostingView(rootView: contentView)

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1100, height: 750),
            styleMask: [.titled, .closable, .resizable, .miniaturizable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.center()
        window.title = "ScreenSpace"
        window.titlebarAppearsTransparent = true
        window.titleVisibility = .hidden
        window.contentView = hostingView
        window.makeKeyAndOrderFront(nil)
        window.isReleasedWhenClosed = false
        window.minSize = NSSize(width: 900, height: 600)
        NSApp.activate(ignoringOtherApps: true)

        self.window = window
    }
}

enum GallerySection: String, CaseIterable, Identifiable {
    case home = "Home"
    case explore = "Explore"
    case library = "Library"
    case admin = "Admin"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .home: return "house"
        case .explore: return "safari"
        case .library: return "square.stack"
        case .admin: return "shield"
        }
    }
}

struct GalleryContentView: View {
    @State private var selectedSection: GallerySection? = .home
    @State private var showSettings = false
    @State private var showUpload = false

    var body: some View {
        NavigationSplitView {
            sidebar
        } detail: {
            detailView
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .navigationSplitViewStyle(.balanced)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button(action: { showUpload = true }) {
                    Label("Upload", systemImage: "arrow.up.circle")
                }
            }
            ToolbarItem(placement: .primaryAction) {
                Button(action: { showSettings = true }) {
                    Label("Settings", systemImage: "gearshape")
                }
            }
        }
        .sheet(isPresented: $showSettings) {
            SettingsView()
        }
        .sheet(isPresented: $showUpload) {
            UploadView()
        }
    }

    private var sidebar: some View {
        List(selection: $selectedSection) {
            Section("Browse") {
                Label("Home", systemImage: "house")
                    .tag(GallerySection.home)
                Label("Explore", systemImage: "safari")
                    .tag(GallerySection.explore)
            }

            Section("Your Stuff") {
                Label("Library", systemImage: "square.stack")
                    .tag(GallerySection.library)
            }

            Section("Manage") {
                Label("Admin", systemImage: "shield")
                    .tag(GallerySection.admin)
            }
        }
        .listStyle(.sidebar)
        .navigationSplitViewColumnWidth(min: 180, ideal: 200, max: 240)
    }

    @ViewBuilder
    private var detailView: some View {
        Group {
            switch selectedSection {
            case .home:
                HomeView()
            case .explore:
                ExploreView()
            case .library:
                LibraryView()
            case .admin:
                AdminView()
            case .none:
                HomeView()
            }
        }
        .scrollContentBackground(.hidden)
        .background(Color(nsColor: .windowBackgroundColor))
    }
}
