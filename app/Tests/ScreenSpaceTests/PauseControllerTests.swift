import XCTest
@testable import ScreenSpace

final class MockPowerSource: PowerSourceProvider, @unchecked Sendable {
    var isOnBattery = false
}

final class MockLockState: LockStateProvider, @unchecked Sendable {
    var isLocked = false
}

@MainActor
final class PauseControllerTests: XCTestCase {
    private func makeController(
        config: AppConfig = .default,
        powerSource: MockPowerSource = MockPowerSource(),
        lockState: MockLockState = MockLockState()
    ) -> (PauseController, MockPowerSource, MockLockState) {
        let controller = PauseController(
            config: config,
            powerSource: powerSource,
            lockState: lockState
        )
        return (controller, powerSource, lockState)
    }

    func testShouldPauseOnBattery() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let (controller, _, _) = makeController(config: config, powerSource: power)
        controller.evaluate()
        XCTAssertTrue(controller.shouldPause)
    }

    func testShouldNotPauseOnAC() {
        let power = MockPowerSource()
        power.isOnBattery = false
        var config = AppConfig.default
        config.pauseOnBattery = true
        let (controller, _, _) = makeController(config: config, powerSource: power)
        controller.evaluate()
        XCTAssertFalse(controller.shouldPause)
    }

    func testShouldNotPauseWhenDisabled() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = false
        let (controller, _, _) = makeController(config: config, powerSource: power)
        controller.evaluate()
        XCTAssertFalse(controller.shouldPause)
    }

    func testShouldPauseWhenLocked() {
        let lock = MockLockState()
        lock.isLocked = true
        let (controller, _, _) = makeController(lockState: lock)
        controller.evaluate()
        XCTAssertTrue(controller.shouldPause)
    }

    func testMultipleConditions() {
        let power = MockPowerSource()
        power.isOnBattery = true
        let lock = MockLockState()
        lock.isLocked = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let (controller, _, _) = makeController(config: config, powerSource: power, lockState: lock)
        controller.evaluate()
        XCTAssertTrue(controller.shouldPause)
    }

    func testConfigUpdate() {
        let power = MockPowerSource()
        power.isOnBattery = true
        var config = AppConfig.default
        config.pauseOnBattery = true
        let (controller, _, _) = makeController(config: config, powerSource: power)
        XCTAssertTrue(controller.shouldPause)

        var newConfig = config
        newConfig.pauseOnBattery = false
        controller.updateConfig(newConfig)
        XCTAssertFalse(controller.shouldPause)
    }
}
