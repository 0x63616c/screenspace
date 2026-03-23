import SwiftUI

/// Glass card with proper macOS vibrancy.
/// On macOS 15: uses ultra thin material with gradient border for depth.
/// On macOS 26: will use .glassEffect() for Liquid Glass.
struct GlassCard<Content: View>: View {
    let cornerRadius: CGFloat
    let content: Content

    init(cornerRadius: CGFloat = 16, @ViewBuilder content: () -> Content) {
        self.cornerRadius = cornerRadius
        self.content = content()
    }

    var body: some View {
        content
            .background {
                ZStack {
                    RoundedRectangle(cornerRadius: cornerRadius)
                        .fill(.ultraThinMaterial)

                    // Subtle gradient border for glass edge effect
                    RoundedRectangle(cornerRadius: cornerRadius)
                        .strokeBorder(
                            LinearGradient(
                                colors: [
                                    .white.opacity(0.3),
                                    .white.opacity(0.05),
                                ],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            lineWidth: 0.5
                        )
                }
            }
            .clipShape(RoundedRectangle(cornerRadius: cornerRadius))
            .shadow(color: .black.opacity(0.1), radius: 8, x: 0, y: 4)
    }
}