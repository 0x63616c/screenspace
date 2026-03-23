import Foundation

/// Manages app auto-updates via Sparkle framework.
/// Sparkle integration requires adding the package dependency:
/// .package(url: "https://github.com/sparkle-project/Sparkle", from: "2.0.0")
///
/// Once added, uncomment the Sparkle import and implementation below.
@MainActor
final class UpdateManager {
    static let shared = UpdateManager()

    /// Check for updates. Call on app launch and from menu bar.
    func checkForUpdates() {
        // TODO: Wire Sparkle SPUStandardUpdaterController
        // import Sparkle
        // private let updaterController = SPUStandardUpdaterController(startingUpdater: true, updaterDelegate: nil, userDriverDelegate: nil)
        // updaterController.checkForUpdates(nil)
    }

    /// Whether automatic update checks are enabled.
    var automaticChecksEnabled: Bool {
        get { true }
        set { /* Wire to Sparkle */ }
    }
}
