import Foundation
import AppKit
import Combine
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

@MainActor
final class PauseController: ObservableObject {
    @Published private(set) var shouldPause: Bool = false

    private var config: AppConfig
    private let powerSource: PowerSourceProvider
    private let lockState: LockStateProvider
    private var isSleeping: Bool = false
    private var cancellables = Set<AnyCancellable>()
    nonisolated(unsafe) private var timer: Timer?

    init(
        config: AppConfig,
        powerSource: PowerSourceProvider = SystemPowerSource(),
        lockState: LockStateProvider = SystemLockState()
    ) {
        self.config = config
        self.powerSource = powerSource
        self.lockState = lockState
        observeSystemState()
        evaluate()
    }

    func updateConfig(_ config: AppConfig) {
        self.config = config
        evaluate()
    }

    func evaluate() {
        var pause = false

        if config.pauseOnBattery && powerSource.isOnBattery {
            pause = true
        }

        if ProcessInfo.processInfo.isLowPowerModeEnabled {
            pause = true
        }

        if lockState.isLocked {
            pause = true
        }

        if isSleeping {
            pause = true
        }

        shouldPause = pause
    }

    private func observeSystemState() {
        let workspace = NSWorkspace.shared.notificationCenter

        workspace.publisher(for: NSWorkspace.willSleepNotification)
            .receive(on: RunLoop.main)
            .sink { [weak self] _ in
                self?.isSleeping = true
                self?.evaluate()
            }
            .store(in: &cancellables)

        workspace.publisher(for: NSWorkspace.didWakeNotification)
            .receive(on: RunLoop.main)
            .sink { [weak self] _ in
                self?.isSleeping = false
                self?.evaluate()
            }
            .store(in: &cancellables)

        NotificationCenter.default.publisher(for: NSNotification.Name.NSProcessInfoPowerStateDidChange)
            .receive(on: RunLoop.main)
            .sink { [weak self] _ in
                self?.evaluate()
            }
            .store(in: &cancellables)

        DistributedNotificationCenter.default().publisher(
            for: NSNotification.Name("com.apple.screenIsLocked")
        )
        .receive(on: RunLoop.main)
        .sink { [weak self] _ in
            self?.evaluate()
        }
        .store(in: &cancellables)

        DistributedNotificationCenter.default().publisher(
            for: NSNotification.Name("com.apple.screenIsUnlocked")
        )
        .receive(on: RunLoop.main)
        .sink { [weak self] _ in
            // Small delay to let lock state provider update
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
                self?.evaluate()
            }
        }
        .store(in: &cancellables)

        // Periodic check for power source changes (no system notification for this)
        timer = Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                self?.evaluate()
            }
        }
    }

    deinit {
        let t = timer
        t?.invalidate()
    }
}
