import SwiftUI

struct ResolutionBadge: View {
    let width: Int
    let height: Int

    static func label(for width: Int, height: Int) -> String {
        if width >= 3840 || height >= 2160 { return "4K" }
        if width >= 2560 || height >= 1440 { return "2K" }
        return "1080p"
    }

    private var label: String {
        Self.label(for: width, height: height)
    }

    var body: some View {
        Text(label)
            .font(.system(size: 10, weight: .bold, design: .rounded))
            .foregroundStyle(.white)
            .padding(.horizontal, 6)
            .padding(.vertical, 3)
            .background {
                Capsule()
                    .fill(.ultraThinMaterial)
                    .overlay {
                        Capsule()
                            .strokeBorder(.white.opacity(0.2), lineWidth: 0.5)
                    }
            }
            .accessibilityLabel("Resolution: \(label)")
    }
}
