import Testing
@testable import ScreenSpace

@MainActor
final class MockPowerSource: @preconcurrency PowerSourceProvider {
    var isOnBattery = false
}

@MainActor
final class MockLockState: @preconcurrency LockStateProvider {
    var isLocked = false
}

@MainActor
struct PauseControllerTests {
    private func makeController(
        config: AppConfig = .default,
        powerSource: MockPowerSource = MockPowerSource(),
        lockState: MockLockState = MockLockState()
    ) -> PauseController {
        PauseController(config: config, powerSource: powerSource, lockState: lockState)
    }

    @Test("pauses when on battery and pauseOnBattery is enabled")
    func pausesOnBattery() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("does not pause on AC power")
    func doesNotPauseOnAC() {
        let power = MockPowerSource()
        power.isOnBattery = false
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(!controller.shouldPause)
    }

    @Test("does not pause when pauseOnBattery is disabled")
    func disabledPauseOnBattery() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = false
        let controller = makeController(config: config, powerSource: power)
        controller.evaluate()
        #expect(!controller.shouldPause)
    }

    @Test("pauses when screen is locked")
    func pausesWhenLocked() {
        let lock = MockLockState()
        lock.isLocked = true
        let controller = makeController(lockState: lock)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("pauses when multiple conditions are true")
    func multipleConditions() {
        let power = MockPowerSource()
        power.isOnBattery = true
        let lock = MockLockState()
        lock.isLocked = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power, lockState: lock)
        controller.evaluate()
        #expect(controller.shouldPause)
    }

    @Test("updateConfig re-evaluates pause state")
    func configUpdate() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let controller = makeController(config: config, powerSource: power)
        #expect(controller.shouldPause)

        var newConfig = config
        newConfig.pauseOnBattery = false
        controller.updateConfig(newConfig)
        #expect(!controller.shouldPause)
    }
}
