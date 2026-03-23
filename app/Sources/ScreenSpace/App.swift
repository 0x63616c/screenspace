import SwiftUI
import AppKit

@main
struct ScreenSpaceApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        Settings {
            Text("ScreenSpace Settings")
        }
    }
}

@MainActor
class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem?
    private let galleryController = GalleryWindowController()
    private let engine = WallpaperEngine()
    private var isPaused = false

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        if let button = statusItem?.button {
            button.image = NSImage(
                systemSymbolName: "photo.on.rectangle",
                accessibilityDescription: "ScreenSpace"
            )
        }

        statusItem?.menu = buildMenu()

        engine.start()
        galleryController.show()
    }

    private func buildMenu() -> NSMenu {
        let menu = NSMenu()

        // Now playing
        let nowPlaying = NSMenuItem(title: "No wallpaper active", action: nil, keyEquivalent: "")
        nowPlaying.isEnabled = false
        menu.addItem(nowPlaying)

        menu.addItem(NSMenuItem.separator())

        // Playback controls
        let playPause = NSMenuItem(title: "Pause", action: #selector(togglePlayPause), keyEquivalent: "p")
        playPause.keyEquivalentModifierMask = [.control, .option]
        menu.addItem(playPause)

        let skip = NSMenuItem(title: "Next Wallpaper", action: #selector(skipToNext), keyEquivalent: "")
        menu.addItem(skip)

        menu.addItem(NSMenuItem.separator())

        // App controls
        menu.addItem(NSMenuItem(title: "Open ScreenSpace", action: #selector(openGallery), keyEquivalent: "o"))
        menu.addItem(NSMenuItem(title: "Check for Updates", action: #selector(checkForUpdates), keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit ScreenSpace", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))

        return menu
    }

    @objc func openGallery() {
        galleryController.show()
    }

    @objc func togglePlayPause() {
        if isPaused {
            engine.resumeAll()
        } else {
            engine.pauseAll()
        }
        isPaused.toggle()

        // Update menu item title
        if let menu = statusItem?.menu,
           let playPauseItem = menu.items.first(where: { $0.action == #selector(togglePlayPause) }) {
            playPauseItem.title = isPaused ? "Resume" : "Pause"
        }
    }

    @objc func skipToNext() {
        // TODO: Wire to playlist manager for next wallpaper rotation
    }

    @objc func checkForUpdates() {
        UpdateManager.shared.checkForUpdates()
    }
}
