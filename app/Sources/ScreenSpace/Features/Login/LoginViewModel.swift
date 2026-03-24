import Foundation

@Observable
@MainActor
final class LoginViewModel {
    private let api: APIProviding
    private let eventLog: EventLogging

    var email = ""
    var password = ""
    var isRegistering = false
    var isLoading = false
    var errorMessage: String?

    var canSubmit: Bool {
        !email.trimmingCharacters(in: .whitespaces).isEmpty
            && password.count >= 8
            && !isLoading
    }

    var onSuccess: (() -> Void)?

    init(api: APIProviding, eventLog: EventLogging) {
        self.api = api
        self.eventLog = eventLog
    }

    func submit() async {
        isLoading = true
        errorMessage = nil
        do {
            if isRegistering {
                _ = try await api.register(email: email, password: password)
                eventLog.log("registered", data: [:])
            } else {
                _ = try await api.login(email: email, password: password)
                eventLog.log("logged_in", data: [:])
            }
            onSuccess?()
        } catch {
            errorMessage = error.localizedDescription
            eventLog.log("error", data: ["context": "login", "message": error.localizedDescription])
        }
        isLoading = false
    }

    func toggleMode() {
        isRegistering.toggle()
        errorMessage = nil
    }
}
