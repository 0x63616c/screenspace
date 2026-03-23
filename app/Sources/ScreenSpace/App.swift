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

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
        if let button = statusItem?.button {
            button.image = NSImage(
                systemSymbolName: "photo.on.rectangle",
                accessibilityDescription: "ScreenSpace"
            )
        }

        let menu = NSMenu()
        menu.addItem(
            NSMenuItem(
                title: "Open ScreenSpace",
                action: #selector(openGallery),
                keyEquivalent: "o"
            )
        )
        menu.addItem(NSMenuItem.separator())
        menu.addItem(
            NSMenuItem(
                title: "Quit",
                action: #selector(NSApplication.terminate(_:)),
                keyEquivalent: "q"
            )
        )
        statusItem?.menu = menu

        galleryController.show()
    }

    @objc func openGallery() {
        galleryController.show()
    }
}
