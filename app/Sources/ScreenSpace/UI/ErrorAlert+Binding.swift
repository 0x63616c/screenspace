import SwiftUI

extension View {
    /// Presents an alert driven by an optional error string.
    /// When the string is non-nil the alert is shown; dismissing sets it back to nil.
    func errorAlert(_ title: String = "Error", message: Binding<String?>) -> some View {
        alert(title, isPresented: Binding(
            get: { message.wrappedValue != nil },
            set: { if !$0 { message.wrappedValue = nil } }
        )) {
            Button("OK") { message.wrappedValue = nil }
        } message: {
            if let text = message.wrappedValue {
                Text(text)
            }
        }
    }
}
