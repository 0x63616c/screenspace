import Testing
@testable import ScreenSpace

@MainActor
struct LoginViewModelTests {
    @Test("canSubmit is false with empty email")
    func canSubmitFalseEmptyEmail() {
        let vm = LoginViewModel(api: MockAPI(), eventLog: MockEventLog())
        vm.password = "password123"

        #expect(vm.canSubmit == false)
    }

    @Test("canSubmit is false with short password")
    func canSubmitFalseShortPassword() {
        let vm = LoginViewModel(api: MockAPI(), eventLog: MockEventLog())
        vm.email = "test@example.com"
        vm.password = "short"

        #expect(vm.canSubmit == false)
    }

    @Test("canSubmit is true with valid credentials")
    func canSubmitTrue() {
        let vm = LoginViewModel(api: MockAPI(), eventLog: MockEventLog())
        vm.email = "test@example.com"
        vm.password = "validpassword"

        #expect(vm.canSubmit == true)
    }

    @Test("successful login calls onSuccess")
    func successfulLoginCallsOnSuccess() async {
        let api = MockAPI()
        api.loginResponse = .success(AuthToken(token: "jwt", role: .user))
        var onSuccessCalled = false
        let vm = LoginViewModel(api: api, eventLog: MockEventLog())
        vm.email = "test@example.com"
        vm.password = "validpassword"
        vm.onSuccess = { onSuccessCalled = true }

        await vm.submit()

        #expect(onSuccessCalled == true)
        #expect(vm.errorMessage == nil)
        #expect(vm.isLoading == false)
    }

    @Test("failed login sets errorMessage")
    func failedLoginSetsError() async {
        let api = MockAPI()
        api.loginResponse = .failure(APIError.httpError(status: 401))
        let vm = LoginViewModel(api: api, eventLog: MockEventLog())
        vm.email = "test@example.com"
        vm.password = "wrongpassword"

        await vm.submit()

        #expect(vm.errorMessage != nil)
        #expect(vm.isLoading == false)
    }

    @Test("toggleMode flips isRegistering and clears error")
    func toggleModeFlips() {
        let vm = LoginViewModel(api: MockAPI(), eventLog: MockEventLog())
        vm.errorMessage = "Some error"
        #expect(vm.isRegistering == false)

        vm.toggleMode()

        #expect(vm.isRegistering == true)
        #expect(vm.errorMessage == nil)
    }

    @Test("logs registered event on successful registration")
    func logsRegisteredEvent() async {
        let api = MockAPI()
        api.registerResponse = .success(AuthToken(token: "jwt", role: .user))
        let log = MockEventLog()
        let vm = LoginViewModel(api: api, eventLog: log)
        vm.email = "new@example.com"
        vm.password = "password123"
        vm.isRegistering = true

        await vm.submit()

        #expect(log.events.contains { $0.event == "registered" })
    }
}
