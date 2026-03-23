import SwiftUI

struct LoginView: View {
    @Environment(AppState.self) var appState
    @Environment(\.dismiss) var dismiss
    @State private var email = ""
    @State private var password = ""
    @State private var isRegistering = false
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: Spacing.lg) {
            Text(isRegistering ? "Create Account" : "Log In")
                .font(.title2).fontWeight(.bold)

            TextField("Email", text: $email)
                .textFieldStyle(.roundedBorder)

            SecureField("Password", text: $password)
                .textFieldStyle(.roundedBorder)

            if let error = errorMessage {
                Text(error)
                    .foregroundStyle(.red)
                    .font(.caption)
            }

            HStack {
                Button("Cancel") { dismiss() }
                    .buttonStyle(.bordered)

                Button(isRegistering ? "Create Account" : "Log In") {
                    Task { await submit() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(email.isEmpty || password.isEmpty || isLoading)
            }

            Button(isRegistering ? "Already have an account? Log in" : "Don't have an account? Create one") {
                isRegistering.toggle()
                errorMessage = nil
            }
            .buttonStyle(.plain)
            .font(.caption)
        }
        .padding()
        .frame(width: 350)
    }

    private func submit() async {
        isLoading = true
        errorMessage = nil
        do {
            if isRegistering {
                try await appState.register(email: email, password: password)
            } else {
                try await appState.login(email: email, password: password)
            }
            dismiss()
        } catch {
            errorMessage = error.localizedDescription
        }
        isLoading = false
    }
}
