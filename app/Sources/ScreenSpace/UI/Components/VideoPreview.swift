import AVKit
import SwiftUI

struct VideoPreview: NSViewRepresentable {
    let url: URL

    func makeCoordinator() -> Coordinator {
        Coordinator()
    }

    func makeNSView(context: Context) -> AVPlayerView {
        let view = AVPlayerView()
        view.controlsStyle = .inline
        view.showsFullScreenToggleButton = false
        let player = AVPlayer(url: url)
        player.isMuted = true
        view.player = player
        player.play()
        let observer = NotificationCenter.default.addObserver(
            forName: .AVPlayerItemDidPlayToEndTime,
            object: player.currentItem,
            queue: .main
        ) { _ in
            player.seek(to: .zero)
            player.play()
        }
        context.coordinator.observer = observer
        return view
    }

    func updateNSView(_ nsView: AVPlayerView, context: Context) {
        guard let currentAsset = nsView.player?.currentItem?.asset as? AVURLAsset,
              currentAsset.url != url else { return }
        let player = AVPlayer(url: url)
        player.isMuted = true
        nsView.player = player
        player.play()
    }

    static func dismantleNSView(_ nsView: AVPlayerView, coordinator: Coordinator) {
        nsView.player?.pause()
        if let observer = coordinator.observer {
            NotificationCenter.default.removeObserver(observer)
        }
    }

    class Coordinator {
        var observer: NSObjectProtocol?
    }
}
