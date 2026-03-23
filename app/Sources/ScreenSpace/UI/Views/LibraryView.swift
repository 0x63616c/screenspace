import SwiftUI

struct LibraryView: View {
    var body: some View {
        VStack {
            Spacer()
            Text("Library")
                .font(.title2)
                .foregroundStyle(.secondary)
            Text("Your local wallpapers and playlists")
                .font(.caption)
                .foregroundStyle(.tertiary)
            Spacer()
        }
    }
}
