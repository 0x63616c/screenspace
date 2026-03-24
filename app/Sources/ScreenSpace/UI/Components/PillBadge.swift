import SwiftUI

struct PillBadge: View {
    let text: String

    var body: some View {
        Text(text)
            .font(Typography.meta)
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(.quaternary)
            .clipShape(Capsule())
    }
}
