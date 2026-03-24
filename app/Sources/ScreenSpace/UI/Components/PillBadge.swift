import SwiftUI

struct PillBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(Typography.meta)
            .padding(.horizontal, Spacing.sm)
            .padding(.vertical, Spacing.xs)
            .background(.quaternary)
            .clipShape(Capsule())
    }
}
