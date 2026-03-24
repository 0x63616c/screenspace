import AppKit
import IOKit.ps

protocol PowerSourceProvider: Sendable {
    var isOnBattery: Bool { get }
}

protocol LockStateProvider: AnyObject, Sendable {
    var isLocked: Bool { get }
}

struct SystemPowerSource: PowerSourceProvider {
    var isOnBattery: Bool {
        guard let snapshot = IOPSCopyPowerSourcesInfo()?.takeRetainedValue(),
              let source = IOPSGetProvidingPowerSourceType(snapshot)?.takeRetainedValue() as? String else {
            return false
        }
        return source == kIOPSBatteryPowerValue as String
    }
}

final class SystemLockState: LockStateProvider, @unchecked Sendable {
    private let _isLocked = NSLock()
    private var _locked = false

    var isLocked: Bool {
        _isLocked.lock()
        defer { _isLocked.unlock() }
        return _locked
    }

    init() {
        DistributedNotificationCenter.default().addObserver(
            self, selector: #selector(screenLocked),
            name: NSNotification.Name("com.apple.screenIsLocked"), object: nil
        )
        DistributedNotificationCenter.default().addObserver(
            self, selector: #selector(screenUnlocked),
            name: NSNotification.Name("com.apple.screenIsUnlocked"), object: nil
        )
    }

    @objc private func screenLocked() {
        _isLocked.lock()
        _locked = true
        _isLocked.unlock()
    }

    @objc private func screenUnlocked() {
        _isLocked.lock()
        _locked = false
        _isLocked.unlock()
    }

    deinit { DistributedNotificationCenter.default().removeObserver(self) }
}

@Observable
@MainActor
final class PauseController {
    private(set) var shouldPause: Bool = false

    private var config: AppConfig
    private let powerSource: PowerSourceProvider
    private let lockState: LockStateProvider
    private var isSleeping: Bool = false
    private var observationTask: Task<Void, Never>?

    init(
        config: AppConfig,
        powerSource: PowerSourceProvider = SystemPowerSource(),
        lockState: LockStateProvider = SystemLockState()
    ) {
        self.config = config
        self.powerSource = powerSource
        self.lockState = lockState
        startObserving()
        evaluate()
    }

    func updateConfig(_ config: AppConfig) {
        self.config = config
        evaluate()
    }

    func evaluate() {
        var pause = false
        if config.pauseOnBattery && powerSource.isOnBattery { pause = true }
        if ProcessInfo.processInfo.isLowPowerModeEnabled { pause = true }
        if lockState.isLocked { pause = true }
        if isSleeping { pause = true }
        shouldPause = pause
    }

    private func startObserving() {
        observationTask = Task { [weak self] in
            await withTaskGroup(of: Void.self) { group in
                group.addTask { await self?.observeSleep() }
                group.addTask { await self?.observeWake() }
                group.addTask { await self?.observePowerState() }
                group.addTask { await self?.observeScreenLocked() }
                group.addTask { await self?.observeScreenUnlocked() }
                group.addTask { await self?.observePowerSourcePeriodically() }
            }
        }
    }

    private func observeSleep() async {
        let notifications = NSWorkspace.shared.notificationCenter.notifications(
            named: NSWorkspace.willSleepNotification
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            isSleeping = true
            evaluate()
        }
    }

    private func observeWake() async {
        let notifications = NSWorkspace.shared.notificationCenter.notifications(
            named: NSWorkspace.didWakeNotification
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            isSleeping = false
            evaluate()
        }
    }

    private func observePowerState() async {
        let notifications = NotificationCenter.default.notifications(
            named: NSNotification.Name.NSProcessInfoPowerStateDidChange
        )
        for await _ in notifications {
            guard !Task.isCancelled else { return }
            evaluate()
        }
    }

    private func observeScreenLocked() async {
        // DistributedNotificationCenter does not support notifications(named:) async sequence.
        // Use addObserver with a Task callback instead.
        await withCheckedContinuation { (continuation: CheckedContinuation<Void, Never>) in
            DistributedNotificationCenter.default().addObserver(
                forName: NSNotification.Name("com.apple.screenIsLocked"),
                object: nil,
                queue: .main
            ) { [weak self] _ in
                Task { @MainActor in
                    self?.evaluate()
                }
            }
            continuation.resume()
        }
        // Keep task alive until cancelled
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(86400))
        }
    }

    private func observeScreenUnlocked() async {
        await withCheckedContinuation { (continuation: CheckedContinuation<Void, Never>) in
            DistributedNotificationCenter.default().addObserver(
                forName: NSNotification.Name("com.apple.screenIsUnlocked"),
                object: nil,
                queue: .main
            ) { [weak self] _ in
                Task { @MainActor in
                    try? await Task.sleep(for: .milliseconds(100))
                    self?.evaluate()
                }
            }
            continuation.resume()
        }
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(86400))
        }
    }

    private func observePowerSourcePeriodically() async {
        while !Task.isCancelled {
            try? await Task.sleep(for: .seconds(30))
            guard !Task.isCancelled else { return }
            evaluate()
        }
    }

    deinit {
        observationTask?.cancel()
    }
}
