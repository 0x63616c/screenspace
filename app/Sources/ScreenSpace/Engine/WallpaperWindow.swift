import AppKit
import AVFoundation

final class WallpaperWindow: NSWindow {
    private let playerLayer = AVPlayerLayer()
    private var player: AVQueuePlayer?
    private var looper: AVPlayerLooper?

    init(screen: NSScreen) {
        super.init(
            contentRect: screen.frame,
            styleMask: .borderless,
            backing: .buffered,
            defer: false
        )

        self.level = NSWindow.Level(rawValue: Int(CGWindowLevelForKey(.desktopWindow)))
        self.collectionBehavior = [.canJoinAllSpaces, .stationary, .ignoresCycle]
        self.isOpaque = false
        self.hasShadow = false
        self.ignoresMouseEvents = true
        self.backgroundColor = .clear

        let view = NSView(frame: screen.frame)
        view.wantsLayer = true
        view.layer?.addSublayer(playerLayer)
        playerLayer.frame = view.bounds
        self.contentView = view
    }

    func play(url: URL, gravity: AVLayerVideoGravity = .resizeAspectFill) {
        let asset = AVURLAsset(url: url)
        let item = AVPlayerItem(asset: asset)
        let queuePlayer = AVQueuePlayer(playerItem: item)
        let playerLooper = AVPlayerLooper(player: queuePlayer, templateItem: item)

        self.player = queuePlayer
        self.looper = playerLooper
        self.playerLayer.player = queuePlayer
        self.playerLayer.videoGravity = gravity
        queuePlayer.play()
        self.orderFront(nil)
    }

    func pause() { player?.pause() }
    func resume() { player?.play() }

    func stop() {
        player?.pause()
        player = nil
        looper = nil
        playerLayer.player = nil
        self.orderOut(nil)
    }

    func updateFrame(to screen: NSScreen) {
        self.setFrame(screen.frame, display: true)
        playerLayer.frame = CGRect(origin: .zero, size: screen.frame.size)
    }
}
