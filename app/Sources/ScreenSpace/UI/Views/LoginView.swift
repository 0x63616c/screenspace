import SwiftUI

struct LoginView: View {
    @Environment(AppState.self) var appState
    @Environment(\.dismiss) var dismiss
    @State private var viewModel: LoginViewModel?

    var body: some View {
        Group {
            if let viewModel {
                LoginContentView(viewModel: viewModel, dismiss: dismiss)
            } else {
                ProgressView()
            }
        }
        .frame(width: 350)
        .padding()
        .task {
            if viewModel == nil {
                let vm = LoginViewModel(api: appState.apiService, eventLog: appState.eventLog)
                vm.onSuccess = { [weak appState] in
                    Task { @MainActor in
                        appState?.currentUser = try? await appState?.apiService.me()
                    }
                    dismiss()
                }
                viewModel = vm
            }
        }
    }
}

private struct LoginContentView: View {
    @Bindable var viewModel: LoginViewModel
    let dismiss: DismissAction

    var body: some View {
        VStack(spacing: Spacing.lg) {
            Text(viewModel.isRegistering ? "Create Account" : "Log In")
                .font(Typography.pageTitle)

            TextField("Email", text: $viewModel.email)
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel("Email address")

            SecureField("Password", text: $viewModel.password)
                .textFieldStyle(.roundedBorder)
                .accessibilityLabel("Password")
                .accessibilityHint("Minimum 8 characters")

            if let error = viewModel.errorMessage {
                Text(error)
                    .foregroundStyle(.red)
                    .font(Typography.meta)
            }

            HStack {
                Button("Cancel") { dismiss() }
                    .buttonStyle(.bordered)

                Button(viewModel.isRegistering ? "Create Account" : "Log In") {
                    Task { await viewModel.submit() }
                }
                .buttonStyle(.borderedProminent)
                .disabled(!viewModel.canSubmit)
                .accessibilityLabel(viewModel.isRegistering ? "Create account" : "Log in")
            }

            Button(viewModel.isRegistering ? "Already have an account? Log in" : "Don't have an account? Create one") {
                viewModel.toggleMode()
            }
            .buttonStyle(.plain)
            .font(Typography.meta)
            .accessibilityAddTraits([.isButton, .isLink])
        }
    }
}
