import AVFoundation

@MainActor
final class LivePlayer: PlayerProviding {
    private let player = AVQueuePlayer()

    nonisolated func play(url: URL) {
        Task { @MainActor in
            let item = AVPlayerItem(url: url)
            player.replaceCurrentItem(with: item)
            player.play()
        }
    }

    nonisolated func pause() {
        Task { @MainActor in player.pause() }
    }

    nonisolated func resume() {
        Task { @MainActor in player.play() }
    }

    nonisolated func seek(to time: Double) {
        Task { @MainActor in
            let cmTime = CMTime(seconds: time, preferredTimescale: 600)
            player.seek(to: cmTime)
        }
    }

    nonisolated func stop() {
        Task { @MainActor in
            player.pause()
            player.replaceCurrentItem(with: nil)
        }
    }
}
