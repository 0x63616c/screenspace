import SwiftUI

struct ExploreView: View {
    var body: some View {
        VStack {
            Spacer()
            Text("Explore")
                .font(.title2)
                .foregroundStyle(.secondary)
            Text("Browse wallpapers by category")
                .font(.caption)
                .foregroundStyle(.tertiary)
            Spacer()
        }
    }
}
