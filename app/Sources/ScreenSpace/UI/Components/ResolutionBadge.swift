import SwiftUI

struct ResolutionBadge: View {
    let width: Int
    let height: Int

    private var label: String {
        if width >= 3840 || height >= 2160 { return "4K" }
        if width >= 2560 || height >= 1440 { return "2K" }
        return "1080p"
    }

    var body: some View {
        Text(label)
            .font(.caption2)
            .fontWeight(.bold)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(.ultraThinMaterial)
            .cornerRadius(4)
    }
}
