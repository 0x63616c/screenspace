import AppKit
import CoreGraphics

enum DisplayIdentifier {
    static func stableID(for screen: NSScreen) -> String {
        guard let screenNumber = screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID else {
            return "unknown"
        }
        let vendor = CGDisplayVendorNumber(screenNumber)
        let model = CGDisplayModelNumber(screenNumber)
        let serial = CGDisplaySerialNumber(screenNumber)

        if serial != 0 {
            return "\(vendor)-\(model)-\(serial)"
        }
        return "fallback-\(screenNumber)"
    }

    static func directDisplayID(for screen: NSScreen) -> CGDirectDisplayID? {
        screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? CGDirectDisplayID
    }
}
